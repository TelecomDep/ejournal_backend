import React, { useEffect, useState } from 'react';
import api from './services/api';
import AttendanceHeatmap from './components/AttendanceHeatmap';
import LoginPage from './components/LoginPage';
import ProfileDescription from './components/ProfileDescription';
import ProfileSquare from './components/ProfileSquare';
import StudentGradesPanel from './components/StudentGradesPanel';
import TeacherAccount from './components/TeacherAccount';

function StudentDashboard({ token, userData, onLogout }) {
  const [attendanceHeatmapData, setAttendanceHeatmapData] = useState([]);

  useEffect(() => {
    let cancelled = false;
    api.getStudentAttendanceHeatmap(token, new Date().getFullYear())
      .then((data) => {
        if (!cancelled) {
          setAttendanceHeatmapData(data.items || []);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setAttendanceHeatmapData([]);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [token]);

  return (
    <div className="contentContainer">
      <div className="logout-top-right">
        <button className="logout-button-top" onClick={onLogout}>
          Выйти
        </button>
      </div>

      <div className="dashboard-grid">
        <aside className="dashboard-sidebar">
          <ProfileSquare userData={userData} />
        </aside>

        <main className="dashboard-main">
          <ProfileDescription userData={userData} />
          <AttendanceHeatmap attendanceData={attendanceHeatmapData} year={new Date().getFullYear()} />
          <StudentGradesPanel token={token} />
        </main>
      </div>
    </div>
  );
}

function App() {
  const [token, setToken] = useState(() => localStorage.getItem('ejournal_token') || '');
  const [userData, setUserData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleLogout = () => {
    localStorage.removeItem('ejournal_token');
    setToken('');
    setUserData(null);
    setError('');
  };

  useEffect(() => {
    if (!token) {
      return;
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

  if (!token) {
    return (
      <LoginPage
        onLogin={handleLogin}
        onRegister={handleRegister}
        loading={loading}
        error={error}
      />
    );
  }

  if (!userData) {
    return <div className="contentContainer">Загрузка...</div>;
  }

  if (userData.role === 'teacher') {
    return <TeacherAccount userData={userData} onLogout={handleLogout} token={token} />;
  }

  return <StudentDashboard token={token} userData={userData} onLogout={handleLogout} />;
}

export default App;
