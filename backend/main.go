package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	token      string
	port       int
	runningDir string
)

func init() {
	flag.StringVar(&token, "t", "", "discord bot auth token")
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

func sendErrorMsg(ds *discordgo.Session, cid string, err error) {
	log.Printf("sending err: %v", err)
	_, sErr := ds.ChannelMessageSend(cid, err.Error())
	if sErr != nil {
		log.Printf("cannot send error message: %v", sErr)
	}
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
	log.Println("initalized discord bot")

	return dg
}

func main() {
	flag.Parse()
	if token == "" {
		log.Fatal("no token provided")
	}

	ongoingSessions := &Sessions{
		states:       map[string]*guildState{},
		pwValidation: map[string]string{},
	}

	dg := initBot(ongoingSessions)
	handlerInit(ongoingSessions)

	sc := make(chan os.Signal, 1)

	go func() {
		log.Print("hosting web server on port: ", port)
		if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil); err != nil {
			log.Fatal("error hosting server: ", err)
		}
	}()

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}
