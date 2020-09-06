import React from 'react';
import logo from './logo.svg';

import './App.css';


class App extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      password: "",
      validated: false,
    }
  }

  componentDidMount() {
    this.setState({ password: "" });

    const socket = new WebSocket("wss://sb.invalidsyn.tax/ws");

    socket.onmessage = (ev) => {
      console.log("ws: message: ", ev.data);

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
      //        "status": "Verified"
      //        "playlists": []string{}...
      //        "nowPlaying": {
      //                "playlist": "https://.../",
      //                "song": "Living La Vida Loca",
      //        },
      // },
      // {
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
      this.setState({ password: ev.data });
    };

    // TODO: enable these to only show on debug.
    socket.onopen = (ev) => {
      // Normal
      console.log("ws: Opening.");
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
    return (
      <div className="App">
        <header className="App-header">
          <img src={logo} className="App-logo" alt="logo" />
          <p>
            Your password is: {this.state.password}
          </p>
          <a
            className="App-link"
            href="https://remindmetowritedocs.invalidtld"
            target="_blank"
            rel="noopener noreferrer"
          >
            Docs
          </a>
        </header>
      </div>
    );
  }
}

export default App;
