package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"time"
)

var (
	ErrDownloadFailed = errors.New("download failed")
)

func genUserAgent() string {
	now := time.Now()
	version := (now.Year() - 2018) + (int(now.Month()) / 4) + 58
	return fmt.Sprintf(
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:%v.0 Gecko/20100101 Firefox/%v.0",
		version, version)

}

func sharedArgs() []string {
	return []string{
		"--user-agent", genUserAgent(),
		"--add-header", "Accept-Language: \"en-US,en;q=0.5\"",
		"--cookies", "cookies.txt",
		"--socket-timeout", "10",
		"--default-search", "auto",
		"--no-playlist",
		"--no-call-home",
		"--no-progress",
		"--format", "bestaudio",
	}
}

func infoArgs() []string {
	shared := sharedArgs()
	return append(shared, "-j")
}

// CMD builds a youtube-dl download command for the given track
func (t Track) CMD() *exec.Cmd {
	/*
		// Here re-encode with ffmpeg which is faster using raw in between
		// TODO: replace ffmpeg args here with contants
		// TODO: test out "-movflags +faststart"
		args := []string{
			"-o", "-", // to stdout (for ffmpeg)
			// "--exec", "ffmpeg -i - -vn -sample_fmt s16 -acodec libopus -ar 48000 -ac 2",
			"--exec", "ffmpeg -vn -c:a libopus -b:a 48K -ac 2",
		}
		args = append(args, sharedArgs()...)
		args = append(args, t.URL)
		return exec.Command("youtube-dl", args...)
	*/
	// XXX: build command in GoLang.
	// Overhead of a shell is OK tbh.
	return exec.Command("bash", workingDir+"/download.sh", t.URL)
}

func runCmd(cmd *exec.Cmd) ([]byte, error) {
	cmd.Dir = workingDir

	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			log.Println(string(ee.Stderr))
		}
		log.Println(string(out)) // XXX Debug
		log.Println(err)         // XXX Debug
		return []byte{}, fmt.Errorf("error running %s", cmd.Path)
	}

	return out, nil
}

type youtubeDLResp struct {
	Formats []struct {
		URL string `json:"url"`
	} `json:"formats"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Uploader string `json:"uploader"`
}

func parseTrack(o []byte) (Track, error) {
	resp := youtubeDLResp{}
	err := json.Unmarshal(o, &resp)
	if err != nil {
		return Track{}, fmt.Errorf("parseTrack: %v", err)
	}
	if len(resp.Formats) == 0 {
		return Track{}, fmt.Errorf("download format not available")
	}
	return Track{Uploader: resp.Uploader, Name: resp.Title, URL: resp.URL}, nil
}

// DLInfo takes a search string (or any yt-dl argument) and converts it
// to a downloadable track.
//
// TODO: Add the ability to handle a playlist, ie return a []Track{} if
// given a playlist.  Ideally for this we could identify what yt-dl is doing
// with the given argument.
func (adm *AudioDownloadManager) DLInfo(search string) (Track, error) {
	args := infoArgs()
	args = append(args, search)
	cmd := exec.Command("/usr/local/bin/youtube-dl", args...)

	out, err := runCmd(cmd)
	if err != nil {
		return Track{}, err
	}

	track, err := parseTrack(out)
	if err != nil {
		return Track{}, err
	}

	return track, nil
}
