import React from 'react';
import SibLogo from '../components/SibLogo';
import { getNavigationForRole, getRoleLabel } from './navigation';
import ProfilePage from './pages/ProfilePage';
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

const renderPage = (activeItem, user) => {
  if (activeItem.key === 'profile') {
    return <ProfilePage user={user} />;
  }

  return (
    <EmptyPage
      title={activeItem.title}
      description="Этот раздел мы перепишем с нуля следующим блоком."
    />
  );
};

const Workspace = ({ user, route, navigate, onLogout }) => {
  const navItems = getNavigationForRole(user?.role);
  const activeItem = navItems.find((item) => item.route === route) || navItems[0];
  const displayName = getDisplayName(user);

  return (
    <div className="ui-shell">
      <aside className="ui-sidebar">
        <div className="ui-brand">
          <SibLogo size={42} />
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
              onClick={() => navigate(item.route)}
            >
              <span className="ui-nav-icon" aria-hidden="true">{item.icon}</span>
              <span>{item.label}</span>
            </button>
          ))}
        </nav>
      </aside>

      <div className="ui-content">
        <header className="ui-topbar">
          <div>
            <span className="ui-user-role">{getRoleLabel(user?.role)}</span>
            <strong>{displayName}</strong>
          </div>
          <button type="button" className="ui-logout-button" onClick={onLogout}>
            Выйти
          </button>
        </header>

        <main className="ui-main">
          {renderPage(activeItem, user)}
        </main>
      </div>
    </div>
  );
};

export default Workspace;
