package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/devoxel/dndmusic/spotify"
)

var (
	token         string
	port          int
	runningDir    string
	spotifyID     string
	spotifySecret string
	videoDir      string
	workingDir    string
	siteURL       string
)

func init() {
	flag.StringVar(&token, "t", "", "discord bot auth token")
	flag.StringVar(&siteURL, "url", "", "site url")
	flag.StringVar(&spotifyID, "spotify-id", "", "spotify id")
	flag.StringVar(&spotifySecret, "spotify-secret", "", "spotify secret")
	flag.StringVar(&videoDir, "video-dir", ".", "video-directory")
	flag.StringVar(&workingDir, "working-dir", ".", "working-directory")
	flag.IntVar(&port, "p", 8080, "port to run the discord bot")
	flag.StringVar(&runningDir, "d", "", "running directory")
}

func validatePassword(pw string) error {
	if len(pw) < 4 {
		return errors.New("session password should be longer than four characters")
	}
	if len(pw) > 32 {
		return errors.New("session password can not be longer than 32 characters")
	}
	return nil
}

func validateWorkingDir() {
	_, err := ioutil.ReadFile(workingDir + "/cookies.txt")
	if err != nil {
		log.Fatal("no cookies file in working dir")
	}
}

func initBot(ongoingSessions *SessionManager) *discordgo.Session {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("cannot init discord bot", err)
	}

	// dg.LogLevel = discordgo.LogDebug
	s := &DiscordBot{ongoingSessions}
	dg.AddHandler(s.incomingMessage)

	if err = dg.Open(); err != nil {
		log.Fatal("cannot init websocket: ", err)
	}

	return dg
}
func initADM() {
	// XXX: dirty global
	adm = &AudioDownloadManager{
		// XXX: AudioDownloadManager could sync cache from file tree.
		passiveDL:     make(chan []Track),
		playlistCache: map[string][]string{},
		trackCache:    map[string]Track{},
		searchCache:   map[string]string{},
		s:             &spotify.Client{ClientID: spotifyID, ClientSecret: spotifySecret},
	}

	if err := adm.s.Authorize(); err != nil {
		log.Fatalf("cannot init spotify client: %v", err)
	}

	if err := adm.readCache(); err != nil {
		log.Fatal(err)
	}

	go adm.PassiveDownload()
}

func main() {
	flag.Parse()

	validateWorkingDir()
	rand.Seed(time.Now().Unix())

	log.Println("starting bot ...") // XXX: Debug

	initADM()

	log.Println("adm started ...") // XXX: Debug

	if token == "" {
		log.Fatal("no token provided")
	}

	if siteURL == "" {
		log.Fatal("no site url provided")
	}

	ongoingSessions := &SessionManager{
		sessions:      sync.Map{},
		guildLookup: sync.Map{},
	}

	dg := initBot(ongoingSessions)

	log.Println("discord initalized ...") // XXX: Debug
	handlerInit(ongoingSessions)

	initSample()

	sc := make(chan os.Signal, 1)

	go func() {
		log.Printf("hosting web server on port: %v ...", port)
		if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil); err != nil {
			log.Fatal("error hosting server: ", err)
		}
	}()

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}
