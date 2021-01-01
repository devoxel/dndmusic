package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/devoxel/dndmusic/spotify"
)

type AudioDownloadManager struct {
	sync.Mutex

	// cache spotify playlists to avoid hitting limits
	playlistCache map[string][]string

	s *spotify.Client
}

func writeJSON(path string, t interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	// No reason to be concerned about bytes here for right now.
	e := json.NewEncoder(f)
	e.SetIndent("", "\t")
	if err := e.Encode(t); err != nil {
		return err
	}
	return f.Close()
}

func loadJSON(path string, t interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	d := json.NewDecoder(f)
	if err := d.Decode(t); err != nil {
		return err
	}

	return nil
}

func getPlaylistCachePath() string {
	return fmt.Sprintf("%s/playlist_cache.json", videoDir)
}

func getTrackCachePath() string {
	return fmt.Sprintf("%s/video_cache.json", videoDir)
}

func getSearchCachePath() string {
	return fmt.Sprintf("%s/search_cache.json", videoDir)
}

// flush cache to disk. don't hotload or anything fancy. eventually we could
// use redis or something fancy for the cache, for now just write to disc and we'll read it back
func (adm *AudioDownloadManager) flushCache() error {
	adm.Lock()
	defer adm.Unlock()

	if err := writeJSON(getPlaylistCachePath(), &adm.playlistCache); err != nil {
		return fmt.Errorf("writeJSON(playlistCache): %w", err)
	}

	return nil
}

func (adm *AudioDownloadManager) readCache() error {
	adm.Lock()

	if err := loadJSON(getPlaylistCachePath(), &adm.playlistCache); err != nil {
		if !os.IsNotExist(err) {
			adm.Unlock()
			return fmt.Errorf("loadJSON(discordCache): %w", err)
		}
	}

	adm.Unlock()

	return adm.flushCache()
}

func randDuration() time.Duration {
	const min time.Duration = time.Second * 60
	rand := time.Duration(rand.Intn(420)+1) * time.Second
	return min + rand
}

// Initalized in init(), see main.go
var adm *AudioDownloadManager = nil

type Player struct {
	sync.Mutex

	q      *PlayerQ
	pcm    chan []int16
	signal chan PlayerSignal

	playerOn bool
	exit     chan struct{}
}

func NewPlayer() *Player {
	return &Player{}
}

func (p *Player) Start(msg func(msg string) error, joinVoice func() (voice *discordgo.VoiceConnection, err error)) {
	log.Println("Start(): starting...") // XXX DEBUG
	p.Lock()
	defer p.Unlock()
	if p.playerOn {
		return
	}

	if p.q == nil {
		p.q = NewPlayerQ()
	}
	p.signal = make(chan PlayerSignal)
	p.playerOn = true
	go p.PlayLoop(msg, joinVoice)
}

func (p *Player) QueueSingle(search string) (Track, error) {
	log.Printf("QueueSingle: queueing %s", search)
	track, err := adm.DLInfo(search)
	if err != nil {
		return Track{}, err
	}

	p.Lock()
	if p.q == nil {
		p.q = NewPlayerQ()
	}
	p.q.Append(track)
	p.Unlock()

	return track, nil
}

func (p *Player) SetPlaylist(playlist *Playlist) error {
	p.Lock()

	// Set current song to top of playlist.
	p.q = NewPlayerQFromPlaylist(playlist.Tracks)

	p.Unlock()
	return nil
}

func (p *Player) Playing() (Track, []Track) {
	p.Lock()
	defer p.Unlock()

	if !p.playerOn {
		return Track{}, []Track{}
	}

	t, pl, err := p.q.Current()
	if err != nil {
		return Track{}, []Track{}
	}

	return t, pl
}

func (p *Player) Skip() {
	p.signal <- SigSkip
}

func (p *Player) Stop() {
	p.signal <- SigStop
}
