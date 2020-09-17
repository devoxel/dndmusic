import React from 'react';

import './App.css';
import InvalidSession from './InvalidSession.js';
import ValidSession from './ValidSession.js';

const socket = new WebSocket("wss://sb.invalidsyn.tax/ws");

const Footer = () => {
  return (
    <div className="Footer">
      <a
        href="https://remindmetowritedocs.invalidtld"
        target="_blank"
        rel="noopener noreferrer"
      >
        ?
      </a>
    </div>
  );
}

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

    socket.onmessage = (ev) => {
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
        this.setState({
          password: "Verified!",
          validated: true,
          playlists: msg.playlists,
          playing: msg.playing,
          current_playist: msg.current_playlist,
        });
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

  handlePlaylist(url) {
    console.log("PLAYLIST HANDLED ", url)

    const msg = {
      'message': 'MusicSelect',
      'type': 'Playlist',
      'playlist': url,
    };
    const toSend = JSON.stringify(msg);
    socket.send(toSend);
  }

  render() {
    let comp = <InvalidSession password={this.state.password}/>
    if (this.state.validated) {
      comp = <ValidSession handlePlaylist={this.handlePlaylist} playlists={this.state.playlists} />
    }

    return (
      <div className="App">
        { comp }
        < Footer />
      </div>
    );
  }
}

export default App;
