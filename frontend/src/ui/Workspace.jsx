import React, { useCallback, useEffect, useState } from 'react';
import api from '../services/api';
import SibLogo from '../components/SibLogo';
import { getNavigationForRole, getRoleLabel } from './navigation';
import ProfilePage from './pages/ProfilePage';
import SchedulePage from './pages/SchedulePage';
import GradesPage from './pages/GradesPage';
import AttendancePage from './pages/AttendancePage';
import AnalyticsPage from './pages/AnalyticsPage';
import StaffAnalyticsPage from './pages/StaffAnalyticsPage';
import DashboardPage from './pages/DashboardPage';
import StaffOverviewPage from './pages/StaffOverviewPage';
import TeacherPage from './pages/TeacherPage';
import AdminUsersPage from './pages/AdminUsersPage';
import AdminNotificationsPage from './pages/AdminNotificationsPage';
import ReportsPage from './pages/ReportsPage';
import AdminSemestersPage from './pages/AdminSemestersPage';
import AntifraudPage from './pages/AntifraudPage';
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
  semesters: (
    <>
      <rect x="3.5" y="4.5" width="17" height="15" rx="2" />
      <path d="M3.5 10h17" />
      <path d="M8 2.5v4" />
      <path d="M16 2.5v4" />
      <path d="M7.5 14.5h3" />
      <path d="M13.5 14.5h3" />
    </>
  ),
  users: (
    <>
      <circle cx="9" cy="8" r="3" />
      <path d="M3.5 19c.6-3.4 2.4-5.2 5.5-5.2s4.9 1.8 5.5 5.2" />
      <path d="M15.5 5.8a3 3 0 0 1 0 5.8" />
      <path d="M16 14c2.6.4 4 2.1 4.5 5" />
    </>
  ),
  reports: (
    <>
      <path d="M6 3.5h8l4 4V20H6z" />
      <path d="M14 3.5V8h4" />
      <path d="M9 12h6M9 15.5h6" />
    </>
  ),
  notifications: (
    <>
      <path d="M6.5 10.5a5.5 5.5 0 0 1 11 0c0 5 2 5.5 2 5.5h-15s2-.5 2-5.5Z" />
      <path d="M10 19h4" />
    </>
  ),
  antifraud: (
    <>
      <path d="M12 3 5 6v5c0 4.6 2.7 8.1 7 10 4.3-1.9 7-5.4 7-10V6l-7-3Z" />
      <path d="M12 8v5M12 16.5v.01" />
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
  ),
  close: (
    <>
      <path d="m6 6 12 12" />
      <path d="M18 6 6 18" />
    </>
  )
};

const NavIcon = ({ name }) => (
  <svg viewBox="0 0 24 24" aria-hidden="true">
    {iconPaths[name] || iconPaths.profile}
  </svg>
);

