import React from 'react';
import logo from './logo.svg';

function InvalidSession(props) {
  return (
    <header className="App-header">
      <img src={logo} className="App-logo" alt="logo" />
      <p>
        Your password is: {props.password}
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
  );
}

export default InvalidSession;
