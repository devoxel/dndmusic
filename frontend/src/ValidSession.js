import React from 'react';
import _ from 'lodash';

function PlaylistCategory(props) {
  console.log(props)

  const playList = _.map(props.playlists, (pl) => {
    return (
      <a href={pl.url} className="Playlist" key={pl.url}>
        <img className="Playlist-img" alt="album art" src={pl.album_art}/>
        <p>{pl.title}</p>
      </a>
    );
  });

  console.log("asdfadf", playList[0])
  console.log(playList instanceof Array)

  return (
    <div className="PlaylistCategory">
     <hr/>
     <p> { props.name } </p>
     { playList }
    </div>
  );
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
    return <PlaylistCategory name={v} playlists={k} />
  });

  console.log(playlists);

  return (
    <header className="App-header">
      { playlists }
    </header>
  );
}

export default ValidSession;
