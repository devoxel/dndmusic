package main

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"

	"github.com/bwmarrin/discordgo"
)

var (
	ErrGuildPlaylistExists       = errors.New("a playlist with that title already exists")
	ErrGuildPlaylistDoesNotExist = errors.New("a playlist with that title does not exist")
)

type SessionManager struct {
	// guildLookup provides a way to ongoing sessions for a guild
	//   i.e., map[guild id] -> session id
	// This is used when we get a discord command, to ensure we modify
	// the session that belongs to that guild
	guildLookup sync.Map // map[string]string

	// sessions contains all ongoing discord sessions
	//   i.e., map[session id] -> state
	sessions sync.Map // map[string]*Session
}

var ErrSessionExists = errors.New("session already exists")
var ErrSessionDoesNotExist = errors.New("session does not exist")

func (s *SessionManager) FromOrCreate(guildID string,
	msg func(msg string) error, joinVoice func() (*discordgo.VoiceConnection, error)) (*Session, string, error) {
	sID, ok := s.guildLookup.Load(guildID)
	if !ok {
		// XXX: WE NEED TO PERSIST GUILDS HERE!! SUPER MEGA IMPORTANT!!!
		seshID := generateSID(s) // assign a new one because of interface reasons :(
		state := newSession()

		s.sessions.Store(seshID, state)
		s.guildLookup.Store(guildID, seshID)

		sID = interface{}(seshID)
	}

	st, ok := s.sessions.Load(sID)
	if !ok {
		return nil, "", fmt.Errorf("Create: no corresponding guild state for session id %v", sID)
	}

	state := st.(*Session) // allow panic here we ever store something that isn't a Session
	state.msg = msg
	state.joinVoice = joinVoice

	return state, sID.(string), nil
}

func (s *SessionManager) FromGuild(guildID string) (*Session, error) {
	sID, exists := s.guildLookup.Load(guildID)
	if !exists {
		return nil, ErrSessionDoesNotExist
	}

	st, exists := s.sessions.Load(sID)
	if !exists {
		return nil, fmt.Errorf("FromGuild: no corresponding guild state for session id %v", sID)
	}
	state := st.(*Session) // allow panic here we ever store something that isn't a Session

	return state, nil
}

func (s *SessionManager) Exists(sID string) bool {
	_, exists := s.sessions.Load(sID)
	return exists
}

func (s *SessionManager) GetState(sID string) (*Session, error) {
	st, exists := s.sessions.Load(sID)
	if !exists {
		return nil, errors.New("invalid id")
	}
	state := st.(*Session) // allow panic here we ever store something that isn't a Session
	return state, nil
}

func (s *SessionManager) SetPlaylist(id, url string) error {
	state, err := s.GetState(id)
	if err != nil {
		return err
	}
	state.SetPlaylist(url)
	return nil
}

func generateSID(ongoingSessions *SessionManager) string {
	// XXX: Ensure uniquness.
	pwi := rand.Intn(899998)
	return strconv.Itoa(pwi + 100000)
}
