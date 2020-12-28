package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/devoxel/dndmusic/spotify"
	"github.com/jonas747/ogg"
)

type AudioDownloadManager struct {
	sync.Mutex

	// cache spotify playlists to avoid hitting limits
	playlistCache map[string]Playlist

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

// flush cache to disk. don't hotload or anything fancy. eventually we could
// use redis or something fancy for the cache, for now just write to disc and we'll read it back
func (adm *AudioDownloadManager) flushCache() error {
	adm.Lock()
	defer adm.Unlock()

	playlistCachePath := fmt.Sprintf("%s/playlist_cache.json", videoDir)
	if err := writeJSON(playlistCachePath, &adm.playlistCache); err != nil {
		return fmt.Errorf("writeJSON(playlistCache): %w", err)
	}

	// XXX: different directory
	trackCachePath := fmt.Sprintf("%s/video_cache.json", videoDir)
	if err := writeJSON(trackCachePath, &adm.trackCache); err != nil {
		return fmt.Errorf("writeJSON(trackCache): %w", err)
	}

	return nil
}

func (adm *AudioDownloadManager) readCache() error {
	adm.Lock()

	if err := loadJSON("discordCache", &adm.playlistCache); err != nil {
		if !os.IsNotExist(err) {
			adm.Unlock()
			return fmt.Errorf("loadJSON(discordCache): %w", err)
		}
	}

	if err := loadJSON(fmt.Sprintf("%s/videoCache", videoDir), &adm.trackCache); err != nil {
		if !os.IsNotExist(err) {
			adm.Unlock()
			return fmt.Errorf("loadJSON(videoCache): %w", err)
		}
	}

	adm.Unlock()

	return adm.flushCache()
}

func (adm *AudioDownloadManager) DownloadPlaylist(url string) (Playlist, error) {
	adm.Lock()
	if pl, exists := adm.playlistCache[url]; exists {
		adm.Unlock()
		log.Println("using cache")
		return pl, nil
	}

	pl, err := adm.s.GetPlaylist(url)
	if err != nil {
		adm.Unlock()
		return nil, err
	}
	adm.Unlock()

	page := pl.Tracks
	tracks := []Track{}

	for {
		for _, t := range page.Tracks {
			tracks = append(tracks, Track{t.Track.Name, t.Track.Artists[0].Name, ""})
		}

		err = adm.s.NextPage(&page)
		if err == spotify.ErrNoMorePages {
			adm.Lock()
			adm.playlistCache[url] = tracks
			adm.Unlock()

			adm.flushCache() // XXX: debug, shouldnt flush after every use
			return tracks, nil
		} else if err != nil {
			return nil, err
		}
	}

}

func urlescape(s string) string {
	// XXX: could be more effecient, i'm lazy though
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "/", "")
	return s
}

func (adm *AudioDownloadManager) DownloadTrack(track Track) (Track, error) {
	if track.Path != "" {
		return Track{}, errors.New("DownloadTrack: attempting to download track with path info")
	}

	if track.URL == "" {
		return Track{}, errors.New("DownloadTrack: cannot download track without URL")
	}

	adm.Lock()
	if t, exists := adm.trackCache[track.ID()]; exists {
		// Got a valid track in cache.
		adm.Unlock()
		return t, nil
	}
	adm.Unlock()

	id, err := tmpFileName([]string{"3gp", "aac", "flv", "m4a", "mp3", "mp4", "ogg", "wav", " webm"})
	if err != nil {
		return Track{}, err
	}

	args := []string{
		"-o",
		fmt.Sprintf("%s/%s.%%(ext)s", videoDir, id),
		"--restrict-filenames",
		"--user-agent", "Mozilla/5.0 (Windows NT 5.1; rv:36.0) Gecko/20100101 Firefox/36.0",
		"-x",
		"--cookies", "cookies.txt",
		"--audio-format", "opus",
		"--socket-timeout", "10",
		"--default-search", "auto",
		"--no-playlist",
		"--no-call-home",
		"--no-progress",
		urlescape(fmt.Sprintf("%s - %s", track.Name, track.Artist)),
	}

	cmd := exec.Command("/usr/local/bin/youtube-dl", args...)
	cmd.Dir = workingDir
	fmt.Println(cmd) // XXX DEBUG

	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			fmt.Println(string(ee.Stderr))
		}
		fmt.Println(string(out)) // XXX
		fmt.Println(err)
		return Track{}, fmt.Errorf("error downloading track")
	}
	fmt.Println("youtube-dl output: ", string(out)) // XXX

	filename := fmt.Sprintf("%s/%s.opus", videoDir, id)

	// Validate file exists
	_, err = os.Stat(filename)
	if err != nil {
		return Track{}, err
	}
	track.Path = filename

	adm.Lock()
	adm.trackCache[track.ID()] = track
	adm.Unlock()

	adm.flushCache() // XXX: Don't flush cache every iteration.

	return track, nil
}

