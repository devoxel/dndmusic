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
	case "start":
		s.handleCreate(ds, m)
	case "stop":
		s.handleStop(ds, m)
	case "play":
		s.handlePlay(ds, m, strings.Join(cmd[1:], " "))
	case "skip":
		s.handleSkip(ds, m)
	case "add_playlist":
		if len(cmd) > 1 {
			// handle spotify playlist download
			s.sendErrorMsg(ds, m, errors.New("remind devoxel to implement this"))
			return
		}
		name := cmd[1]
		category := "misc" // XXX maybe shouldnt be magic text
		s.handleAdd(ds, m, name, category, []Track{})
	case "delete_playlist":
		name := cmd[1]
		s.handleDelete(ds, m, name)
	}
}

func (s *DiscordServer) handlePlay(ds *discordgo.Session, m *discordgo.MessageCreate, search string) {
	// XXX: remove duplication here
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

	gs, _, err := s.sessions.FromOrCreate(m.GuildID, sendMsg, joinVoice)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}

	if err := gs.QueueSingle(search); err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}

	// TODO: send nicely formatted ACK
}

func (s *DiscordServer) handleAdd(ds *discordgo.Session, m *discordgo.MessageCreate, name, category string, tracks []Track) {
	gs, err := s.sessions.FromGuild(m.GuildID)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}

	pl, err := NewPlaylist(name, category, tracks)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
		return

	}

	if err := gs.AddPlaylist(pl); err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}
}

func (s *DiscordServer) handleDelete(ds *discordgo.Session, m *discordgo.MessageCreate, name string) {
	/*
		gs, err := s.sessions.FromGuild(m.GuildID)
		if err != nil {
			s.sendErrorMsg(ds, m, err)
			return
		}
		if err := gs.RemovePlaylist(pl); err != nil {
			s.sendErrorMsg(ds, m, err)
			return
		}
	*/
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

	_, sessionToken, err := s.sessions.FromOrCreate(m.GuildID, sendMsg, joinVoice)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
	}

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
