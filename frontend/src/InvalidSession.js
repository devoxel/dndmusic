import React from 'react';
import logo from './logo.svg';

function InvalidSession(props) {
  return (
    <header className="InvalidSession-header">
      <p>
        Your password is: {props.password}
      </p>
    </header>
  );
}

export default InvalidSession;
