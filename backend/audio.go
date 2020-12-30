package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/ogg"
)

func (p *Player) PlayLoop(msg func(msg string) error, joinVoice func() (voice *discordgo.VoiceConnection, err error)) {
	/* XXX
	u
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
