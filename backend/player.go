package main

import "math/rand"

type SigType int

const (
	SigTypeReload = iota
	SigTypeSkip
	SigTypeStop
)

type Track struct {
	Name   string `json:"name,omitempty"`
	Artist string `json:"artist,omitempty"`
	Path   string `json:"path,omitempty"`
	URL    string `json:"url,omitempty"`
}

func (t Track) ID() string {
	// XXX: Change once we allow tracks indexed by youtube URL
	return t.URL
}

type Playlist []Track

func (p Playlist) Shuffle() {
	rand.Shuffle(len(p), func(i, j int) {
		p[i], p[j] = p[j], p[i]
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
