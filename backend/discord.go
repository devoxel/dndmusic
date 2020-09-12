package main

import (
	"errors"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type discordSession interface {
	ChannelMessageSend(channelID string, content string) (*discordgo.Message, error)
}

type DiscordServer struct {
	sessions *Sessions
}

func (s *DiscordServer) sendErrorMsg(ds *discordgo.Session, m *discordgo.MessageCreate, err error) {
	log.Printf("sending err: %v", err)
	_, sErr := ds.ChannelMessageSend(m.ChannelID, err.Error())
	if sErr != nil {
		log.Printf("cannot send error message: %v", sErr)
	}
}

func (s *DiscordServer) handleCreate(ds *discordgo.Session, m *discordgo.MessageCreate, pw string) {
	log.Println("confirm")
	if err := s.sessions.Confirm(pw, m); err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}

	s.sendMessage(ds, m.ChannelID, "Sweet! You're good to go.")
}

func (s *DiscordServer) handleTestGroovy(ds *discordgo.Session, m *discordgo.MessageCreate) {
	s.sendMessage(ds, m.ChannelID, "-p https://open.spotify.com/playlist/3uJFVs1EUBA6jKqWhn9FA1")
}

func (s *DiscordServer) handleDocs(ds *discordgo.Session, m *discordgo.MessageCreate) {
	_, err := ds.ChannelMessageSend(m.ChannelID, "you're using the bot wrong.")
	if err != nil {
		log.Printf("error sending message: %v", err)
		log.Printf("channel.id = %v", m.ChannelID)
		log.Printf("m = %v", m)
	}
}

func (s *DiscordServer) incomingMessage(ds *discordgo.Session, m *discordgo.MessageCreate) {
	const prefix = "🙂"

	if !strings.HasPrefix(m.Content, prefix) {
		return
	}

	cmd := strings.Fields(strings.TrimPrefix(m.Content, prefix))
	for i, c := range cmd {
		cmd[i] = strings.ToLower(strings.TrimSpace(c))
	}

	if len(cmd) == 0 {
		s.handleDocs(ds, m)
		return
	}

	switch cmd[0] {
	case "create":
		log.Println("create")
		if len(cmd) < 2 {
			s.sendErrorMsg(ds, m, errors.New("you need to enter a password dingus"))
			return
		}
		s.handleCreate(ds, m, cmd[1])
	case "testgroovy":
		s.handleTestGroovy(ds, m)
	default:
		s.handleDocs(ds, m)
	}

}

func (s *DiscordServer) sendMessage(ds *discordgo.Session, id, message string) {
	m, err := ds.ChannelMessageSend(id, message)
	if err != nil {
		log.Printf("error sending message: %v", err)
		log.Printf("channel.id = %v", id)
		log.Printf("m = %v", m)
	}
}
