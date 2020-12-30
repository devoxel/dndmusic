package main

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
)

type SigType int

const (
	SigTypeReload = iota
	SigTypeSkip
	SigTypeStop
)

type Track struct {
	Name string `json:"name,omitempty"`
	Path string `json:"path,omitempty"`
	URL  string `json:"url,omitempty"`
}

func (t Track) ID() string {
	return t.URL
}

type Playlist struct {
	Title    string `json:"title,omitempty"`
	Category string `json:"category,omitempty"`

	Tracks []Track `json:"tracks,omitempty"`
}

func NewPlaylist(title string, category string, tracks []Track) (*Playlist, error) {
	if title == "" {
		return nil, errors.New("empty name")
	}

	if category == "" {
		return nil, errors.New("empty category")
	}

	return &Playlist{
		Title:    title,
		Category: category,
		Tracks:   tracks,
	}, nil
}

func NewPlaylistFromSpotifyURL(title string, category string, url string) (*Playlist, error) {
	if url == "" {
		return nil, errors.New("empty url")
	}

	tracks, err := adm.DownloadSpotifyPlaylist(url)
	if err != nil {
		return nil, fmt.Errorf("can't download playlist: %w", err)
	}

	return NewPlaylist(title, category, tracks)
}

func (p Playlist) Shuffle() {
	l := p.Tracks
	rand.Shuffle(len(l), func(i, j int) {
		l[i], l[j] = l[j], l[i]
	})
}

type PlayerSignal struct {
	Type SigType
}

var (
	SigReload = PlayerSignal{Type: SigTypeReload}
	SigSkip   = PlayerSignal{Type: SigTypeSkip}
	SigStop   = PlayerSignal{Type: SigTypeStop}
)

var ErrNoSongs = errors.New("no songs in player queue!")

// PlayerQ is the playlist type used by a player.
//
// A player will contain one instance of this playlist, which will
// be used to add / remove tracks.
//
// In general, a PlayerQ can be modified at any time but
// this will not affect the player's current state.
//
// Later when adding balanced Playlist we should be able to
// replace this using an interface to have a BalancedPlayerQ
type PlayerQ struct {
	sync.Mutex

	autoClear bool
	current   int
	playlist  []Track
}

func NewPlayerQ() *PlayerQ {
	return &PlayerQ{
		autoClear: true,
		current:   0,
		playlist:  []Track{},
	}
}

func NewPlayerQFromPlaylist(from []Track) *PlayerQ {
	return &PlayerQ{
		autoClear: false,
		current:   0,
		playlist:  from,
	}
}

func (p *PlayerQ) ToggleShuffle() {
	p.Lock()
	defer p.Unlock()
	p.autoClear = !p.autoClear
}

func (p *PlayerQ) Len() int {
	return len(p.playlist)
}

func (p *PlayerQ) Append(t Track) {
	p.Lock()
	defer p.Unlock()

	p.playlist = append(p.playlist, t)
}

func (p *PlayerQ) Insert(idx int, t Track) error {
	p.Lock()
	defer p.Unlock()
	if idx < 0 {
		return errors.New("index cannot be below zero")
	}

	if idx >= len(p.playlist) {
		p.playlist = append(p.playlist, t)
		return nil
	}

	p.playlist = append(p.playlist[:idx+1], p.playlist[idx:]...)
	p.playlist[idx] = t

	return nil
}

func (p *PlayerQ) SkipNext() (Track, bool) {
	p.Lock()
	defer p.Unlock()
	p.current += 1

	// TODO: Maybe reshuffle here
	if p.current > (len(p.playlist) - 1) {
		p.current = 0

		if p.autoClear {
			p.playlist = []Track{}
			return Track{}, true
		}
	}

	return p.playlist[p.current], false
}

func (p *PlayerQ) Current() (Track, []Track, error) {
	p.Lock()
	defer p.Unlock()
	if len(p.playlist) == 0 {
		return Track{}, []Track{}, ErrNoSongs
	}

	if p.current >= len(p.playlist) {
		// wuh woh
		return Track{}, []Track{}, errors.New("the queue has been mangled...")
	}

	return p.playlist[p.current], p.playlist, nil
}
