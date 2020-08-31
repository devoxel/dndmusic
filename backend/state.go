package main

import (
	"errors"
	"log"
	"math/rand"
	"strconv"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type guildState struct {
	confirmed bool
	channel   string
	password  string
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
	state, exists := s.states[id]
	if !exists {
		return false
	}

	return state.confirmed
}

func genPassword(ongoingSessions *Sessions) string {
	pwi := rand.Intn(899998)
	return strconv.Itoa(pwi + 100000)
}

func (s *Sessions) Password(id string, ongoingSessions *Sessions) string {
	state, exists := s.states[id]
	if !exists {
		state = &guildState{}
	}

	if state.password == "" {
		state.password = genPassword(ongoingSessions)
	}

	s.Lock()
	defer s.Unlock()
	s.pwValidation[state.password] = id
	s.states[id] = state

	return state.password
}
