// Credits to @github.com/ducc for his work on github.com/ducc/GoMusicBot
// which helped simplify this audio processing from a gigantic shit stack of fuck
// into a somewhat reasonable thing.  They give credit to @github.com/bwmarrin's also.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/ogg"
)

func waitForReady(conn *discordgo.VoiceConnection) error {
	const limit = time.Second * 60
	t := time.NewTimer(limit)
	for {
		select {
		case <-t.C:
			return fmt.Errorf("waited over timeout (>%v) for discord connection to ready up", limit)
		default:
			if conn.Ready {
				return nil
			}
			time.Sleep(time.Second)
		}
	}

}

const (
	sampleRate = 48000
	DLBitrate  = "48000K"
	channels   = 2
	frameSize  = 960
	maxBytes   = frameSize * 4
)

// PlayLoop manages the Player, grabbing tracks off the Q and decoding them.
//
// PlayLoop handles various signals, like file skipping.
func (p *Player) PlayLoop(msg func(string) error, joinVoice func() (*discordgo.VoiceConnection, error)) {
	p.Lock()
	p.playerOn = true
	// Using extra channels prevents ffmpeg stutters from disupting our output.
	// Why 64? I heard it's 1 stacks worth.
	audio := make(chan []byte, 64)
	p.Unlock()
	defer func() {
		p.Lock()
		p.playerOn = false
		p.Unlock()
	}()

	logErr := func(err error) {
		log.Println("PlayLoop: error: ", err)
		msg(fmt.Sprintf("uh oh: %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start toDiscord goroutine & ensure it dies correctly
	go func() {
		if err := p.toDiscord(ctx, audio, joinVoice); err != nil {
			logErr(err)
		}
	}()

	for {
		t, _, err := p.q.Current()
		if err == ErrNoSongs {
			time.Sleep(500 * time.Millisecond)
			continue
		} else if err != nil {
			logErr(err)
			return
		}

		log.Println("PlayLoop: playing track =", t)

		sig, err := p.DecodeTrackLoop(ctx, audio, t.CMD())
		if err != nil {
			logErr(err)
			return
		}

		/* TODO: move this logic to parent, stopping playback should be controlled from coordinater */
		switch sig.Type {
		case SigTypeReload:
			log.Println("got clear")
			continue
		case SigTypeSkip:
			p.q.SkipNext()
			continue
		case SigTypeStop:
			return
		case SigTypeErr:
			logErr(sig.Err)
			return
		default:
			err := errors.New("got unknown signal")
			logErr(err)
			return
		}

		// TODO: should impelment a catch here to prevent very fast reloading
	}
}

// DecodeTrack decodes a track (using it's URL) and sends it into the
// pcm channel.
//
// DecodeTrackLoop is also responsible for handling signals like
// - reload / skip / etc
// since it's controlling PCM input.
func (p *Player) DecodeTrackLoop(ctx context.Context, audio chan []byte, f *exec.Cmd) (PlayerSignal, error) {
	log.Println("DecodeTrackLoop: starting ", f.Args)
	const ffmpegBuffer = 16384 * 4

	out, err := f.StdoutPipe()
	if err != nil {
		return SigStop, err
	}
	f.Stderr = os.Stderr

	defer func() {
		if f.Process != nil {
			if err := f.Process.Kill(); err != nil {
				log.Printf("DecodeTrackLoop: error killing ffpmeg: %v", err)
			}
		}
	}()

	in := bufio.NewReaderSize(out, ffmpegBuffer)
	if err := f.Start(); err != nil {
		return SigStop, err
	}
	decoder := ogg.NewPacketDecoder(ogg.NewDecoder(in))

	skip := 2
	for {
		pkt, _, err := decoder.Decode()
		if err != nil && err != io.EOF {
			return SigStop, fmt.Errorf("error reading ogg: %w", err)
		} else if err == io.EOF {
			return SigSkip, nil
		}

		if skip > 0 {
			skip--
			continue
		}

		select {
		case <-ctx.Done():
			// We've been told to finish up here.
			return SigStop, nil
		case in := <-p.signal:
			return in, nil
		case audio <- pkt:
		}
	}
}

// toDiscord is responsible for handling the discord audio connection
func (p *Player) toDiscord(ctx context.Context, audio chan []byte,
	joinVoice func() (*discordgo.VoiceConnection, error)) error {
	conn, err := joinVoice()
	if err != nil {
		return err
	}
	defer conn.Disconnect()

	// conn.LogLevel = discordgo.LogDebug // uncomment if stuff starts acting weird
	if err := waitForReady(conn); err != nil {
		return err
	}

	if err := conn.Speaking(true); err != nil {
		return err
	}

	var in []byte
	for {
		select {
		case <-ctx.Done():
			return nil
		case in = <-audio:
		case <-time.After(time.Second * 120):
			// haven't got a frame in a long time
			// assume everything is okay and we are supposed to leave now.
			// this doubles as an auto timeout.
			return nil
		}

		select {
		case conn.OpusSend <- in:
		case <-time.After(time.Second):
			// We haven't been able to send a frame in a second, assume something is fucked
			return errors.New("couldn't send audio to discord")
		}
	}
}
