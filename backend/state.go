package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type guildState struct {
	sync.Mutex

	confirmed bool
	password  string

	playlists []WSPlaylist

	msg       func(msg string) error
	joinVoice func() (voice *discordgo.VoiceConnection, err error)
	p         *Player
}

func newGuildState() *guildState {
	playlist := make([]WSPlaylist, len(samplePlaylists))
	copy(playlist, samplePlaylists)

	// TODO: persist playlists
	return &guildState{
		p:         NewPlayer(),
		playlists: samplePlaylists,
	}
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

func (gs *guildState) Playing() (Track, []Track) {
	return gs.p.Playing()
}

func (gs *guildState) Skip() {
	gs.p.Skip()
}

func (gs *guildState) Stop() {
	gs.p.Stop()
}

func (gs *guildState) Playlists() []WSPlaylist {
	gs.Lock()
	defer gs.Unlock()
	return gs.playlists
}

func validatePlaylist(p WSPlaylist) error {
	if p.Title == "" {
		return errors.New("empty name")
	}

	if p.URL == "" {
		return errors.New("empty url")
	}

	if p.Category == "" {
		return errors.New("empty category")
	}

	httpPrefix := func(s string) bool {
		return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
	}

	if httpPrefix(p.Title) {
		return errors.New("Title should not have http prefix")
	}

	if !httpPrefix(p.URL) {
		return errors.New("URL has no https:// prefix")
	}

	_, err := adm.DownloadPlaylist(p.URL)
	if err != nil {
		return fmt.Errorf("can't download playlist: %w", err)
	}

	return nil
}

func (gs *guildState) AddPlaylist(p WSPlaylist) error {
	if err := validatePlaylist(p); err != nil {
		return err
	}

	gs.Lock()
	defer gs.Unlock()

	gs.playlists = append(gs.playlists, p)
	return nil
}

func (gs *guildState) RemovePlaylist(match WSPlaylist) error {
	gs.Lock()
	defer gs.Unlock()

	// O(n) to remove a playlist, would be nice to remove this.
	got := -1
	for i, p := range gs.playlists {
		if p.Equal(match) {
			got = i
			break
		}
	}

	if got == -1 {
		return errors.New("no matching playlist")
	}

	copy(gs.playlists[got:], gs.playlists[got+1:])
	gs.playlists = gs.playlists[:len(gs.playlists)-1]

	return nil
}

type Sessions struct {
	sync.Mutex

	// map[guild id] -> session id
	guildLookup map[string]string

	// map[session id] -> state
	states map[string]*guildState

	// map[passwords] -> session id
	pwValidation map[string]string
}

var ErrSessionExists = errors.New("session already exists")

func (s *Sessions) Create(m *discordgo.MessageCreate,
	msg func(msg string) error, joinVoice func() (*discordgo.VoiceConnection, error)) (string, error) {
	s.Lock()
	defer s.Unlock()

	gid := m.GuildID
	sid, ok := s.guildLookup[gid]
	if !ok {
		sid = genPassword(s)
		state := newGuildState()
		// create guildState
		s.states[sid] = state
		s.guildLookup[gid] = sid
	}

	state, ok := s.states[sid]
	if !ok {
		return "", errors.New("Create: no corresponding guild state for session id " + sid)
	}

	state.confirmed = true
	state.msg = msg
	state.joinVoice = joinVoice

	return sid, nil
}

func (s *Sessions) FromGuild(gid string) (*guildState, error) {
	sid, ok := s.guildLookup[gid]
	if !ok {
		return nil, errors.New("start a session first")
	}

	state, ok := s.states[sid]
	if !ok {
		return nil, errors.New("FromGuild: no corresponding guild state for session id " + sid)
	}

	return state, nil
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
	return exists && state.confirmed
}

func (s *Sessions) GetState(id string) (*guildState, error) {
	s.Lock()
	defer s.Unlock()
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
	state, err := s.GetState(id)
	if err != nil {
		return err
	}

	state.SetPlaylist(url)
	return nil
}

func genPassword(ongoingSessions *Sessions) string {
	// XXX make unique.
	pwi := rand.Intn(899998)
	return strconv.Itoa(pwi + 100000)
}
