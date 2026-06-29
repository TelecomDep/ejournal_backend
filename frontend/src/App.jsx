import React, { useEffect, useState } from 'react';
import api from './services/api';
import LoginPage from './components/LoginPage';
import AppShell from './components/AppShell';
import ThemeToggle from './components/ThemeToggle';
import TeacherAccount from './components/TeacherAccount';
import StaffDashboard from './components/StaffDashboard';
import ProfilePage from './pages/ProfilePage';
import AttendancePage from './pages/AttendancePage';
import GradesPage from './pages/GradesPage';
import useHashRoute from './hooks/useHashRoute';

const STUDENT_NAV = [
  { key: 'profile', label: 'Профиль', route: '/profile' },
  { key: 'attendance', label: 'Посещаемость', route: '/attendance' },
  { key: 'grades', label: 'Оценки', route: '/grades' }
];

const TEACHER_NAV = [
  { key: 'profile', label: 'Профиль', route: '/profile' },
  { key: 'attendance', label: 'Посещаемость', route: '/teacher/attendance' },
  { key: 'stats', label: 'Статистика группы', route: '/teacher/stats' },
  { key: 'grades', label: 'Оценки', route: '/teacher/grades' }
];

const TEACHER_SECTION = {
  attendance: 'attendance',
  stats: 'statistics',
  grades: 'grades'
};

// Надзорные роли (зав. кафедрой / декан / админ) видят единую сводку по охвату.
const STAFF_NAV = [
  { key: 'profile', label: 'Профиль', route: '/profile' },
  { key: 'overview', label: 'Сводка', route: '/staff/overview' }
];

const STAFF_ROLES = ['admin', 'head', 'dean'];

const ROLE_LABELS = {
  admin: 'Администратор',
  dean: 'Декан',
  head: 'Зав. кафедрой',
  teacher: 'Преподаватель',
  student: 'Студент'
};

const getJoinInviteToken = () => {
  const hash = window.location.hash || '';
  if (!hash.startsWith('#/attendance/join')) {
    return '';
  }
  const queryStart = hash.indexOf('?');
  if (queryStart === -1) {
    return '';
  }
  const params = new URLSearchParams(hash.slice(queryStart + 1));
  return params.get('token') || '';
};

function App() {
  const [token, setToken] = useState(() => localStorage.getItem('ejournal_token') || '');
  const [userData, setUserData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [pendingInvite, setPendingInvite] = useState(
    () => getJoinInviteToken() || localStorage.getItem('ejournal_pending_invite') || ''
  );
  const [attendanceNotice, setAttendanceNotice] = useState(null);
  const { route, navigate } = useHashRoute();

  const handleLogout = () => {
    localStorage.removeItem('ejournal_token');
    setToken('');
    setUserData(null);
    setError('');
  };

  useEffect(() => {
    if (!token) {
      return undefined;
    }

    let cancelled = false;
    setLoading(true);
    api.getProfile(token)
      .then((profile) => {
        if (!cancelled) {
          setUserData(profile);
          setError('');
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(api.getErrorMessage(err, 'Не удалось загрузить профиль'));
          handleLogout();
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [token]);

  useEffect(() => {
    const invite = getJoinInviteToken();
    if (invite) {
      localStorage.setItem('ejournal_pending_invite', invite);
      setPendingInvite(invite);
    }
  }, [route]);

  useEffect(() => {
    if (!token || !userData || !pendingInvite) {
      return undefined;
    }

    let cancelled = false;
    api.confirmAttendance(token, pendingInvite)
      .then(() => {
        if (!cancelled) {
          setAttendanceNotice({ type: 'success', text: 'Посещение успешно отмечено!' });
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setAttendanceNotice({
            type: 'error',
            text: api.getErrorMessage(err, 'Не удалось отметить посещение')
          });
        }
      })
      .finally(() => {
        if (cancelled) {
          return;
        }
        localStorage.removeItem('ejournal_pending_invite');
        setPendingInvite('');
        navigate(userData.role === 'teacher' ? '/teacher/attendance' : '/attendance');
      });

    return () => {
      cancelled = true;
    };
  }, [token, userData, pendingInvite]);

  const handleLogin = async (login, password) => {
    setLoading(true);
    setError('');
    try {
      const result = await api.login(login, password);
      localStorage.setItem('ejournal_token', result.token);
      setToken(result.token);
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось войти'));
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (login, password, registrationCode) => {
    setLoading(true);
    setError('');
    try {
      const result = await api.register(login, password, registrationCode);
      localStorage.setItem('ejournal_token', result.token);
      setToken(result.token);
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось зарегистрироваться'));
    } finally {
      setLoading(false);
    }
  };

  const handleAvatarUpdated = (url) => {
    setUserData((prev) => (prev ? { ...prev, avatar: url } : prev));
  };

  const handleUserDataUpdated = (fields) => {
    setUserData((prev) => (prev ? { ...prev, ...fields } : prev));
  };

  let body;

  if (!token) {
    body = (
      <LoginPage
        onLogin={handleLogin}
        onRegister={handleRegister}
        loading={loading}
        error={error}
      />
    );
  } else if (!userData) {
    body = <div className="app-loading">Загрузка…</div>;
  } else {
    const isTeacher = userData.role === 'teacher';
    const isStaff = STAFF_ROLES.includes(userData.role);
    const navItems = isStaff ? STAFF_NAV : isTeacher ? TEACHER_NAV : STUDENT_NAV;
    const activeItem = navItems.find((item) => item.route === route) || navItems[0];
    const displayName = userData.teacher_name || userData.name || userData.login || 'Пользователь';
    const roleLabel = ROLE_LABELS[userData.role] || 'Студент';

    let page;
    if (activeItem.key === 'profile') {
      page = (
        <ProfilePage
          token={token}
          userData={userData}
          onAvatarUpdated={handleAvatarUpdated}
          onUserDataUpdated={handleUserDataUpdated}
        />
      );
    } else if (isStaff) {
      page = <StaffDashboard token={token} />;
    } else if (isTeacher) {
      page = <TeacherAccount userData={userData} token={token} section={TEACHER_SECTION[activeItem.key]} />;
    } else if (activeItem.key === 'attendance') {
      page = <AttendancePage token={token} />;
    } else if (activeItem.key === 'grades') {
      page = <GradesPage token={token} />;
    }

    body = (
      <AppShell
        title="СибГУТИ"
        subtitle={`${displayName} · ${roleLabel}`}
        navItems={navItems}
        activeKey={activeItem.key}
        onNavigate={(key) => {
          const next = navItems.find((item) => item.key === key);
          if (next) {
            navigate(next.route);
          }
        }}
        user={displayName}
        onLogout={handleLogout}
      >
        {page}
      </AppShell>
    );
  }

  return (
    <>
      {attendanceNotice && (
        <div
          className={`attendance-notice attendance-notice--${attendanceNotice.type}`}
          role="status"
          onClick={() => setAttendanceNotice(null)}
        >
          <span>{attendanceNotice.text}</span>
          <button type="button" aria-label="Закрыть" onClick={() => setAttendanceNotice(null)}>×</button>
        </div>
      )}
      {body}
      <ThemeToggle />
    </>
  );
}

export default App;
