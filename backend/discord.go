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

type DiscordBot struct {
	sessions *Sessions
}

// Have to wrap the function to allow us to use an interface
func (s *DiscordBot) incomingMessage(ds *discordgo.Session, m *discordgo.MessageCreate) {
	s.handleMessage(ds, m)
}

func (s *DiscordBot) handleMessage(ds *discordgo.Session, m *discordgo.MessageCreate) {
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
	case "create", "start":
		s.handleCreate(ds, m)
	case "stop":
		s.handleStop(ds, m)
	case "q", "queue":
		s.handleQueue(ds, m)
	case "play", "p":
		s.handlePlay(ds, m, strings.Join(cmd[1:], " "))
	case "skip", "s":
		s.handleSkip(ds, m)
	case "add_playlist":
		if len(cmd) > 1 {
			// TODO: handle spotify playlist download
			s.sendErrorMsg(ds, m, errors.New("remind devoxel to implement this"))
			return
		}
		name := cmd[1]
		category := "misc"
		s.handleAdd(ds, m, name, category, []Track{})
	case "delete_playlist":
		name := cmd[1]
		s.handleDelete(ds, m, name)
	}
}

func (s *DiscordBot) handlePlay(ds *discordgo.Session, m *discordgo.MessageCreate, search string) {
	gs, _, err := s.getOrCreateSession(ds, m)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}

	track, err := gs.QueueSingle(search)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}

	msg := &discordgo.MessageSend{
		Embed: &discordgo.MessageEmbed{
			Color: 3447003,
			Fields: []*discordgo.MessageEmbedField{{
				Name:  "Queued",
				Value: fmt.Sprintf("[%s](%s)", track.Name, track.URL),
			}},
		},
	}

	if _, err = ds.ChannelMessageSendComplex(m.ChannelID, msg); err != nil {
		log.Println("handlePlay: %v", err)
	}
}

func (s *DiscordBot) handleAdd(ds *discordgo.Session, m *discordgo.MessageCreate, name, category string, tracks []Track) {
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

func (s *DiscordBot) handleDelete(ds *discordgo.Session, m *discordgo.MessageCreate, name string) {
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

func (s *DiscordBot) sendErrorMsg(ds discordSession, m *discordgo.MessageCreate, err error) {
	log.Printf("sending err: %v", err)
	_, sErr := ds.ChannelMessageSend(m.ChannelID, err.Error())
	if sErr != nil {
		log.Printf("cannot send error message: %v", sErr)
	}
}

func (s *DiscordBot) getSenderCID(ds *discordgo.Session, guildID, authorID string) (string, error) {
	// Find the guild for that channel.
	g, err := ds.State.Guild(guildID)
	if err != nil {
		// Could not find guild.
		return "", err
	}

	// Look for the message sender in that guild's current voice states.
	for _, vs := range g.VoiceStates {
		if vs.UserID == authorID {
			return vs.ChannelID, nil
		}
	}
	return "", errors.New("You can't create a session if you're not in a voice channel")
}

func (s *DiscordBot) handleQueue(ds *discordgo.Session, m *discordgo.MessageCreate) {
	gs, err := s.sessions.FromGuild(m.GuildID)
	if err == ErrSessionDoesNotExist {
		// TODO: would be nice to actually check and log these errors everywhere
		ds.ChannelMessageSend(m.ChannelID, "i'm not playing anything")
		return
	} else if err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}

	tracks := []string{}
	playing, playlist := gs.Playing()
	for _, t := range playlist {
		if t.ID() == playing.ID() {
			tracks = append(tracks, "+  "+t.Name+" (now playing)")
		} else {
			tracks = append(tracks, "-  "+t.Name+"")
		}
	}

	msg := strings.Join(tracks, "\n")
	_, err = ds.ChannelMessageSend(m.ChannelID, "```\n"+msg+"\n```")
	if err != nil {
		log.Printf("handleQueue: %v", err)
	}

}

func (s *DiscordBot) sendMsg(ds *discordgo.Session, channelID, msg string) error {
	_, err := ds.ChannelMessageSend(channelID, msg)
	if err != nil {
		log.Printf("sendMsg: %v", err)
	}
	return err
}

func (s *DiscordBot) partialSendMsg(ds *discordgo.Session, channelID string) func(string) error {
	return func(m string) error {
		return s.sendMsg(ds, channelID, m)
	}
}

func (s *DiscordBot) partialJoinVoice(ds *discordgo.Session, guildID, authorID string) (func() (*discordgo.VoiceConnection, error), error) {
	audioID, err := s.getSenderCID(ds, guildID, authorID)
	if err != nil {
		return nil, err
	}

	return func() (*discordgo.VoiceConnection, error) {
		return ds.ChannelVoiceJoin(guildID, audioID, false, true)
	}, nil
}

func (s *DiscordBot) getOrCreateSession(ds *discordgo.Session, m *discordgo.MessageCreate) (*guildState, string, error) {
	joinVoice, err := s.partialJoinVoice(ds, m.GuildID, m.Author.ID)
	if err != nil {
		return nil, "", err
	}
	sendMsg := s.partialSendMsg(ds, m.ChannelID)
	return s.sessions.FromOrCreate(m.GuildID, sendMsg, joinVoice)
}

func (s *DiscordBot) handleCreate(ds *discordgo.Session, m *discordgo.MessageCreate) {
	_, sessionToken, err := s.getOrCreateSession(ds, m)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}
	s.sendMsg(ds, m.GuildID, fmt.Sprintf("%s %s/?s=%s", "join here: ", siteURL, sessionToken))
}

func (s *DiscordBot) handleStop(ds *discordgo.Session, m *discordgo.MessageCreate) {
	gs, err := s.sessions.FromGuild(m.GuildID)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}
	gs.Stop()

	ds.ChannelMessageSend(m.ChannelID, "bye! see you soon :)")
}

func (s *DiscordBot) handleSkip(ds *discordgo.Session, m *discordgo.MessageCreate) {
	gs, err := s.sessions.FromGuild(m.GuildID)
	if err != nil {
		s.sendErrorMsg(ds, m, err)
		return
	}
	gs.Skip()
}

func (s *DiscordBot) sendMessage(ds discordSession, id, message string) {
	m, err := ds.ChannelMessageSend(id, message)
	if err != nil {
		log.Printf("error sending message: %v", err)
		log.Printf("channel.id = %v", id)
		log.Printf("m = %v", m)
	}
}
