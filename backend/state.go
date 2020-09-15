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
	sync.Mutex

	confirmed bool
	password  string

	msg       func(msg string) error
	joinVoice func() (voice *discordgo.VoiceConnection, err error)
	p         *Player
}

func (gs *guildState) SetPlaylist(url string) {
	gs.Lock()
	defer gs.Unlock()

	if err := gs.p.SetPlaylist(url); err != nil {
		log.Printf("SetPlaylist: cannot set: %v", err)

		msg := fmt.Sprintf("Couldn't set your playlist. Here's the error, if it helps: %v", err)
		gs.msg(msg)
	}

	// Signal that we want to join the voice channel and start playing.
	gs.p.StartPlayLoop(gs.msg, gs.joinVoice)

	return
}

type Sessions struct {
	sync.Mutex

	// map[session] = states
	states map[string]*guildState

	// map[passwords] = session
	pwValidation map[string]string
}

func (s *Sessions) Confirm(pw string, m *discordgo.MessageCreate,
	msg func(msg string) error, joinVoice func() (*discordgo.VoiceConnection, error)) error {
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
	state.msg = msg
	state.joinVoice = joinVoice

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
		state = &guildState{p: NewPlayer()}
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
