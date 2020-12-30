package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"sync"

	"github.com/bwmarrin/discordgo"
)

var (
	ErrGuildPlaylistExists       = errors.New("a playlist with that title already exists")
	ErrGuildPlaylistDoesNotExist = errors.New("a playlist with that title does not exist")
)

// guildPlaylists store all the guildPlaylists sorted (but with O(n logn) inserts)
// not thread safe (although this should not be relevant since they are only used
// by guildState, which locks itself anyway).
//
// since the web client curls VERY regularly it makes sense to implement it this
// way. we could also have some kind of cache and use a sorted set, but since
// we don't expect too many playlists to be added his is okay.
type guildPlaylists struct {
	playlists []*Playlist
	keys      map[string]struct{}
}

func newGuildPlaylists() *guildPlaylists {
	return &guildPlaylists{
		keys:      map[string]struct{}{},
		playlists: []*Playlist{},
	}
}

func (gp *guildPlaylists) GetAll() []*Playlist {
	return gp.playlists
}

func (gp *guildPlaylists) get(t string) (int, error) {
	_, exists := gp.keys[t]
	if !exists {
		return -1, ErrGuildPlaylistDoesNotExist
	}

	i := sort.Search(len(gp.playlists), func(i int) bool {
		return gp.playlists[i].Title >= t
	})

	if i >= len(gp.playlists) || gp.playlists[i].Title != t {
		// key's value does not reflect reality
		log.Printf("ERROR: data integrity issue: playlist '%s' did not exist in data but has key", t)
		delete(gp.keys, t)
		return -1, ErrGuildPlaylistDoesNotExist
	}

	return i, nil

}

func (gp *guildPlaylists) Get(t string) (*Playlist, error) {
	i, err := gp.get(t)
	if err != nil {
		return nil, err
	}
	return gp.playlists[i], nil
}

func (gp *guildPlaylists) Insert(pl *Playlist) error {
	_, exists := gp.keys[pl.Title]
	if exists {
		return ErrGuildPlaylistExists
	}

	gp.keys[pl.Title] = struct{}{}
	gp.playlists = append(gp.playlists, pl)
	gp.sort()

	return nil
}

func (gp *guildPlaylists) Remove(t string) error {
	_, exists := gp.keys[t]
	if !exists {
		return ErrGuildPlaylistDoesNotExist
	}

	delete(gp.keys, t)

	i, err := gp.get(t)
	if err != nil {
		return err
	}

	l := len(gp.playlists)
	copy(gp.playlists[i:], gp.playlists[i+1:])
	gp.playlists[l-1] = nil
	gp.playlists = gp.playlists[:l-1]
	return nil
}

func (gp *guildPlaylists) sort() {
	sort.Slice(gp.playlists, func(i, j int) bool {
		return gp.playlists[i].Title < gp.playlists[j].Title
	})
}

type guildState struct {
	sync.Mutex

	playlists *guildPlaylists

	msg       func(msg string) error
	joinVoice func() (voice *discordgo.VoiceConnection, err error)
	p         *Player
}

func newGuildState() *guildState {
	playlists := newGuildPlaylists()

	for _, sample := range samplePlaylists {
		newPl, err := NewPlaylist(sample.Title, sample.Category, sample.Tracks)
		if err != nil {
			log.Fatalf("could not init guildState from sample playlist: %v", err)
		}
		playlists.Insert(newPl)
	}

	// TODO: persist playlists
	return &guildState{
		p:         NewPlayer(),
		playlists: playlists,
	}
}

