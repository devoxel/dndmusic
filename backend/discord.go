package main

import (
	"errors"
	"fmt"
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

func (s *DiscordServer) handleMessage(ds *discordgo.Session, m *discordgo.MessageCreate) {
	const prefix = ";"

	if !strings.HasPrefix(m.Content, prefix) {
		return
	}

	cmd := strings.Fields(strings.TrimPrefix(m.Content, prefix))
	for i, c := range cmd {
		if i == 0 {
			cmd[i] = strings.ToLower(cmd[i])
		}
		cmd[i] = strings.TrimSpace(c)
	}

	if len(cmd) == 0 {
		return
	}

	switch cmd[0] {
	case "create":
		s.handleCreate(ds, m)
	case "stop":
		s.handleStop(ds, m)
	case "skip":
		s.handleSkip(ds, m)
	case "add":
		pl, err := s.parsePlaylistArgs(cmd)
		if err != nil {
			s.sendErrorMsg(ds, m, err)
			return
		}
		s.handleAdd(ds, m, pl)
	case "delete":
		pl, err := s.parsePlaylistArgs(cmd)
		if err != nil {
			s.sendErrorMsg(ds, m, err)
			return
		}
		s.handleDelete(ds, m, pl)
	}
}

func (s *DiscordServer) handleAdd(ds *discordgo.Session, m *discordgo.MessageCreate, pl WSPlaylist) {
	gs, err := s.sessions.FromGuild(m.GuildID)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}

	if err := gs.AddPlaylist(pl); err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}
}

func (s *DiscordServer) handleDelete(ds *discordgo.Session, m *discordgo.MessageCreate, pl WSPlaylist) {
	gs, err := s.sessions.FromGuild(m.GuildID)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}
	if err := gs.RemovePlaylist(pl); err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}
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

func (s *DiscordServer) handleCreate(ds *discordgo.Session, m *discordgo.MessageCreate) {
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

	fmt.Println("createing sessions")
	sessionToken, err := s.sessions.Create(m, sendMsg, joinVoice)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
	}

	fmt.Println(sessionToken)

	hi := "join here: "
	sendMsg(fmt.Sprintf("%s %s/?s=%s", hi, siteURL, sessionToken))
}

func (s *DiscordServer) handleStop(ds *discordgo.Session, m *discordgo.MessageCreate) {
	sendMsg := func(msg string) error {
		_, err := ds.ChannelMessageSend(m.ChannelID, msg)
		return err
	}

	gs, err := s.sessions.FromGuild(m.GuildID)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}
	gs.Stop()

	sendMsg("bye! see you soon :)")
}

func (s *DiscordServer) handleSkip(ds *discordgo.Session, m *discordgo.MessageCreate) {
	gs, err := s.sessions.FromGuild(m.GuildID)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}
	gs.Skip()
}

func (s *DiscordServer) sendMessage(ds discordSession, id, message string) {
	m, err := ds.ChannelMessageSend(id, message)
	if err != nil {
		log.Printf("error sending message: %v", err)
		log.Printf("channel.id = %v", id)
		log.Printf("m = %v", m)
	}
}

func (s *DiscordServer) parsePlaylistArgs(cmd []string) (WSPlaylist, error) {
	if len(cmd) != 4 {
		return WSPlaylist{}, errors.New("should be: 🙂 name url category")
	}
	log.Println("CMD=", cmd)

	return WSPlaylist{cmd[1], cmd[2], cmd[3]}, nil
}
