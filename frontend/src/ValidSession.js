import React from 'react';
import _ from 'lodash';
import PlayerBar from './Player.js';

function Playlist(props) {
  const playList = _.map(props.playlists, (pl) => {
    return (
      <div className="Playlist" >
        <p className="Playlist-Title">
          <a key={pl.url} onClick={() => { props.handlePlaylist(pl.title); }} className="Playlist-Link">
            {pl.title}
          </a>
        </p>
      </div>
    );
  });

  return (
    <div>
      { playList }
    </div>
  );
}

function Folder (props) {
  const ch = props.up ? "↓": "↑"
  return (
    <div className="PlaylistCategory-Folder" onClick={props.onClick}>{ch}</div>
  );
}

class PlaylistCategory extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      show: true,
    }
  }

  render() {
    // <img className="Playlist-img" alt="album art" src={pl.album_art}/>
    let playlist = ( <br/> )
    if (this.state.show) {
      playlist = (<Playlist handlePlaylist={this.props.handlePlaylist} playlists={this.props.playlists}/>)
    }

    return (
      <div className="PlaylistCategory">
        <h4 className="PlaylistCategory-Title"> { this.props.name } </h4>
        <Folder up={this.state.show} onClick={() => { this.setState({show: !this.state.show}) }} />
        { playlist }
      </div>
    );
  }
}

function ValidSession(props) {
  const categorys = {}

  _.map(props.playlists, (p) => {
    categorys[p.category] = []
  });

  _.map(props.playlists, (p) => {
    categorys[p.category].push(p)
  });

  console.log(categorys)

  const playlists = _.map(categorys, (k, v) => {
    return <PlaylistCategory handlePlaylist={props.handlePlaylist} name={v} playlists={k} />
  });

  console.log(playlists);

  return (
    <div className="ValidSession-body">
      { playlists }
      < PlayerBar
        playing={props.playing}
        current_playlist={props.current_playlist}
        handleSkip={props.handleSkip}
      />
    </div>
  );
}

export default ValidSession;
