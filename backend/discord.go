package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type DiscordServer struct {
	sessions *Sessions
}

func (s *DiscordServer) handleCreate(ds *discordgo.Session, m *discordgo.MessageCreate, pw string) {
	if err := s.sessions.Confirm(pw, m); err != nil {
		sendErrorMsg(ds, m.ChannelID, err)
		return
	}

	s.sendMessage(ds, m.ChannelID, "Sweet! You're good to go.")
}

func (s *DiscordServer) incomingMessage(ds *discordgo.Session, m *discordgo.MessageCreate) {
	const prefix = "🙂"

	if !strings.HasPrefix(m.Content, prefix) {
		/// XXX DEBUG
		log.Println("no prefix", m.Content)
		return
	}

	cmd := strings.Fields(strings.TrimPrefix(m.Content, prefix))
	for i, c := range cmd {
		cmd[i] = strings.ToLower(strings.TrimSpace(c))
	}

	switch cmd[0] {
	case "create":
		s.handleCreate(ds, m, cmd[1])
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
