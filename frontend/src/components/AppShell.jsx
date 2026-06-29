import React from 'react';
import SibLogo from './SibLogo';
import './AppShell.css';

const AppShell = ({
  title,
  subtitle,
  navItems = [],
  activeKey,
  onNavigate,
  user,
  onLogout,
  children
}) => (
  <div className="app-shell">
    <header className="app-bar">
      <div className="app-bar-brand">
        <SibLogo size={44} />
        <div className="app-bar-titles">
          <span className="app-bar-title">{title}</span>
          {subtitle ? <span className="app-bar-subtitle">{subtitle}</span> : null}
        </div>
      </div>
      <div className="app-bar-actions">
        {user ? <span className="app-bar-user">{user}</span> : null}
        <button type="button" className="app-bar-logout" onClick={onLogout}>
          Выйти
        </button>
      </div>
    </header>

    {navItems.length > 0 && (
      <nav className="app-nav" aria-label="Разделы">
        {navItems.map((item) => (
          <button
            key={item.key}
            type="button"
            className={`app-nav-link ${activeKey === item.key ? 'active' : ''}`}
            onClick={() => onNavigate(item.key)}
          >
            {item.label}
          </button>
        ))}
      </nav>
    )}

    <main className="app-main">{children}</main>
  </div>
);

export default AppShell;
