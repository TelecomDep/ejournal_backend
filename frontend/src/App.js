import React, { useEffect, useState } from 'react';
import LoginCard from './components/LoginCard';
import TeacherPanel from './components/TeacherPanel';
import StudentPanel from './components/StudentPanel';
import api from './services/api';
import { sha256Hex } from './utils/hash';

function App() {
  const [token, setToken] = useState(localStorage.getItem('token') || '');
  const [profile, setProfile] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const [lastLink, setLastLink] = useState(null);
  const [stats, setStats] = useState([]);
  const [confirmResult, setConfirmResult] = useState(null);

  useEffect(() => {
    if (!token) {
      setProfile(null);
      return;
    }

    setLoading(true);
    api
      .getProfile(token)
      .then((data) => {
        setProfile({
          user_id: data.user_id,
          login: data.login,
          role: data.role
        });
      })
      .catch((e) => {
        localStorage.removeItem('token');
        setToken('');
        setError(e.backend?.error || e.message || 'Не удалось получить профиль');
      })
      .finally(() => setLoading(false));
  }, [token]);

  const resetMessages = () => {
    setError('');
    setNotice('');
  };

  const handleLogin = async (login, passwordRaw, roleHash) => {
    resetMessages();
    setLoading(true);
    try {
      const passwordHash = await sha256Hex(passwordRaw);
      const data = await api.login(login, passwordHash, roleHash);
      localStorage.setItem('token', data.token);
      setToken(data.token);
      setNotice('Вход выполнен (SHA-256 + role_hash).');
    } catch (e) {
      setError(e.backend?.error || e.message || 'Ошибка входа');
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (login, passwordRaw, roleHash) => {
    resetMessages();
    setLoading(true);
    try {
      const passwordHash = await sha256Hex(passwordRaw);
      await api.register(login, passwordHash, roleHash);
      const auth = await api.login(login, passwordHash, roleHash);
      localStorage.setItem('token', auth.token);
      setToken(auth.token);
      setNotice('Регистрация и вход выполнены (SHA-256 + role_hash).');
    } catch (e) {
      setError(e.backend?.error || e.message || 'Ошибка регистрации');
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem('token');
    setToken('');
    setProfile(null);
    setLastLink(null);
    setStats([]);
    setConfirmResult(null);
    resetMessages();
  };

  const refreshProfile = async () => {
    resetMessages();
    setLoading(true);
    try {
      const data = await api.getProfile(token);
      setProfile(data);
      setNotice('Профиль обновлен.');
    } catch (e) {
      setError(e.backend?.error || e.message || 'Не удалось обновить профиль');
    } finally {
      setLoading(false);
    }
  };

  const createAttendanceLink = async (payload) => {
    resetMessages();
    setLoading(true);
    try {
      const data = await api.createAttendanceLink(token, payload);
      setLastLink(data);
      setNotice('Ссылка посещаемости создана.');
    } catch (e) {
      setError(e.backend?.error || e.message || 'Не удалось создать ссылку');
    } finally {
      setLoading(false);
    }
  };

  const loadGroupStats = async (payload) => {
    resetMessages();
    setLoading(true);
    try {
      const data = await api.getAttendanceByGroup(token, payload);
      setStats(Array.isArray(data.stats) ? data.stats : []);
      setNotice('Статистика загружена.');
    } catch (e) {
      setError(e.backend?.error || e.message || 'Не удалось загрузить статистику');
    } finally {
      setLoading(false);
    }
  };

  const confirmAttendance = async (inviteToken) => {
    resetMessages();
    setLoading(true);
    try {
      const data = await api.confirmAttendance(token, inviteToken);
      setConfirmResult(data);
      setNotice('Посещаемость подтверждена.');
    } catch (e) {
      setError(e.backend?.error || e.message || 'Не удалось подтвердить посещаемость');
    } finally {
      setLoading(false);
    }
  };

  if (!profile) {
    return <LoginCard onLogin={handleLogin} onRegister={handleRegister} loading={loading} error={error} />;
  }

  return (
    <main className="container">
      <header className="card header-card">
        <div>
          <h1>EJournal Dashboard</h1>
          <p className="muted">
            Пользователь: <strong>{profile.login}</strong> | Роль: <strong>{profile.role}</strong> | ID: <strong>{profile.user_id}</strong>
          </p>
        </div>

        <div className="row gap-sm">
          <button className="btn" onClick={refreshProfile} disabled={loading}>Обновить профиль</button>
          <button className="btn btn-danger" onClick={handleLogout}>Выйти</button>
        </div>
      </header>

      {error && <div className="card error-box">{error}</div>}
      {notice && <div className="card notice-box">{notice}</div>}

      {profile.role === 'teacher' && (
        <TeacherPanel
          onCreateLink={createAttendanceLink}
          onLoadStats={loadGroupStats}
          lastLink={lastLink}
          stats={stats}
          loading={loading}
        />
      )}

      {profile.role === 'student' && (
        <StudentPanel
          onConfirmAttendance={confirmAttendance}
          loading={loading}
          confirmResult={confirmResult}
        />
      )}
    </main>
  );
}

export default App;
