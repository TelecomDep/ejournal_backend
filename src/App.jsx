import React, { useState, useEffect } from 'react';
import ProfileSquare from './components/ProfileSquare';
import ProfileDescription from './components/ProfileDescription';
import TeacherAccount from './components/TeacherAccount';
import LoginPage from './components/LoginPage';
import DataTable from './components/DataTable';
import StudentGradesPanel from './components/StudentGradesPanel';
import AttendanceGrid from './components/AttendanceGrid';
import api from './services/api';
import { sha256Hex } from './utils/hash';
import './App.css';

function App() {
  const [token, setToken] = useState(localStorage.getItem('token') || '');
  const [userData, setUserData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [attendanceHeatmapData, setAttendanceHeatmapData] = useState([]);
  const [attendanceYear] = useState(new Date().getFullYear());
  const [studentsData, setStudentsData] = useState([]);
  const [attendanceTableData, setAttendanceTableData] = useState([]);

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
        setError(api.getErrorMessage(err, 'Сессия истекла. Выполните вход заново.'));
      })
      .finally(() => setLoading(false));
  }, [token]);

  useEffect(() => {
    if (!token || !userData || userData.role !== 'student') {
      return;
    }

    // Загрузка данных для тепловой карты посещаемости
    api.getStudentAttendanceHeatmap(token, attendanceYear)
      .then((response) => {
        if (Array.isArray(response)) {
          setAttendanceHeatmapData(response);
        }
      })
      .catch((err) => {
        console.error('Attendance heatmap load failed:', err);
      });

    // Загрузка списка студентов
    api.getStudentsList(token)
      .then((response) => {
        if (Array.isArray(response)) {
          setStudentsData(response);
        }
      })
      .catch((err) => {
        console.error('Students list load failed:', err);
      });

    // Загрузка таблицы посещаемости
    api.getAttendanceTable(token)
      .then((response) => {
        if (Array.isArray(response)) {
          setAttendanceTableData(response);
        }
      })
      .catch((err) => {
        console.error('Attendance table load failed:', err);
      });
  }, [token, userData, attendanceYear]);

  const handleLogin = async (login, password) => {
    setError('');
    setLoading(true);

    try {
      const passwordHash = await sha256Hex(password);
      const response = await api.login(login, passwordHash);
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
      setError(api.getErrorMessage(err, 'Ошибка входа'));
      console.error('Login failed:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (login, password, registrationCode) => {
    setError('');
    setLoading(true);

    try {
      const passwordHash = await sha256Hex(password);
      await api.register(login, passwordHash, registrationCode);
      const response = await api.login(login, passwordHash);

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
      setError(api.getErrorMessage(err, 'Ошибка регистрации'));
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

  if (userData?.role === 'teacher') {
    return (
      <TeacherAccount 
        userData={userData} 
        onLogout={handleLogout}
        token={token}
      />
    );
  }

  return (
    <div className="contentContainer">
      <div className="logout-top-right">
        <button className="logout-button-top" onClick={handleLogout}>
          Выйти
        </button>
      </div>

      <div className="dashboard-grid">
        <aside className="dashboard-sidebar">
          <ProfileSquare userData={userData} />
        </aside>

        <main className="dashboard-main">
          <ProfileDescription userData={userData} />
          <AttendanceGrid attendanceData={attendanceHeatmapData} maxPerDay={8} />
          <StudentGradesPanel token={token} />
        </main>
      </div>

      <div className="tables-row">
        <DataTable data={studentsData} type="students" title="Список студентов" />
        <DataTable data={attendanceTableData} type="attendance" title="Таблица посещаемости" />
      </div>
    </div>
  );
}

export default App;