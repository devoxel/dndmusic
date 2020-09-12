package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type guildState struct {
	confirmed bool

	channel  string
	password string

	playlist string
}

func (gs *guildState) SetPlaylist(url string) {
	gs.playlist = url
	fmt.Println("guildState: setPlaylist: ", url)
	// Check if we're currently playlist this playlist, if so, bounce.
	// Download playlist if we haven't we haven't seen it before.
	// Shuffle playlist (by default).
	// Set current song to top of playlist.
	// Signal that we want to join the voice channel and start playing.
}

type Sessions struct {
	sync.Mutex

	// map[session] = states
	states map[string]*guildState

	// map[passwords] = session
	pwValidation map[string]string
}

func (s *Sessions) Confirm(pw string, m *discordgo.MessageCreate) error {
	s.Lock()
	defer s.Unlock()

	// TODO: Add max pw inputs

	session, exists := s.pwValidation[pw]
	if !exists {
		// XXX: DEBUG
		log.Println(pw, s.states, s.pwValidation)
		return errors.New("invalid password")
	}

	delete(s.pwValidation, pw)

	state := s.states[session]
	state.confirmed = true
	s.states[session] = state

	return nil
}

func (s *Sessions) Exists(id string) bool {
	s.Lock()
	defer s.Unlock()

	_, exists := s.states[id]
	return exists
}

func (s *Sessions) Validate(id string) bool {
	s.Lock()
	defer s.Unlock()

	state, exists := s.states[id]
	if !exists {
		return false
	}

	return state.confirmed
}

func (s *Sessions) GetState(id string) (*guildState, error) {
	state, exists := s.states[id]
	if !exists {
		return nil, errors.New("invalid id")
	}

	if !state.confirmed {
		return nil, errors.New("unverified id")
	}

	return state, nil
}

func (s *Sessions) SetPlaylist(id, url string) error {
	s.Lock()
	state, err := s.GetState(id)
	if err != nil {
		return err
	}
	s.Unlock()

	state.SetPlaylist(url)

	return nil
}

func genPassword(ongoingSessions *Sessions) string {
	pwi := rand.Intn(899998)
	return strconv.Itoa(pwi + 100000)
}

func (s *Sessions) Password(id string) string {
	state, exists := s.states[id]
	if !exists {
		state = &guildState{}
	}

	if state.password == "" {
		state.password = genPassword(s)
	}

	s.Lock()
	defer s.Unlock()
	s.pwValidation[state.password] = id
	s.states[id] = state

	return state.password
}