func (adm *AudioDownloadManager) QueueTracksForPassiveDownload(tracks []Track) {
	adm.passiveDL <- tracks
}

func randDuration() time.Duration {
	min := time.Duration(60)
	rand := time.Duration(rand.Intn(480)+1) * time.Second
	if rand < min {
		return min
	}
	return rand
}

func (adm *AudioDownloadManager) PassiveDownload() {
	downloadQueue := []Track{}
	timer := time.NewTimer(randDuration())

	for {
		select {
		case tracks := <-adm.passiveDL:
			// XXX do this without reallocating
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
			fmt.Println("PassiveDownload: next wait duration: %v", d)
			timer.Reset(d)
			if !poppedTrack {
				break
			}

			if _, err := adm.DownloadTrack(track); err != nil {
				// XXX: remove erroring tracks from the list as a bandaid.
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

	currentPlaylist string
	playing         int
	playlist        Playlist

	playerOn bool
	signal   chan PlayerSignal
}

func NewPlayer() *Player {
	return &Player{}
}

func (p *Player) SetPlaylist(url string) error {
	p.Lock()

	// Check if we're currently playlist this playlist, if so, bounce.
	if p.currentPlaylist == url {
		p.Unlock()
		return nil
	}

	pl, err := adm.DownloadPlaylist(url)
	if err != nil {
		p.Unlock()
		return err
	}

	pl.Shuffle()

	// Set current song to top of playlist.
	p.currentPlaylist = url

	p.playing = 0
	p.playlist = pl

	if p.playerOn {
		p.signal <- SigReload
	}

	p.Unlock()

	adm.QueueTracksForPassiveDownload(pl[1:])

	return nil
}

func (p *Player) HandleSignal(in PlayerSignal) (bool, bool) {
	switch in.Type {
	case SigTypeReload:
		p.Lock()
		p.playing += 1
		p.Unlock()
		return true, false
	case SigTypeStop:
		return false, true
	case SigTypeSkip:
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

	if len(p.playlist) == 0 {
		log.Println("playing with 0 length playlist??")
		return Track{}, []Track{}
	}

	c := p.playlist[p.playing]
	return Track{Name: c.Name, Artist: c.Artist}, p.playlist
}

func (p *Player) DownloadCurrent() (Track, error) {
	p.Lock()
	t := p.playlist[p.playing]
	p.Unlock()

	return adm.DownloadTrack(t)
}

func (p *Player) Skip() {
	p.signal <- SigSkip
}

func (p *Player) Stop() {
	p.signal <- SigStop
}

func (p *Player) StartPlayLoop(msg func(msg string) error, joinVoice func() (voice *discordgo.VoiceConnection, err error)) {
	fmt.Println("StartPlayLoop(): starting...") // XXX DEBUG

	p.Lock()
	defer p.Unlock()
	if p.playerOn {
		return
	}

	p.signal = make(chan PlayerSignal)
	p.playerOn = true

	fmt.Println("StartPlayLoop(): loops on b") // XXX DEBUG
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

	handleErr := func(err error) {
		fmt.Println("err", err) // XXX
		msg("Uh oh, something went wrong.")
		msg(fmt.Sprintf("Here's the log if it helps: %v", err))
	}

	conn, err := joinVoice()
	if err != nil {
		handleErr(err)
		return
	}
	defer conn.Disconnect()

	conn.LogLevel = discordgo.LogDebug

	t := time.NewTimer(time.Second * 60)

Loop:
	for {
		select {
		case <-t.C:
			handleErr(errors.New("waited too long for voice ready"))
			return
		default:
			if conn.Ready {
				break Loop
			}
			time.Sleep(time.Second)
		}
	}

	for {
		if p.playing >= len(p.playlist) {
			p.Lock()
			p.playing = 0
			p.Unlock()
		}

		// XXX: A lot of functions here mutate stuff.
		//
		// We should download audio ahead of time too, DownloadCurrent should
		// communicate to a global youtube downloader worker pool, which will
		// prioritize the current song and add the rest as to download.
		//
		// Really I need to design a SANE way of handling all these threads together.
		// This function is handling both the audio control & the playback.

		track, err := p.DownloadCurrent()
		if err != nil {
			handleErr(err)
			return
		}

		f, err := os.Open(track.Path)
		if err != nil {
			handleErr(err)
			return
		}

		dec := ogg.NewPacketDecoder(ogg.NewDecoder(f))

		if err := conn.Speaking(true); err != nil {
			f.Close()
			handleErr(err)
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
				handleErr(fmt.Errorf("error decoding opus frame: %w", err))
				return
			} else if err == io.EOF {

				p.Lock()
				p.playing += 1
				p.Unlock()

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
				handleErr(errors.New("discord connection borked"))
				return
			}
		}

		if err := conn.Speaking(false); err != nil {
			f.Close()
			handleErr(err)
			return
		}

		time.Sleep(250 * time.Millisecond)
	}
}
