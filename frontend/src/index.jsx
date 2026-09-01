import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './App.css';
import './theme.css';

// The teacher-facing application uses one consistent light palette. Ignore a
// stale dark-theme preference left by an older production build so that a
// deployment cannot unexpectedly switch the whole app to a dark background.
document.documentElement.setAttribute('data-theme', 'light');

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
