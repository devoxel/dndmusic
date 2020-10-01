import React from 'react';
import logo from './logo.svg';

function InvalidSession(props) {
  return (
    <header className="InvalidSession-header">
      <div className="InvalidSession">
        <p>
          Your password is:&nbsp;
            <span className="InvalidSession-password">
              {props.password}
            </span>
        </p>
      </div>
    </header>
  );
}

export default InvalidSession;
