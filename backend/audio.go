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
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/devoxel/dndmusic/spotify"
	"github.com/jonas747/dca"
)

type Track struct {
	Name   string
	Artist string
	Path   string
}

type Playlist []*Track

func (p Playlist) Shuffle() {
	rand.Shuffle(len(p), func(i, j int) {
		p[i], p[j] = p[j], p[i]
	})
}

type AudioDownloadManager struct {
	sync.Mutex

	// playlist cache
	cache map[string]Playlist
	s     *spotify.Client
}

// flush cache to disk. don't hotload or anything fancy. eventually we could
// use redis or something fancy for the cache, for now just write to disc and we'll read it back
func (adm *AudioDownloadManager) flushCache() error {
	adm.Lock()
	defer adm.Unlock()

	f, err := os.Create("./discordCache")
	if err != nil {
		return err
	}

	e := json.NewEncoder(f)
	e.SetIndent("", "\t") // XXX Debug
	if err := e.Encode(&adm.cache); err != nil {
		return err
	}

	return nil
}

func (adm *AudioDownloadManager) readCache() error {
	f, err := os.Open("./discordCache")
	if os.IsNotExist(err) {
		return adm.flushCache()
	} else if err != nil {
		return err
	}

	d := json.NewDecoder(f)

	adm.Lock()
	defer adm.Unlock()
	if err := d.Decode(&adm.cache); err != nil {
		return err
	}

	return nil
}

func (adm *AudioDownloadManager) DownloadPlaylist(url string) (Playlist, error) {
	adm.Lock()
	if pl, exists := adm.cache[url]; exists {
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
	tracks := []*Track{}

	for {
		for _, t := range page.Tracks {
			tracks = append(tracks, &Track{t.Track.Name, t.Track.Artists[0].Name, ""})
		}

		err = adm.s.NextPage(&page)
		if err == spotify.ErrNoMorePages {
			adm.Lock()
			adm.cache[url] = tracks
			adm.Unlock()

			adm.flushCache() // XXX: debug, shouldnt flush after every use
			return tracks, nil
		} else if err != nil {
			return nil, err
		}
	}

}

func (adm *AudioDownloadManager) DownloadTrack(track *Track) error {
	if track.Path != "" {
		fmt.Println("using track cache")
		return nil
	}

	id, err := tmpFileName([]string{"3gp", "aac", "flv", "m4a", "mp3", "mp4", "ogg", "wav", " webm"})
	if err != nil {
		return err
	}

	args := []string{
		"-o",
		fmt.Sprintf("%s.%%(ext)s", id),
		"--restrict-filenames",
		"-x",
		"--audio-format", "opus",
		"--socket-timeout", "10",
		"--default-search", "auto",
		"--no-playlist",
		"--no-call-home",
		"--no-progress",
		fmt.Sprintf("%s - %s", track.Name, track.Artist),
	}

	cmd := exec.Command("/usr/local/bin/youtube-dl", args...)
	fmt.Println(cmd) // XXX DEBUG

	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			fmt.Println(string(ee.Stderr))
		}
		fmt.Println(string(out)) // XXX
		fmt.Println(err)
		return fmt.Errorf("error downloading track")
	}
	fmt.Println("youtube-dl output: ", string(out)) // XXX

	filename := fmt.Sprintf("%s.opus", id)

	// Validate file exists
	_, err = os.Stat(filename)
	if err != nil {
		return err
	}

	adm.Lock()
	track.Path = filename
	adm.Unlock()
	adm.flushCache()

	return nil
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

	playerOn    bool
	playerReset chan struct{}
}

func NewPlayer() *Player {
	return &Player{}
}

func (p *Player) SetPlaylist(url string) error {
	p.Lock()
	defer p.Unlock()

	// Check if we're currently playlist this playlist, if so, bounce.
	if p.currentPlaylist == url {
		return nil
	}

	pl, err := adm.DownloadPlaylist(url)
	if err != nil {
		return err
	}

	pl.Shuffle()

	// Set current song to top of playlist.
	p.currentPlaylist = url

	p.playing = 0
	p.playlist = pl

	if p.playerOn {
		p.playerReset <- struct{}{}
	}

	return nil
}

func (p *Player) DownloadCurrent() error {
	p.Lock()
	defer p.Unlock()
	return adm.DownloadTrack(p.playlist[p.playing])
}

func (p *Player) StartPlayLoop(msg func(msg string) error, joinVoice func() (voice *discordgo.VoiceConnection, err error)) {
	p.Lock()
	defer p.Unlock()
	if p.playerOn {
		return
	}

	p.playerReset = make(chan struct{})
	p.playerOn = true
	go p.PlayLoop(msg, joinVoice)
}

func (p *Player) PlayLoop(msg func(msg string) error, joinVoice func() (voice *discordgo.VoiceConnection, err error)) {
	defer func() {
		p.Lock()
		p.playerReset = nil
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

		if err := p.DownloadCurrent(); err != nil {
			handleErr(err)
			return
		}

		track := p.playlist[p.playing]
		f, err := os.Open(track.Path)
		if err != nil {
			handleErr(err)
			return
		}

		/* dec := ogg.NewDecoder(f) */
		dec, err := dca.EncodeMem(f, dca.StdEncodeOptions)
		if err != nil {
			f.Close()
			handleErr(err)
			return
		}

		if err := conn.Speaking(true); err != nil {
			f.Close()
			handleErr(err)
			return
		}

		time.Sleep(250 * time.Millisecond)

	EncodeLoop:
		for {
			/*
				page, err := dec.Decode()

				if err != nil && err == io.EOF {
					fmt.Println("play-debug: EOF") // XXX
					break
				} else if err != nil {
					f.Close()
					handleErr(err)
					return
				}

				bos := page.Type&ogg.BOS == 1
				fmt.Println("play-debug: bos=", bos) // XXX

				if bos && reading == 0xFF {
					reading = page.Serial
					fmt.Println("play-debug: reading ", reading) // XXX
					continue
				} else if bos {
					f.Close()
					handleErr(errors.New("unable to handle multiple ogg streams"))
					return
				}

				cop := page.Type&ogg.COP == 1
				if cop || len(packet) == 0 {
					fmt.Println("cop")
					packet = append(page.Packet) // XXX: don't reallocate
				} else if len(packet) > 0 {
					fmt.Println("flushing packet") //XXX
					conn.OpusSend <- packet
					packet = []byte{} // XXX: don't reallocate
				}

				if page.Type&ogg.EOS == 1 {
					fmt.Println("EOS") // XXX
					break
				}
			*/
			frame, err := dec.OpusFrame()
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
			case <-p.playerReset:
				break EncodeLoop
			case conn.OpusSend <- frame:
			case <-time.After(time.Second):
				// We haven't been able to send a frame in a second, assume the connection is borked
				handleErr(errors.New("discord connection borked"))
				return
			}
		}

		if err := dec.Stop(); err != nil {
			f.Close()
			handleErr(err)
			return
		}

		if err := conn.Speaking(false); err != nil {
			f.Close()
			handleErr(err)
			return
		}

		time.Sleep(250 * time.Millisecond)
	}
}
