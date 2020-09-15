package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
)

func genSessionID(ongoingSessions *Sessions) string {
	for {
		session := strconv.Itoa(rand.Int())
		if !ongoingSessions.Exists(session) {
			return session
		}
	}
}

func writeError(where string, w http.ResponseWriter, r *http.Request, err error, c int) {
	log.Printf("%s: %v", where, err)
	http.Error(w, err.Error(), c)
}

func identityHandler(ongoingSessions *Sessions,
	cookieSession *sessions.Session, r *http.Request) (string, error) {
	// Current UserID Session
	id, exists := cookieSession.Values["id"]
	if !exists {
		id = genSessionID(ongoingSessions)
		cookieSession.Values["id"] = id
		log.Printf("new user, creating id: %v", id)
	}

	sid, valid := id.(string)
	if !valid {
		log.Printf("invalid cookie session: got %v", sid)
		sid = genSessionID(ongoingSessions)
		cookieSession.Values["id"] = sid
	}

	return sid, nil
}

type wsMsg struct {
	Message string `json:"message"`

	// StatusCheck
	Status    string       `json:"status,omitempty"`
	Password  string       `json:"password,omitempty"`
	Playlists []WSPlaylist `json:"playlists,omitempty"`

	// MusicSelect
	Type       string `json:"type,omitempty"` // UNUSED
	WSPlaylist string `json:"playlist,omitempty"`
}

type WSPlaylist struct {
	Title    string `json:"title,omitempty"`
	URL      string `json:"url,omitempty"`
	AlbumArt string `json:"album_art,omitempty"`
	Category string `json:"category,omitempty"`
}

func wsInvalidSession(ongoingSessions *Sessions, id string, req wsMsg) (wsMsg, error) {
	res := wsMsg{}

	if req.Message != "StatusCheck" {
		return res, errors.New("wsInvalidSession: Non StatusCheck in unvalidated session")
	}

	return wsMsg{
		Message:  "StatusCheckResponse",
		Status:   "Unverified",
		Password: ongoingSessions.Password(id),
	}, nil
}

func wsStatusCheck(ongoingSessions *Sessions, id string, req wsMsg) (wsMsg, error) {
	samplePlaylists := []WSPlaylist{
		{
			Title:    "Monsters: Tribesmen",
			AlbumArt: "https://i.scdn.co/image/ab67706c0000da842011b5c6608cb3063b3c9593",
			URL:      "https://open.spotify.com/playlist/2crzs0lic8x58JyPZM8k3v",
			Category: "Monsters",
		},
		{
			Title:    "Atmosphere: The Underdark",
			AlbumArt: "https://i.scdn.co/image/ab67706c0000da84107d8e2911ad8be24598e90a",
			URL:      "https://open.spotify.com/playlist/5Qhtamj9NCxluijLnQ4edN",
			Category: "Atmosphere",
		},
		{
			Title:    "PoTA: Sacred Stone Monastery",
			URL:      "https://open.spotify.com/playlist/3uJFVs1EUBA6jKqWhn9FA1",
			AlbumArt: "https://i.scdn.co/image/ab67706c0000da8443fdd964673d401481cd14b0",
			Category: "PoTA",
		},
		{
			Title:    "PoTA: Tower of Eagle Yokes",
			URL:      "https://open.spotify.com/playlist/3uJFVs1EUBA6jKqWhn9FA1",
			AlbumArt: "https://i.scdn.co/image/ab67706c0000da8443fdd964673d401481cd14b0",
			Category: "PoTA",
		},
		{
			Title:    "PoTA: That Town with the Big Hole",
			URL:      "https://open.spotify.com/playlist/3uJFVs1EUBA6jKqWhn9FA1",
			AlbumArt: "https://i.scdn.co/image/ab67706c0000da8443fdd964673d401481cd14b0",
			Category: "PoTA",
		},
		{
			Title:    "Atmosphere: The Capital",
			URL:      "https://open.spotify.com/playlist/2t5TWAPs6HYuJ3xbpjHYpx",
			AlbumArt: "https://i.scdn.co/image/ab67706c0000bebbe4884464ee49fddc2bee89c4",
			Category: "Atmosphere",
		},
		{
			Title:    "KoToR: Sad Star Wars",
			URL:      "https://open.spotify.com/playlist/3uJFVs1EUBA6jKqWhn9FA1",
			AlbumArt: "https://i.scdn.co/image/ab67706c0000da8443fdd964673d401481cd14b0",
			Category: "KoToR",
		},
		{
			Title:    "KoToR: Fighting the Sith",
			URL:      "https://open.spotify.com/playlist/3uJFVs1EUBA6jKqWhn9FA1",
			AlbumArt: "https://i.scdn.co/image/ab67706c0000da8443fdd964673d401481cd14b0",
			Category: "KoToR",
		},
		{
			Title:    "KoToR: Lightsabers!",
			URL:      "https://open.spotify.com/playlist/3uJFVs1EUBA6jKqWhn9FA1",
			AlbumArt: "https://i.scdn.co/image/ab67706c0000da8443fdd964673d401481cd14b0",
			Category: "KoToR",
		},
	}

	return wsMsg{
		Message:   "StatusCheckResponse",
		Status:    "Verified",
		Playlists: samplePlaylists,
	}, nil
}

