import React from 'react';
import _ from 'lodash';

function PlayerBar(props) {
  let player = (<div className="Player-Empty"/>);

  if (props.current_playlist.length !== 0) {
    player = (
      < Player
        playing={props.playing}
        current_playlist={props.current_playlist}
        handleSkip={props.handleSkip}
      />
    );
  }

  /* TODO: add docs
  <a
    href="https://remindmetowritedocs.invalidtld"
    target="_blank"
    rel="noopener noreferrer"
  >
    ?
  </a>
  */

  return (
    <div className="PlayerBar">
      { player }
    </div>
  );
}

// TODO: make this whole player float on the top and let you show the current playlist, and click to a specific song
class Player extends React.Component {
  constructor(props) {
    super(props);
    this.state = { show_playing: false };
  }

  toggle_show() {
    this.setState((prev) => {
      return { show_playing: !prev.show_playing }
    });
  }

  render() {
    console.log("PLAYER", this.props);

    // TODO: add onClick for playlists
    let playlist = _.map(this.props.current_playlist, (track, i) => {
      return (
        <div className="Player-Track" key={i}>
          <span className="Player-TrackName">{track.name}&nbsp;</span>
          <span className="Player-TrackSep"> - </span> 
          <span className="Player-TrackArtist">{track.artist}</span>
        </div>
      );
    });

    if (!this.state.show_playing) {
      playlist = ( <span></span> );
    }

    return (
      <div className="Player">
          <div className="Player-NowPlaying">
            <span className="Player-Note">♫&nbsp;</span>
            <span className="Player-Text">Now Playing: </span>
            <span className="Player-Name">{this.props.playing.name}&nbsp;</span>
            <span className="Player-Sep"> - </span> 
            <span className="Player-Artist">{this.props.playing.artist}</span>
          </div>

          <button
              type="button"
              className="Player-SkipButton"
              onClick={() => { this.props.handleSkip() }}>
            >>
          </button>

          <div className="Player-PopUp">
            <button
              type="button"
              className="Player-PopUpButton"
              onClick={() => { this.toggle_show() }}>
              Queue
            </button>
            <div className="Player-PopUpContent">
              { playlist }
            </div>
          </div>
      </div>
    );
  }
}

export default PlayerBar;
