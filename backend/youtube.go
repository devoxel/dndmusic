package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
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
		"--max-filesize", "50m",
	}
}

func (adm *AudioDownloadManager) DownloadTrack(track Track) (Track, error) {
	if track.Path != "" {
		return track, nil
	}

	if track.URL == "" {
		return Track{}, errors.New("DownloadTrack: cannot download track without URL")
	}

	adm.Lock()
	if t, exists := adm.trackCache[track.ID()]; exists {
		if adm.trackCache[track.ID()].Path != "" {
			// Got a valid track in cache.
			adm.Unlock()
			return t, nil
		}
	}
	adm.Unlock()

	id, err := tmpFileName([]string{"3gp", "aac", "flv", "m4a", "mp3", "mp4", "ogg", "wav", " webm"})
	if err != nil {
		return Track{}, err
	}

	args := sharedArgs()
	args = append(args, []string{
		"-o",
		fmt.Sprintf("%s/%s.%%(ext)s", videoDir, id),
		"--restrict-filenames",
		"-x",
		"--audio-format", "opus",
		track.URL}...)

	cmd := exec.Command("/usr/local/bin/youtube-dl", args...)
	cmd.Dir = workingDir
	log.Println(cmd) // XXX DEBUG

	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			log.Println(string(ee.Stderr))
		}
		log.Println(string(out)) // XXX Debug
		log.Println(err)         // XXX Debug
		return Track{}, fmt.Errorf("error downloading track")
	}
	log.Println("youtube-dl output: ", string(out)) // XXX debug

	filename := fmt.Sprintf("%s/%s.opus", videoDir, id)

	// Validate file exists
	_, err = os.Stat(filename)
	if err != nil {
		// youtube dl can fail for a few reasons
		return Track{}, ErrDownloadFailed
	}
	track.Path = filename

	adm.Lock()
	adm.trackCache[track.ID()] = track
	adm.Unlock()

	adm.flushCache() // XXX: Don't flush cache every iteration.

	return track, nil
}

func (adm *AudioDownloadManager) GetYoutubeURL(search string) (Track, error) {
	adm.Lock()
	if uID, exists := adm.searchCache[search]; exists {
		defer adm.Unlock()

		// Got a valid UID, lets check if we have it downloaded first
		if t, exists := adm.trackCache[uID]; exists {
			// Got a valid track in cache.
			return t, nil
		}

		return Track{Name: search, URL: uID}, nil

	}
	adm.Unlock()

	// This is a pretty ugly function, could use refactoring.
	log.Printf("getYoutubeURLS: setting youtube urls for %v tracks", len(search))

	args := sharedArgs()
	args = append(args, "--get-id")
	args = append(args, "ytsearch:"+fmt.Sprintf("%v", search))

	cmd := exec.Command("/usr/local/bin/youtube-dl", args...)
	cmd.Dir = workingDir
	log.Printf("getYoutubeURLs: executing %#v", cmd) // XXX DEBUG

	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			log.Println("getYoutubeURLs: exit error, stderr: ", string(ee.Stderr))
		}
		return Track{}, err
	}

	outStr := strings.TrimSpace(string(out))
	outIDs := strings.Split(outStr, "\n")

	for _, id := range outIDs {
		t := Track{Name: search}
		id := strings.TrimSpace(id)
		if len(id) == 0 {
			continue
		}

		t.URL = fmt.Sprintf("https://youtube.com/watch?v=%v", id)

		adm.Lock()
		adm.searchCache[search] = t.URL
		adm.trackCache[t.URL] = t
		adm.Unlock()

		return t, nil
	}

	return Track{}, errors.New("couldn't find a match on youtube! im out of ideas chief")

}

func (adm *AudioDownloadManager) GetYoutubeURLs(search []string) ([]Track, error) {
	return []Track{}, errors.New("FIXME")
	/*
		// This is a pretty ugly function, could use refactoring.
		log.Printf("getYoutubeURLS: setting youtube urls for %v tracks", len(search))


		args := sharedArgs()
		args = append(args, "--get-id")
		for _, s := range search {
			args = append(args, "ytsearch:"+fmt.Sprintf("%v", s))
		}

		cmd := exec.Command("/usr/local/bin/youtube-dl", args...)
		cmd.Dir = workingDir
		log.Printf("getYoutubeURLs: executing %#v", cmd) // XXX DEBUG

		out, err := cmd.Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				log.Println("getYoutubeURLs: exit error, stderr: ", string(ee.Stderr))
			}
			return []Track{}, err
		}

		outStr := strings.TrimSpace(string(out))
		outIDs := strings.Split(outStr, "\n")

		newTracks := []Track{}
		ct := 0
		for _, id := range outIDs {
			if ct > (len(search) - 1) {
				log.Printf("getYoutubeURLs: too many output ids: %v", outIDs)
				return newTracks, nil
			}

			t := Track{Name: search[ct]}
			id := strings.TrimSpace(id)
			if len(id) == 0 {

				continue
			}

			t.URL = fmt.Sprintf("https://youtube.com/watch?v=%v", id)
			newTracks = append(newTracks, t)
			ct += 1
		}

		return newTracks, nil
	*/
}
