package main

import (
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/gorilla/sessions"
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

func writeIndexHTML(w http.ResponseWriter, r *http.Request) {
	const index = "bundle/index.html"

	d, err := ioutil.ReadFile(index)
	if err != nil {
		writeError("writeIndexHTML: Read", w, r, err, http.StatusInternalServerError)
		return
	}

	// TODO: Easy optimization here: change this to read on boot.
	if _, err = w.Write(d); err != nil {
		writeError("writeIndexHTML: Write", w, r, err, http.StatusInternalServerError)
	}
}

func handlerInit() {
	// ongoingSessions is essentially the applications global state.
	// TODO: Add cleanup
	ongoingSessions := &Sessions{
		states:       map[string]*guildState{},
		pwValidation: map[string]string{},
	}

	store := sessions.NewCookieStore([]byte("asdfasdf"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// session := checkUserSession(r)
		cookieSession, _ := store.Get(r, "session-name")

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

		if err := cookieSession.Save(r, w); err != nil {
			log.Printf("couldnt save session: %v", err)
		}

		// One page app, so the only UI visable to user should be this one.
		writeIndexHTML(w, r)

		/* Switch to ws UI
		// Either show a password for a unvalidated session, or we can show a normal session.
		switch ongoingSessions.Validate(sid) {
		case true:
			log.Printf("valid user %v", id)
			fmt.Fprintf(w, "hi %v", sid)
		case false:
			// this case also creates the new user.
			log.Printf("unvalidated user %v", id)
			fmt.Fprintf(w, "your password is %v", ongoingSessions.Password(sid, ongoingSessions))
		}
		*/
	})
}
