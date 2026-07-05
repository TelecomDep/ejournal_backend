import React, { useEffect, useState } from 'react';
import api from './services/api';
import LoginPage from './components/LoginPage';
import Workspace from './ui/Workspace';
import useHashRoute from './hooks/useHashRoute';

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
    body = (
      <Workspace
        token={token}
        user={userData}
        route={route}
        navigate={navigate}
        onLogout={handleLogout}
      />
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
    </>
  );
}

export default App;
