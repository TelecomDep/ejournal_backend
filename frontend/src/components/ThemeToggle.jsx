import React, { useEffect, useState } from 'react';

const STORAGE_KEY = 'ejournal_theme';

const readInitialTheme = () => {
  const fromAttr = document.documentElement.getAttribute('data-theme');
  if (fromAttr === 'light' || fromAttr === 'dark') {
    return fromAttr;
  }
  return localStorage.getItem(STORAGE_KEY) === 'dark' ? 'dark' : 'light';
};

const ThemeToggle = () => {
  const [theme, setTheme] = useState(readInitialTheme);

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem(STORAGE_KEY, theme);
  }, [theme]);

  const isDark = theme === 'dark';

  return (
    <button
      type="button"
      className="theme-toggle"
      onClick={() => setTheme(isDark ? 'light' : 'dark')}
      aria-label={isDark ? 'Включить светлую тему' : 'Включить тёмную тему'}
      title={isDark ? 'Светлая тема' : 'Тёмная тема'}
    >
      {isDark ? '☀️' : '🌙'}
    </button>
  );
};

export default ThemeToggle;