const renderPage = (
  activeItem,
  user,
  token,
  route,
  navigate,
  onUserUpdate,
  onUnreadCountChange,
  onNotificationCreated
) => {
  if (activeItem.key === 'dashboard') {
    return <DashboardPage user={user} navigate={navigate} />;
  }
  if (activeItem.key === 'overview' && ['admin', 'head', 'dean'].includes(user?.role)) {
    return <StaffOverviewPage token={token} />;
  }
  if (activeItem.key === 'analytics' && ['admin', 'head', 'dean'].includes(user?.role)) {
    return <StaffAnalyticsPage token={token} />;
  }
  if (activeItem.key === 'reports' && ['admin', 'head', 'dean', 'teacher'].includes(user?.role)) {
    return <ReportsPage token={token} user={user} />;
  }
  if (activeItem.key === 'semesters' && user?.role === 'admin') {
    return <AdminSemestersPage token={token} />;
  }
  if (activeItem.key === 'users' && user?.role === 'admin') {
    return <AdminUsersPage token={token} currentUser={user} />;
  }
  if (activeItem.key === 'notifications' && user?.role === 'admin') {
    return <AdminNotificationsPage token={token} onNotificationCreated={onNotificationCreated} />;
  }
  if (activeItem.key === 'antifraud' && ['admin', 'teacher'].includes(user?.role)) {
    return <AntifraudPage token={token} user={user} />;
  }
  if (activeItem.key === 'profile') {
    const initialTab = route === '/profile/notifications'
      ? 'notifications'
      : route === '/profile/security'
        ? 'security'
        : 'profile';
    return (
      <ProfilePage
        user={user}
        token={token}
        initialTab={initialTab}
        onTabChange={(tab) => navigate(tab === 'profile' ? '/profile' : `/profile/${tab}`)}
        onUserUpdate={onUserUpdate}
        onUnreadCountChange={onUnreadCountChange}
      />
    );
  }
  if (activeItem.key === 'schedule') {
    return <SchedulePage user={user} token={token} />;
  }
  if (activeItem.key === 'grades' && user?.role === 'student') {
    return <GradesPage user={user} token={token} />;
  }
  if (activeItem.key === 'attendance' && user?.role === 'student') {
    return <AttendancePage user={user} token={token} />;
  }
  if (activeItem.key === 'analytics' && user?.role === 'student') {
    return <AnalyticsPage user={user} token={token} />;
  }
  if (user?.role === 'teacher' && ['attendance', 'stats', 'grades'].includes(activeItem.key)) {
    const section = activeItem.key === 'stats' ? 'statistics' : activeItem.key;
    return <TeacherPage token={token} section={section} />;
  }

  return (
    <EmptyPage
      title={activeItem.title}
      description="Этот раздел мы перепишем с нуля следующим блоком."
    />
  );
};

const Workspace = ({ user, token, route, navigate, onLogout, onUserUpdate }) => {
  const [isNavOpen, setIsNavOpen] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const navItems = getNavigationForRole(user?.role);
  const activeItem = navItems.find((item) => (
    item.route === route || (item.key === 'profile' && route.startsWith('/profile/'))
  )) || navItems[0];
  const displayName = getDisplayName(user);

  const refreshUnreadCount = useCallback(async () => {
    try {
      const payload = await api.getUnreadNotificationsCount(token);
      setUnreadCount(Math.max(0, Number(payload?.unread_count) || 0));
    } catch {
      // Notification availability must not block the rest of the workspace.
    }
  }, [token]);

  useEffect(() => {
    refreshUnreadCount();
    const interval = window.setInterval(refreshUnreadCount, 60000);
    const handleFocus = () => refreshUnreadCount();
    window.addEventListener('focus', handleFocus);
    return () => {
      window.clearInterval(interval);
      window.removeEventListener('focus', handleFocus);
    };
  }, [refreshUnreadCount]);

  useEffect(() => {
    if (!isNavOpen) return undefined;

    const previousOverflow = document.body.style.overflow;
    const handleKeyDown = (event) => {
      if (event.key === 'Escape') setIsNavOpen(false);
    };

    document.body.style.overflow = 'hidden';
    window.addEventListener('keydown', handleKeyDown);

    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [isNavOpen]);

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
            <button
              type="button"
              className="ui-sidebar-close"
              aria-label="Закрыть навигацию"
              onClick={() => setIsNavOpen(false)}
            >
              <NavIcon name="close" />
            </button>
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

          <div className="ui-topbar-actions">
            <button
              type="button"
              className="ui-notification-button"
              aria-label={unreadCount > 0 ? `Уведомления: ${unreadCount} непрочитанных` : 'Уведомления'}
              title="Уведомления"
              onClick={() => handleNavigate('/profile/notifications')}
            >
              <NavIcon name="notifications" />
              {unreadCount > 0 && (
                <span className="ui-notification-badge" aria-hidden="true">
                  {unreadCount > 99 ? '99+' : unreadCount}
                </span>
              )}
            </button>
          </div>
        </header>

        <main className="ui-main">
          {renderPage(
            activeItem,
            user,
            token,
            route,
            navigate,
            onUserUpdate,
            setUnreadCount,
            refreshUnreadCount
          )}
        </main>
      </div>
    </div>
  );
};

export default Workspace;
