import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import 'tabulator-tables/dist/css/tabulator.min.css';
import './App.css';
import './theme.css';

// Apply the saved theme before the first paint to avoid a flash of the wrong theme.
const savedTheme = localStorage.getItem('ejournal_theme') === 'dark' ? 'dark' : 'light';
document.documentElement.setAttribute('data-theme', savedTheme);

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);