func wsMusicSelect(ongoingSessions *Sessions, id string, req wsMsg) error {
	/* XXX: Eventually return to show errors to user.
	return wsMsg{
		Message "MusicSelectionResponse",
	}
	*/

	return ongoingSessions.SetPlaylist(id, req.WSPlaylist)
}

func readLoop(c *websocket.Conn, id string, ongoingSessions *Sessions) {
	// it would be more clever to not create my own simplistic RPC protocol.
	// here and instead use a proper RPC over websocket.
	// but lets be simple about it and just go for it.

	// TODO: Limit the amount of loops here to prevent ddos without a ticker.
	t := time.NewTicker(500 * time.Millisecond)
	defer c.Close()

	for {
		<-t.C

		messageType, r, err := c.NextReader()
		if err != nil {
			log.Printf("readLoop: read error: %v", err)
			c.Close()
			return
		}

		if messageType != websocket.TextMessage {
			log.Println("readLoop: bad message type")
			c.Close()
			return
		}

		var req wsMsg
		d := json.NewDecoder(r)

		if err = d.Decode(&req); err != nil {
			log.Printf("readLoop: Decode: %v", err)
			c.Close()
			return
		}

		// The state of Validate will change when the discord bot is correctly
		// validated. Shared state: a reliable system indeed!

		var res wsMsg

		switch {
		case !ongoingSessions.Validate(id): // XXX
			res, err = wsInvalidSession(ongoingSessions, id, req)
			if err != nil {
				log.Printf("readLoop: InvalidSession: %v", err)
				c.Close()
				return
			}
		case req.Message == "StatusCheck":
			res, err = wsStatusCheck(ongoingSessions, id, req)
			if err != nil {
				log.Printf("readLoop: StatusCheck: %v", err)
				c.Close()
				return
			}
		case req.Message == "MusicSelect":
			err = wsMusicSelect(ongoingSessions, id, req)
			if err != nil {
				log.Printf("readLoop: MusicSelect: %v", err)
				c.Close()
				return
			}
			continue
		}

		w, err := c.NextWriter(websocket.TextMessage)
		if err != nil {
			log.Printf("readLoop: NextWriter: %v", err)
			c.Close()
			return
		}

		e := json.NewEncoder(w)
		if err = e.Encode(res); err != nil {
			log.Printf("readLoop: Encode: %v", err)
			c.Close()
			return
		}

	}
}

func websocketHandler(ongoingSessions *Sessions) func(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true }, // XXX DEBUG
	}

	// XXX: change secret key
	store := sessions.NewCookieStore([]byte("asdfasdf"))

	return func(w http.ResponseWriter, r *http.Request) {
		// XXX DBEUG spew.Dump(r.Header)
		cookieSession, _ := store.Get(r, "session-name")

		id, err := identityHandler(ongoingSessions, cookieSession, r)
		if err != nil {
			writeError("/ws", w, r, fmt.Errorf("couldn't save session: %w", err), 500)
			return
		}

		// identityHandler will save the new ID, so save it to the HTTP in case of user reloading.
		if err := cookieSession.Save(r, w); err != nil {
			writeError("/ws", w, r, fmt.Errorf("couldn't save session: %w", err), 500)
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			writeError("ws", w, r, err, 500)
			return
		}

		fmt.Println("upgraded")

		readLoop(conn, id, ongoingSessions)
	}
}

func handlerInit(ongoingSessions *Sessions) {
	frontendPath := path.Join(runningDir, "frontend/build")
	index := path.Join(frontendPath, "index.html")
	_, err := os.Stat(index)
	if err != nil {
		log.Fatalf("cannot stat frontend path %v: %v", index, err)
	}

	d, _ := ioutil.ReadFile(index)
	fmt.Println(string(d))

	staticHandler := http.FileServer(http.Dir(frontendPath))

	http.HandleFunc("/ws", websocketHandler(ongoingSessions))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Should probably use something cached.
		// XXX: Remove hardcoded URL.
		staticHandler.ServeHTTP(w, r)
	})
}
