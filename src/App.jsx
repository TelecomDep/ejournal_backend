import React, { useState, useEffect } from 'react';
import ProfileSquare from './components/ProfileSquare';
import ProfileDescription from './components/ProfileDescription';
import PersonalAccount from './components/PersonalAccount';
import TeacherAccount from './components/TeacherAccount';
import LoginPage from './components/LoginPage';
import DataTable from './components/DataTable';
import api from './services/api';
import './App.css';

const sampleStudents = [
  { id: 1, name: 'Иван Иванов', group: 'A-101', subject1: 'Математика', subject2: 'Физика', subject3: 'Информатика', subject4: 'Английский' },
  { id: 2, name: 'Мария Петрова', group: 'A-103', subject1: 'Русский', subject2: 'История', subject3: 'Химия', subject4: 'Литература' },
  { id: 3, name: 'Сергей Кузнецов', group: 'B-205', subject1: 'География', subject2: 'Биология', subject3: 'Физика', subject4: 'Труд' }
];

const sampleAttendance = [
  { id: 1, name: 'Иван Иванов', date1: '✓', date2: '✓', date3: '✗', date4: '✓', date5: '✓' },
  { id: 2, name: 'Мария Петрова', date1: '✓', date2: '✓', date3: '✓', date4: '✓', date5: '✓' },
  { id: 3, name: 'Сергей Кузнецов', date1: '✗', date2: '✓', date3: '✓', date4: '✗', date5: '✓' }
];

function App() {
  const [token, setToken] = useState(localStorage.getItem('token') || '');
  const [userData, setUserData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!token) {
      setUserData(null);
      return;
    }

    setLoading(true);
    api.getProfile(token)
      .then((response) => {
        if (response?.user_id || response?.login) {
          setUserData({
            ...response,
            name: response.name || response.login
          });
        } else {
          throw new Error(response?.error || 'Не удалось получить профиль');
        }
      })
      .catch((err) => {
        console.error('Profile load failed:', err);
        localStorage.removeItem('token');
        setToken('');
        setError('Сессия истекла. Выполните вход заново.');
      })
      .finally(() => setLoading(false));
  }, [token]);

  const handleLogin = async (login, password, role) => {
    setError('');
    setLoading(true);

    try {
      const response = await api.login(login, password);
      if (response?.token) {
        localStorage.setItem('token', response.token);
        setToken(response.token);
        setUserData({
          ...response,
          name: response.login
        });
      } else {
        throw new Error(response?.error || 'Не удалось войти');
      }
    } catch (err) {
      setError(err.response?.data?.error || err.message || 'Ошибка входа');
      console.error('Login failed:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (login, password, role, inviteCode) => {
    setError('');
    setLoading(true);

    try {
      const response = await api.register(login, password, role, inviteCode);
      if (response?.token) {
        localStorage.setItem('token', response.token);
        setToken(response.token);
        setUserData({
          ...response,
          name: response.login
        });
      } else {
        throw new Error(response?.error || 'Не удалось зарегистрироваться');
      }
    } catch (err) {
      setError(err.response?.data?.error || err.message || 'Ошибка регистрации');
      console.error('Register failed:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem('token');
    setToken('');
    setUserData(null);
    setError('');
  };

  if (!userData) {
    return <LoginPage onLogin={handleLogin} onRegister={handleRegister} loading={loading} error={error} />;
  }

  // Render teacher account page if user is a teacher
  if (userData?.role === 'teacher') {
    return (
      <TeacherAccount 
        userData={userData} 
        onLogout={handleLogout}
        token={token}
      />
    );
  }

  // Render student account page
  return (
    <div className="contentContainer">
      <div className="page-header">
        <ProfileSquare userData={userData} />
        <ProfileDescription userData={userData} />
      </div>

      <PersonalAccount userData={userData} onLogout={handleLogout} />

      <div className="tables-row">
        <DataTable data={sampleStudents} type="students" title="Список студентов" />
        <DataTable data={sampleAttendance} type="attendance" title="Таблица посещаемости" />
      </div>
    </div>
  );
}

export default App;
