import React, { useState } from 'react';
import SibLogo from '../components/SibLogo';
import { getNavigationForRole, getRoleLabel } from './navigation';
import ProfilePage from './pages/ProfilePage';
import SchedulePage from './pages/SchedulePage';
import TeacherPage from './pages/TeacherPage';
import './ui.css';

const getDisplayName = (user) => (
  user?.teacher_name || user?.name || user?.login || 'Пользователь'
);

const EmptyPage = ({ title, description }) => (
  <section className="ui-page">
    <div className="ui-page-header">
      <p className="ui-kicker">Новый интерфейс</p>
      <h1>{title}</h1>
      <p>{description}</p>
    </div>
  </section>
);

const iconPaths = {
  dashboard: (
    <>
      <path d="M4 10.5 12 4l8 6.5" />
      <path d="M6.5 9.5V20h4.2v-5.6h2.6V20h4.2V9.5" />
    </>
  ),
  schedule: (
    <>
      <rect x="4" y="5.5" width="16" height="14.5" rx="2.5" />
      <path d="M8 3.5v4" />
      <path d="M16 3.5v4" />
      <path d="M4 10h16" />
      <path d="M8 13.5h.01M12 13.5h.01M16 13.5h.01M8 17h.01M12 17h.01M16 17h.01" />
    </>
  ),
  grades: (
    <>
      <path d="M3.5 9 12 5l8.5 4-8.5 4-8.5-4Z" />
      <path d="M6.5 10.6v4.1c1.7 1.7 3.5 2.5 5.5 2.5s3.8-.8 5.5-2.5v-4.1" />
      <path d="M19.5 10v5.5" />
      <path d="M19.5 18.5v.01" />
    </>
  ),
  attendance: (
    <>
      <circle cx="12" cy="12" r="8.5" />
      <path d="m8.6 12.2 2.3 2.3 4.8-5" />
    </>
  ),
  analytics: (
    <>
      <path d="M5 19V9.5h3.2V19H5Z" />
      <path d="M10.4 19V5.5h3.2V19h-3.2Z" />
      <path d="M15.8 19v-7h3.2v7h-3.2Z" />
    </>
  ),
  stats: (
    <>
      <path d="M5 19V9.5h3.2V19H5Z" />
      <path d="M10.4 19V5.5h3.2V19h-3.2Z" />
      <path d="M15.8 19v-7h3.2v7h-3.2Z" />
    </>
  ),
  overview: (
    <>
      <path d="M5 19V9.5h3.2V19H5Z" />
      <path d="M10.4 19V5.5h3.2V19h-3.2Z" />
      <path d="M15.8 19v-7h3.2v7h-3.2Z" />
    </>
  ),
  profile: (
    <>
      <circle cx="12" cy="8" r="3.4" />
      <path d="M5.4 19c.9-3.5 3.1-5.3 6.6-5.3s5.7 1.8 6.6 5.3" />
    </>
  ),
  logout: (
    <>
      <path d="M10 6H6.8A1.8 1.8 0 0 0 5 7.8v8.4A1.8 1.8 0 0 0 6.8 18H10" />
      <path d="M13 8.5 16.5 12 13 15.5" />
      <path d="M16.5 12H9" />
    </>
  )
};

const NavIcon = ({ name }) => (
  <svg viewBox="0 0 24 24" aria-hidden="true">
    {iconPaths[name] || iconPaths.profile}
  </svg>
);

const renderPage = (activeItem, user, token) => {
  if (activeItem.key === 'profile') {
    return <ProfilePage user={user} token={token} />;
  }
  if (activeItem.key === 'schedule') {
    return <SchedulePage user={user} token={token} />;
  }
  if (user?.role === 'teacher' && ['attendance', 'stats', 'grades'].includes(activeItem.key)) {
    const section = activeItem.key === 'stats' ? 'statistics' : activeItem.key;
    return <TeacherPage user={user} token={token} section={section} />;
  }

  return (
    <EmptyPage
      title={activeItem.title}
      description="Этот раздел мы перепишем с нуля следующим блоком."
    />
  );
};

const Workspace = ({ user, token, route, navigate, onLogout }) => {
  const [isNavOpen, setIsNavOpen] = useState(false);
  const navItems = getNavigationForRole(user?.role);
  const activeItem = navItems.find((item) => item.route === route) || navItems[0];
  const displayName = getDisplayName(user);

  const handleNavigate = (to) => {
    navigate(to);
    setIsNavOpen(false);
  };

  return (
    <div className="ui-shell">
      <button
        type="button"
        className={`ui-nav-backdrop ${isNavOpen ? 'is-open' : ''}`}
        aria-label="Закрыть навигацию"
        onClick={() => setIsNavOpen(false)}
      />

      <aside className={`ui-sidebar ${isNavOpen ? 'is-open' : ''}`}>
        <div>
          <div className="ui-brand">
            <SibLogo size={38} />
            <div>
              <span className="ui-brand-title">СибГУТИ</span>
              <span className="ui-brand-subtitle">Электронный журнал</span>
            </div>
          </div>

          <nav className="ui-nav" aria-label="Разделы">
            {navItems.map((item) => (
              <button
                key={item.key}
                type="button"
                className={`ui-nav-button ${activeItem.key === item.key ? 'is-active' : ''}`}
                onClick={() => handleNavigate(item.route)}
              >
                <span className="ui-nav-icon" aria-hidden="true">
                  <NavIcon name={item.key} />
                </span>
                <span>{item.label}</span>
              </button>
            ))}
          </nav>
        </div>

        <button type="button" className="ui-sidebar-logout" onClick={onLogout}>
          <span className="ui-nav-icon" aria-hidden="true">
            <NavIcon name="logout" />
          </span>
          <span>Выйти</span>
        </button>
      </aside>

      <div className="ui-content">
        <header className="ui-topbar">
          <div className="ui-topbar-user">
            <button
              type="button"
              className="ui-menu-button"
              aria-label="Открыть навигацию"
              onClick={() => setIsNavOpen(true)}
            >
              <span />
              <span />
              <span />
            </button>

            <div>
              <span className="ui-user-role">{getRoleLabel(user?.role)}</span>
              <strong>{displayName}</strong>
            </div>
          </div>

          <div className="ui-topbar-actions" aria-hidden="true">
            <span className="ui-status-dot" />
          </div>
        </header>

        <main className="ui-main">
          {renderPage(activeItem, user, token)}
        </main>
      </div>
    </div>
  );
};

export default Workspace;
