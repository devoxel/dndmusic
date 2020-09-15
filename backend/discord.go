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

// Have to wrap the function to allow us to use an interface
func (s *DiscordServer) incomingMessage(ds *discordgo.Session, m *discordgo.MessageCreate) {
	s.handleMessage(ds, m)
}

func (s *DiscordServer) sendErrorMsg(ds discordSession, m *discordgo.MessageCreate, err error) {
	log.Printf("sending err: %v", err)
	_, sErr := ds.ChannelMessageSend(m.ChannelID, err.Error())
	if sErr != nil {
		log.Printf("cannot send error message: %v", sErr)
	}
}

func (s *DiscordServer) getSenderCID(ds *discordgo.Session, m *discordgo.MessageCreate) (string, error) {
	// Find the guild for that channel.
	g, err := ds.State.Guild(m.GuildID)
	if err != nil {
		// Could not find guild.
		return "", err
	}

	// Look for the message sender in that guild's current voice states.
	for _, vs := range g.VoiceStates {
		if vs.UserID == m.Author.ID {
			return vs.ChannelID, nil
		}
	}
	return "", errors.New("You can't create a session if you're not in a voice channel")
}

func (s *DiscordServer) handleCreate(ds *discordgo.Session, m *discordgo.MessageCreate, pw string) {
	log.Println("confirm")

	sendMsg := func(msg string) error {
		_, err := ds.ChannelMessageSend(m.ChannelID, msg)
		return err
	}

	joinVoice := func() (*discordgo.VoiceConnection, error) {
		// get channel sender is in
		cid, err := s.getSenderCID(ds, m)
		if err != nil {
			return nil, err
		}
		return ds.ChannelVoiceJoin(m.GuildID, cid, false, true)
	}

	if err := s.sessions.Confirm(pw, m, sendMsg, joinVoice); err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}

	s.sendMessage(ds, m.ChannelID, "Sweet! You're good to go.")
}

func (s *DiscordServer) handleDocs(ds discordSession, m *discordgo.MessageCreate) {
	_, err := ds.ChannelMessageSend(m.ChannelID, "you're using the bot wrong.")
	if err != nil {
		log.Printf("error sending message: %v", err)
		log.Printf("channel.id = %v", m.ChannelID)
		log.Printf("m = %v", m)
	}
}

func (s *DiscordServer) handleMessage(ds *discordgo.Session, m *discordgo.MessageCreate) {
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
	default:
		s.handleDocs(ds, m)
	}
}

func (s *DiscordServer) sendMessage(ds discordSession, id, message string) {
	m, err := ds.ChannelMessageSend(id, message)
	if err != nil {
		log.Printf("error sending message: %v", err)
		log.Printf("channel.id = %v", id)
		log.Printf("m = %v", m)
	}
}