func (gs *guildState) SetPlaylist(title string) {
	gs.Lock()
	defer gs.Unlock()

	pl, err := gs.playlists.Get(title)
	if err != nil {
		log.Printf("SetPlaylist: cannot find playlist: %v", err) // XXX Debug
		gs.msg(fmt.Sprintf("Sorry, I can't find the playlist %#v.", title))
		return
	}

	if err := gs.p.SetPlaylist(pl); err != nil {
		log.Printf("SetPlaylist: cannot set: %v", err)
		msg := fmt.Sprintf("Couldn't set your playlist. Here's the debug output: %#v", err)
		gs.msg(msg)
		return
	}

	// Signal that we want to join the voice channel and start playing.
	gs.p.StartPlayLoop(gs.msg, gs.joinVoice)
}

func (gs *guildState) QueueSingle(search string) error {
	gs.Lock()
	defer gs.Unlock()

	if err := gs.p.QueueSingle(search); err != nil {
		log.Printf("QueueSingle(%s) error: %v", search, err)
		msg := fmt.Sprintf("Oops! Flargunnstow failed at the modest tasks that was his charge. Debug: %#v", err)
		gs.msg(msg)
		return err
	}

	// Signal that we want to join the voice channel and start playing.
	gs.p.StartPlayLoop(gs.msg, gs.joinVoice)
	return nil
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

func (gs *guildState) Playlists() []*Playlist {
	gs.Lock()
	defer gs.Unlock()

	return gs.playlists.GetAll()
}

func (gs *guildState) AddPlaylist(p *Playlist) error {
	gs.Lock()
	defer gs.Unlock()

	return gs.playlists.Insert(p)
}

func (gs *guildState) RemovePlaylist(title string) error {
	gs.Lock()
	defer gs.Unlock()

	return gs.playlists.Remove(title)
}

type Sessions struct {
	// guildLookup provides a way to ongoing sessions for a guild
	//   i.e., map[guild id] -> session id
	// This is used when we get a discord command, to ensure we modify
	// the session that belongs to that guild
	guildLookup sync.Map // map[string]string

	// states contains all ongoing discord sessions
	//   i.e., map[session id] -> state
	states sync.Map // map[string]*guildState
}

var ErrSessionExists = errors.New("session already exists")
var ErrSessionDoesNotExist = errors.New("session does not exist")

func (s *Sessions) FromOrCreate(guildID string,
	msg func(msg string) error, joinVoice func() (*discordgo.VoiceConnection, error)) (*guildState, string, error) {
	sID, ok := s.guildLookup.Load(guildID)
	if !ok {
		// XXX: WE NEED TO PERSIST GUILDS HERE!! SUPER MEGA IMPORTANT!!!
		seshID := generateSID(s) // assign a new one because of interface reasons :(
		state := newGuildState()

		s.states.Store(seshID, state)
		s.guildLookup.Store(guildID, seshID)

		sID = interface{}(seshID)
	}

	st, ok := s.states.Load(sID)
	if !ok {
		return nil, "", fmt.Errorf("Create: no corresponding guild state for session id %v", sID)
	}

	state := st.(*guildState) // allow panic here we ever store something that isn't a guildState
	state.msg = msg
	state.joinVoice = joinVoice

	return state, sID.(string), nil
}

func (s *Sessions) FromGuild(guildID string) (*guildState, error) {
	sID, exists := s.guildLookup.Load(guildID)
	if !exists {
		return nil, ErrSessionDoesNotExist
	}

	st, exists := s.states.Load(sID)
	if !exists {
		return nil, fmt.Errorf("FromGuild: no corresponding guild state for session id %v", sID)
	}
	state := st.(*guildState) // allow panic here we ever store something that isn't a guildState

	return state, nil
}

func (s *Sessions) Exists(sID string) bool {
	_, exists := s.states.Load(sID)
	return exists
}

func (s *Sessions) GetState(sID string) (*guildState, error) {
	st, exists := s.states.Load(sID)
	if !exists {
		return nil, errors.New("invalid id")
	}
	state := st.(*guildState) // allow panic here we ever store something that isn't a guildState
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

func generateSID(ongoingSessions *Sessions) string {
	// XXX make unique.
	pwi := rand.Intn(899998)
	return strconv.Itoa(pwi + 100000)
}
