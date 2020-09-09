import React from 'react';

import './App.css';
import InvalidSession from './InvalidSession.js';
import ValidSession from './ValidSession.js';

class App extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      password: "",
      validated: false,
      playlists: null,
    }
  }

  componentDidMount() {
    this.setState({ password: "" });

    const socket = new WebSocket("wss://sb.invalidsyn.tax/ws");

    socket.onmessage = (ev) => {
      // TODO: create WebSocket chatter protocol here.
      // Create login success message, allowing app to transtion to 
      // music screen. State should be stored in the message, to allow
      // for user creation/etc down the line.
      //
      // On connection server should wait for StatusCheck message. Then
      // the server should report a status and write the password required
      // if that status is "Unverified".
      // 
      // If the status is Verified then the server should return a Music UI
      // state - a list of playlists to render on the UI to be selected by the
      // user. If a user clicks on a playlist, the client should send a MusicSelect
      // message, which prompts the bot to start playing music.
      //
      // # StatusCheck (sent by client) Ask the server for a status
      //
      // ## Request
      //
      // {
      //        "message": "StatusCheck",
      // }
      //
      // ## Responses
      //
      // {
      //        "message": "StatusCheckResponse",
      //        "status": "Verified"
      //        "playlists": []string{}...
      //        "nowPlaying": {
      //                "playlist": "https://.../",
      //                "song": "Living La Vida Loca",
      //        },
      // },
      // {
      //        "message": "StatusCheckResponse",
      //        "status": "Unverified"
      //        "password": "123123",
      // }
      //
      // 
      // # Music Selection
      //
      // ## Request
      //
      // {
      //        "message": "MusicSelect",
      //        "type": "Playlist",
      //        "playlist": "https://.../",
      // }
      //
      // {
      //        "message": "MusicSelect",
      //        "type": "SkipSong",
      // }
      //
      // {
      //        "message": "MusicSelect",
      //        "type": "SetSong",
      //        "song": "",
      // }
      //
      // - 
      //
      const msg = JSON.parse(ev.data);
      console.log("ws: message: ", msg);

      if (msg.message === "StatusCheckResponse") {
        /*
        if (!validStatusCheckResponse(msg)) {
          // TODO: handle errors
          return
        }
        console.log("ws: StatusCheckResponse"); // XXX: DEBUG
        */

        if (msg.status === "Unverified") {
          console.log("unverified"); // XXX: DEBUG
          this.setState({ password: msg.password });
          return
        }

        console.log("ws: Verified"); // XXX: DEBUG
        this.setState({ password: "Verified!", validated: true, playlists: msg.playlists });
      }

    };

    // TODO: enable these to only show on debug.
    socket.onopen = (ev) => {
      // Normal
      console.log("ws: Opening.");

      setInterval(() => {
        const msg = { 'message': 'StatusCheck' };
        const toSend = JSON.stringify(msg);
        console.log(toSend);
        socket.send(toSend);
      }, 600);
    }

    socket.onclose = (ev) => {
      // Shouldn't normally close.
      // TODO: Attempt a reconnect.
      // TODO: Show "Disconnected from server..." message.
      //
      console.log("ws: Closing.");
      console.log(ev);
    };

    socket.onerror = (ev) => {
      // Shouldn't normally error (obviously).
      // TODO: Show 504 error message.
      console.log("ws: Error.");
      console.log(ev);
    };

  }

  render() {
    let comp = <InvalidSession password={this.state.password}/>
    if (this.state.validated) {
      comp = <ValidSession playlists={this.state.playlists} />
    }

    return (
      <div className="App">
        { comp }
      </div>
    );
  }
}

export default App;
