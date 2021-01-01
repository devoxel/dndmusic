package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/layeh/gopus"
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
	channels   = 2
	frameSize  = 960
	maxBytes   = frameSize * 4
)

func (t Track) CMD() *exec.Cmd {
	// oh lordy what a hack
	/*
		args := []string{}
		args = append(args, dlCmd()...)
		args = append(args, []string{
			"|",
		})
	*/
	return exec.Command(
		"bash",
		workingDir+"/dl.sh",
		t.URL,
	)
}

// PlayLoop manages the Player, grabbing tracks off the Q and decoding them.
//
// PlayLoop handles various signals, like file skipping.
func (p *Player) PlayLoop(msg func(string) error, joinVoice func() (*discordgo.VoiceConnection, error)) {
	p.Lock()
	p.playerOn = true
	p.pcm = make(chan []int16, 2)
	p.Unlock()
	defer func() {
		p.Lock()
		p.playerOn = false
		p.Unlock()
	}()

	logErr := func(err error) {
		log.Println("PlayLoop: error: ", err)
		msg(fmt.Sprintf("uh oh: %v", err)) // XXX: we could hide these
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start toDiscord goroutine
	go func() {
		if err := p.toDiscord(ctx, joinVoice); err != nil {
			logErr(err)
		}
		log.Println("PlayLoop: toDiscord exit")
		// If discord exits, we should finish also, since it exits after
		// 1 min anyway.
		// TODO: could maybe change who is in control of the timeout here
		cancel()
	}()

	for {
		t, _, err := p.q.Current()
		if err == ErrNoSongs {
			time.Sleep(500 * time.Millisecond)
			continue
		} else if err != nil {
			logErr(err)
			cancel()
			return
		}

		log.Println("PlayLoop: playing track =", t)

		quit, err := p.DecodeTrackLoop(ctx, t.CMD())
		if err != nil {
			logErr(err)
			cancel()
			return
		}

		if quit {
			cancel()
			return
		}
	}
}

// DecodeTrack decodes a track (using it's URL) and sends it into the
// pcm channel.
//
// DecodeTrackLoop is also responsible for handling signals like
// - reload / skip / etc
// since it's controlling PCM input.
//
// Credits are due to @github.com/ducc for his work on github.com/ducc/GoMusicBot
// which helped simplify this function from a gigantic shit stack of fuck
// into a somewhat reasonable thing.
// FWIW, they give credit to @github.com/bwmarrin's example, so thanks :)
func (p *Player) DecodeTrackLoop(ctx context.Context, f *exec.Cmd) (bool, error) {
	log.Println("DecodeTrackLoop: starting ", f.Args)
	const ffmpegBuffer = 16384

	out, err := f.StdoutPipe()
	if err != nil {
		return false, err
	}

	f.Stderr = os.Stderr

	defer func() {
		if f.Process != nil {
			if err := f.Process.Kill(); err != nil {
				log.Printf("DecodeTrackLoop: error killing ffpmeg: %v", err)
			}
		}
	}()

	ffIn := bufio.NewReaderSize(out, ffmpegBuffer)
	if err := f.Start(); err != nil {
		return false, err
	}

	buf := make([]int16, frameSize*channels)
	for {
		err = binary.Read(ffIn, binary.LittleEndian, &buf)
		if err == io.EOF {
			return false, nil
		} else if err == io.ErrUnexpectedEOF {
			// TODO: fix
			return false, nil
		} else if err != nil {
			return true, err
		}

		select {
		case <-ctx.Done():
			// We've been told to finish up here.
			return false, nil
		case in := <-p.signal:
			switch in.Type {
			case SigTypeReload:
				return false, nil
			case SigTypeSkip:
				p.q.SkipNext()
				return false, nil
			case SigTypeStop:
				return true, nil
			case SigTypeErr:
				return false, in.Err
			}
		case p.pcm <- buf:
		}
	}
	return false, nil
}

// toDiscord is responsible for handling the discord audio connection
//
// TODO: see if using ffmpeg c bindings would improve performance here.
//
// Credits are due to @github.com/ducc for his work on github.com/ducc/GoMusicBot
// which helped simplify this function from a gigantic shit stack of fuck
// into a somewhat reasonable thing.
// FWIW, they give credit to @github.com/bwmarrin's example, so thanks :)
func (p *Player) toDiscord(ctx context.Context,
	joinVoice func() (*discordgo.VoiceConnection, error)) error {
	encoder, err := gopus.NewEncoder(sampleRate, channels, gopus.Audio)
	if err != nil {
		return err
	}

	conn, err := joinVoice()
	if err != nil {
		return err
	}
	defer conn.Disconnect()

	conn.LogLevel = discordgo.LogDebug // XXX Debug
	if err := waitForReady(conn); err != nil {
		return err
	}

	var disabledSpeaking bool
	if err := conn.Speaking(true); err != nil {
		return err
	}

	for {
		// First grab packet from PCM (probably an ffmpeg stream)
		// TODO: investigate if we can do this inside ffmpeg
		var rawSamples []int16

		select {
		case <-ctx.Done():
			return nil
		case rawSamples = <-p.pcm:
			/*
				case <-time.After(time.Second * 1):
					if err := conn.Speaking(false); err != nil {
						return err
					}
					disabledSpeaking = true
				case <-time.After(time.Second * 10):
					// haven't got a frame in a while.
					// assume everything is okay and we are supposed to leave now.
					// this doubles as an auto timeout.
					return nil
			*/
		}

		if disabledSpeaking {
			if err := conn.Speaking(true); err != nil {
				return err
			}
		}

		opus, err := encoder.Encode(rawSamples, frameSize, maxBytes)
		if err != nil {
			return err
		}

		/* send packet to connection */
		select {
		case conn.OpusSend <- opus:
		case <-time.After(time.Second):
			// We haven't been able to send a frame in a second, assume something is fucked
			return errors.New("couldn't send audio to discord")
		}
	}
}
