package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
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
)

func init() {
	flag.StringVar(&token, "t", "", "discord bot auth token")
	flag.StringVar(&spotifyID, "spotify-id", "", "spotify id")
	flag.StringVar(&spotifySecret, "spotify-secret", "", "spotify secret")
	flag.StringVar(&videoDir, "video-dir", ".", "video-directory")
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

func initBot(ongoingSessions *Sessions) *discordgo.Session {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("cannot init discord bot")
	}

	s := &DiscordServer{ongoingSessions}
	dg.AddHandler(s.incomingMessage)

	if err = dg.Open(); err != nil {
		log.Fatal("cannot init websocket:", err)
	}

	return dg
}
func initADM() {
	// XXX: dirty global
	adm = &AudioDownloadManager{
		// XXX: AudioDownloadManager could sync cache from file tree.
		passiveDL: make(chan []Track),
		cache:     map[string]Playlist{},
		tcache:    map[string]Track{},
		s:         &spotify.Client{ClientID: spotifyID, ClientSecret: spotifySecret},
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
	rand.Seed(time.Now().Unix())

	log.Println("starting bot ...") // XXX: Debug

	initADM()

	log.Println("adm started ...") // XXX: Debug

	if token == "" {
		log.Fatal("no token provided")
	}

	ongoingSessions := &Sessions{
		states:         map[string]*guildState{},
		pwValidation:   map[string]string{},
		discordToState: map[string]string{},
	}

	dg := initBot(ongoingSessions)

	log.Println("discord initalized ...") // XXX: Debug
	handlerInit(ongoingSessions)

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
