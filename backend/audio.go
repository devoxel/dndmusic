package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/devoxel/dndmusic/spotify"
	"github.com/jonas747/ogg"
)

type AudioDownloadManager struct {
	sync.Mutex

	// searchCache caches youtube searchs
	// string returned is the track UID, which is also its URL
	searchCache map[string]string

	// cache spotify playlists to avoid hitting limits
	playlistCache map[string][]string

	// cache tracks by mapping the tracks UID to a track that contains the
	// file path of that track
	trackCache map[string]Track

	passiveDL chan []Track

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

	if err := writeJSON(getTrackCachePath(), &adm.trackCache); err != nil {
		return fmt.Errorf("writeJSON(trackCache): %w", err)
	}

	if err := writeJSON(getSearchCachePath(), &adm.searchCache); err != nil {
		return fmt.Errorf("writeJSON(searchCache): %w", err)
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

	if err := loadJSON(getTrackCachePath(), &adm.trackCache); err != nil {
		if !os.IsNotExist(err) {
			adm.Unlock()
			return fmt.Errorf("loadJSON(videoCache): %w", err)
		}
	}

	if err := loadJSON(getSearchCachePath(), &adm.searchCache); err != nil {
		if !os.IsNotExist(err) {
			adm.Unlock()
			return fmt.Errorf("loadJSON(searchCache): %w", err)
		}
	}

	adm.Unlock()

	return adm.flushCache()
}

func (adm *AudioDownloadManager) DownloadSpotifyPlaylist(url string) ([]Track, error) {
	adm.Lock()

	pl, err := adm.s.GetPlaylist(url)
	if err != nil {
		adm.Unlock()
		return nil, err
	}
	adm.Unlock()

	page := pl.Tracks
	tracks := []string{}

AddLoop:
	for {
		for _, t := range page.Tracks {
			tracks = append(tracks, t.Track.Name+" - "+t.Track.Artists[0].Name)
		}

		err = adm.s.NextPage(&page)
		if err == spotify.ErrNoMorePages {
			adm.Lock()
			adm.playlistCache[url] = tracks
			adm.Unlock()

			adm.flushCache() // XXX: debug, shouldnt flush after every use
			break AddLoop
		} else if err != nil {
			return nil, err
		}
	}

	return adm.GetYoutubeURLs(tracks)
}

func (adm *AudioDownloadManager) QueueTracksForPassiveDownload(tracks []Track) {
	adm.passiveDL <- tracks
}

func randDuration() time.Duration {
	const min time.Duration = time.Second * 60
	rand := time.Duration(rand.Intn(420)+1) * time.Second
	return min + rand
}

func (adm *AudioDownloadManager) PassiveDownload() {
	downloadQueue := []Track{}
	timer := time.NewTimer(randDuration())

	for {
		select {
		case tracks := <-adm.passiveDL:
			downloadQueue = append(tracks, downloadQueue...)
		case <-timer.C:
			/* pop off queue until we get a non downloaded track */
			var poppedTrack bool
			var track Track

			for len(downloadQueue) > 0 {
				track = downloadQueue[0]
				downloadQueue = downloadQueue[1:]
				log.Printf("PassiveDownload: handling passive download: %s.\t%d left", track, len(downloadQueue))

				adm.Lock()
				id := track.ID()
				if _, exists := adm.trackCache[id]; exists {
					adm.Unlock()
					continue
				}
				adm.Unlock()

				poppedTrack = true
				break
			}

			d := randDuration()
			log.Printf("PassiveDownload: next wait duration: %v\n", d)
			timer.Reset(d)
			if !poppedTrack {
				break
			}

			if _, err := adm.DownloadTrack(track); err != nil {
				// XXX: Remove erroring tracks from the list as a band-aid.
				log.Printf("PassiveDownload failed: %w", err)
			}
		}
	}
}

func tmpFileName(prefixes []string) (string, error) {
	for i := 0; i < 10000; i++ {
		r := rand.Uint32()

		var found bool
		id := strconv.Itoa(int(r))
		for _, ext := range prefixes {
			_, err := os.Stat(fmt.Sprintf("./%s.%s", id, ext))
			if os.IsNotExist(err) {
				continue
			} else if err != nil {
				return "", err
			}
			found = true
		}

		if !found {
			return id, nil
		}
	}
	return "", errors.New("couldn't create tmp file name, how insane is that")
}

// Initalized in init(), see main.go
var adm *AudioDownloadManager = nil

type Player struct {
	sync.Mutex

	q *PlayerQ

	playerOn bool
	signal   chan PlayerSignal
}

func NewPlayer() *Player {
	return &Player{}
}

func (p *Player) QueueSingle(search string) error {
	track, err := adm.GetYoutubeURL(search)
	if err != nil {
		return err
	}

	p.Lock()
	if p.q == nil {
		p.q = NewPlayerQ()
	} else {
		// Don't queue if we don't have a playlist popping.
		adm.QueueTracksForPassiveDownload([]Track{track})
	}
	// Set track to bottom of playlist.
	p.q.Append(track)
	p.Unlock()
	return nil
}

func (p *Player) SetPlaylist(playlist *Playlist) error {
	p.Lock()

	// Set current song to top of playlist.
	p.q = NewPlayerQFromPlaylist(playlist.Tracks)
	if p.playerOn {
		p.signal <- SigReload
	}

	p.Unlock()
	adm.QueueTracksForPassiveDownload(playlist.Tracks[1:])

	return nil
}

func (p *Player) HandleSignal(in PlayerSignal) (bool, bool) {
	switch in.Type {
	case SigTypeReload:
		return true, false
	case SigTypeStop:
		return false, true
	case SigTypeSkip:
		p.q.SkipNext()
		return true, false
	default:
		log.Fatal("invalid player signal")
	}
	return false, false
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

func (p *Player) DownloadCurrent() (Track, error) {
	t, _, err := p.q.Current()
	if err != nil {
		return Track{}, err
	}
	return adm.DownloadTrack(t)
}

func (p *Player) Skip() {
	p.signal <- SigSkip
}

func (p *Player) Stop() {
	p.signal <- SigStop
}

func (p *Player) StartPlayLoop(msg func(msg string) error, joinVoice func() (voice *discordgo.VoiceConnection, err error)) {
	log.Println("StartPlayLoop(): starting...") // XXX DEBUG

	p.Lock()
	defer p.Unlock()
	if p.playerOn {
		return
	}

	p.signal = make(chan PlayerSignal)
	p.playerOn = true

	log.Println("StartPlayLoop(): loops on b") // XXX DEBUG
	go p.PlayLoop(msg, joinVoice)
}

func (p *Player) PlayLoop(msg func(msg string) error, joinVoice func() (voice *discordgo.VoiceConnection, err error)) {
	/* XXX
	This function needs to be split up and managed properly.
	Right now it does everything.
	*/
	defer func() {
		p.Lock()
		p.signal = nil
		p.playerOn = false
		p.Unlock()
	}()

	handleErr := func(m string, err error) {
		if m == "" {
			m = "uh oh, something went wrong."
		}
		log.Println("PlayLoop: error: ", err)
		msg(m)
	}

	conn, err := joinVoice()
	if err != nil {
		handleErr("", err)
		return
	}
	defer conn.Disconnect()

	conn.LogLevel = discordgo.LogDebug // XXX Debug

	t := time.NewTimer(time.Second * 60)

WaitLoop:
	for {
		select {
		case <-t.C:
			handleErr("", errors.New("waited too long for voice ready"))
			return
		default:
			if conn.Ready {
				break WaitLoop
			}
			time.Sleep(time.Second)
		}
	}

	// XXX: A lot of functions here mutate stuff.
	//
	// We should download audio ahead of time too, DownloadCurrent should
	// communicate to a global youtube downloader worker pool, which will
	// prioritize the current song and add the rest as to download.

WriteLoop:
	for {
		// TODO: we could do a timeout loop here rather than exiting immediatly
		// but im not going to do this right now.
		track, _, err := p.q.Current()
		if err == ErrNoSongs {
			return
		} else if err != nil {
			handleErr("", err)
			return
		}

		track, err = adm.DownloadTrack(track)
		if err == ErrDownloadFailed {
			m := fmt.Sprintf("failed to download %s. is it a dumb 10hour songs? i only have so much bandwith, anyway, moving on", track)
			handleErr(m, err)
			if _, exitLoop := p.q.SkipNext(); exitLoop {
				break WriteLoop
			}
		} else if err != nil {
			m := fmt.Sprintf("failed to download %s. i dont know why, anyway, moving on", track)
			handleErr(m, err)
			if _, exitLoop := p.q.SkipNext(); exitLoop {
				break WriteLoop
			}
			return
		}

		f, err := os.Open(track.Path)
		if err != nil {
			handleErr(fmt.Sprintf("i lost %s. yes i know i shouldnt have. i dunno, moving on", track), err)
			if _, exitLoop := p.q.SkipNext(); exitLoop {
				break WriteLoop
			}
			return
		}

		dec := ogg.NewPacketDecoder(ogg.NewDecoder(f))
		if err := conn.Speaking(true); err != nil {
			f.Close()
			handleErr("", err)
			return
		}

		time.Sleep(250 * time.Millisecond)

		// Account for the ID & comment header
		// See https://wiki.xiph.org/index.php?title=OggOpus.
		//
		// NOTE: If i want to calculate time into song we need to read this,
		// find pre-skip time & use that and granule data to calculate
		// PCM samples (seeking to the end of the song). Maybe its better to
		// set a timer based on the download length 5Head.

		skip := 2
	EncodeLoop:
		for {
			packet, _, err := dec.Decode()
			if skip > 0 {
				skip--
				continue
			}

			if err != nil && err != io.EOF {
				f.Close()
				err = fmt.Errorf("error decoding opus frame: %w", err)
				handleErr("", err)
				if _, exitLoop := p.q.SkipNext(); exitLoop {
					break WriteLoop
				}
			} else if err == io.EOF {
				_, exit := p.q.SkipNext()
				if exit {
					return
				}
				break
			}

			select {
			case in := <-p.signal:
				reload, exit := p.HandleSignal(in)
				if exit {
					return
				}
				if reload {
					break EncodeLoop
				}
			case conn.OpusSend <- packet:
			case <-time.After(time.Second):
				// We haven't been able to send a frame in a second, assume the connection is borked
				handleErr("", errors.New("discord is connection borked"))
				return
			}
		}

		if err := conn.Speaking(false); err != nil {
			f.Close()
			handleErr("", err)
			return
		}

		time.Sleep(250 * time.Millisecond)
	}
}
