import React from 'react';

import './App.css';
import InvalidSession from './InvalidSession.js';
import ValidSession from './ValidSession.js';

const urlParams = new URLSearchParams(window.location.search);
const session = urlParams.get('s');

const socket = new WebSocket("wss://" + window.location.host +"/ws?s=" + session);

class App extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      validated: false,
      playlists: null,
    }
  }

  componentDidMount() {
    this.setState({
      validated: false,
      playlists: [],
      playing: "",
      current_playlist: [],
    });

    socket.onmessage = (ev) => {
      const msg = JSON.parse(ev.data);
      console.log("ws: message: ", msg);

      if (msg.message === "StatusCheckResponse") {
        console.log("ws: StatusCheckResponse"); // XXX: DEBUG

        const playing = 'playing' in msg ? msg.playing : "";
        const cplaylist = 'current_playlist' in msg ? msg.current_playlist : [];

        this.setState({
          validated: true,
          playlists: msg.playlists,
          playing: playing,
          current_playlist: cplaylist,
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

  handleSkip() {
    const msg = { 'message': 'MusicSkip' };
    const toSend = JSON.stringify(msg);
    socket.send(toSend);
  }

  render() {
    let comp = <InvalidSession />
    if (this.state.validated) {
      comp = <ValidSession
        handlePlaylist={this.handlePlaylist}
        handleSkip={this.handleSkip}

        playlists={this.state.playlists}
        playing={this.state.playing}
        current_playlist={this.state.current_playlist}
      />
    }

    return (
      <div className="App">
        { comp }
      </div>
    );
  }
}

export default App;
