import React, { useState, useEffect } from 'react';
import api from '../services/api';
import './TeacherAccount.css';

const TeacherAccount = ({ userData, onLogout, token }) => {
  const displayName = userData?.name || userData?.login || 'Преподаватель';
  
  // Session creation state
  const [sessionForm, setSessionForm] = useState({
    subjectId: 1,
    groupIds: [1],
    lessonName: 'Networks',
    expiresMinutes: 20
  });
  
  // Statistics state
  const [statsForm, setStatsForm] = useState({
    groupId: 1,
    subjectId: null
  });
  
  const [sessionLoading, setSessionLoading] = useState(false);
  const [statsLoading, setStatsLoading] = useState(false);
  const [sessionError, setSessionError] = useState('');
  const [statsError, setStatsError] = useState('');
  const [sessionResult, setSessionResult] = useState(null);
  const [statsResult, setStatsResult] = useState(null);
  const [activeTab, setActiveTab] = useState('sessions');

  // Handle session form input
  const handleSessionInputChange = (e) => {
    const { name, value } = e.target;
    if (name === 'groupIds') {
      // Parse comma-separated group IDs
      setSessionForm({
        ...sessionForm,
        [name]: value.split(',').map(id => parseInt(id.trim())).filter(id => !isNaN(id))
      });
    } else {
      setSessionForm({
        ...sessionForm,
        [name]: name === 'expiresMinutes' ? parseInt(value) : (name === 'subjectId' ? parseInt(value) : value)
      });
    }
  };

  // Handle stats form input
  const handleStatsInputChange = (e) => {
    const { name, value } = e.target;
    setStatsForm({
      ...statsForm,
      [name]: value === '' ? null : (name === 'subjectId' ? (value ? parseInt(value) : null) : parseInt(value))
    });
  };

  // Create attendance session
  const handleCreateSession = async (e) => {
    e.preventDefault();
    setSessionError('');
    setSessionResult(null);
    setSessionLoading(true);

    try {
      const response = await api.createAttendanceLink(
        token,
        sessionForm.subjectId,
        sessionForm.groupIds,
        sessionForm.lessonName,
        sessionForm.expiresMinutes
      );
      
      if (response?.result) {
        setSessionResult(response.result);
        // Reset form
        setSessionForm({
          subjectId: 1,
          groupIds: [1],
          lessonName: 'Networks',
          expiresMinutes: 20
        });
      } else {
        throw new Error(response?.error || 'Ошибка при создании сессии');
      }
    } catch (err) {
      setSessionError(err.response?.data?.error || err.message || 'Ошибка при создании сессии');
      console.error('Create session error:', err);
    } finally {
      setSessionLoading(false);
    }
  };

  // Get group statistics
  const handleGetStats = async (e) => {
    e.preventDefault();
    setStatsError('');
    setStatsResult(null);
    setStatsLoading(true);

    try {
      // This would need an API endpoint in the backend
      // For now, we'll use a placeholder that returns mock data
      const response = await api.getGroupStats(token, statsForm.groupId, statsForm.subjectId);
      
      if (response?.result) {
        setStatsResult(response.result);
      } else {
        throw new Error(response?.error || 'Ошибка при получении статистики');
      }
    } catch (err) {
      setStatsError(err.response?.data?.error || err.message || 'Ошибка при получении статистики');
      console.error('Get stats error:', err);
    } finally {
      setStatsLoading(false);
    }
  };

  return (
    <div className="teacher-account">
      <div className="teacher-header">
        <div className="teacher-info">
          <h1>EJournal Dashboard</h1>
          <p className="teacher-subtitle">
            Пользователь: <strong>{displayName}</strong> | Роль: <strong>{userData?.role || 'teacher'}</strong> | ID: <strong>{userData?.user_id || userData?.id}</strong>
          </p>
        </div>
        <div className="teacher-actions">
          <button className="update-profile-btn">Обновить профиль</button>
          <button className="logout-btn" onClick={onLogout}>Выйти</button>
        </div>
      </div>

      <div className="teacher-tabs">
        <button 
          className={`tab-btn ${activeTab === 'sessions' ? 'active' : ''}`}
          onClick={() => setActiveTab('sessions')}
        >
          Создать сессию
        </button>
        <button 
          className={`tab-btn ${activeTab === 'statistics' ? 'active' : ''}`}
          onClick={() => setActiveTab('statistics')}
        >
          Статистика группы
        </button>
      </div>

      {/* Create Session Tab */}
      {activeTab === 'sessions' && (
        <div className="teacher-section">
          <h2>Преподаватель: создать сессию</h2>
          
          <form onSubmit={handleCreateSession} className="teacher-form">
            <div className="form-group">
              <label htmlFor="subjectId">Subject ID</label>
              <input
                id="subjectId"
                type="number"
                name="subjectId"
                value={sessionForm.subjectId}
                onChange={handleSessionInputChange}
                min="1"
                required
              />
            </div>

            <div className="form-group">
              <label htmlFor="groupIds">Group IDs (через запятую)</label>
              <input
                id="groupIds"
                type="text"
                name="groupIds"
                value={sessionForm.groupIds.join(', ')}
                onChange={handleSessionInputChange}
                placeholder="1, 2, 3"
                required
              />
            </div>

            <div className="form-group">
              <label htmlFor="lessonName">Lesson name</label>
              <input
                id="lessonName"
                type="text"
                name="lessonName"
                value={sessionForm.lessonName}
                onChange={handleSessionInputChange}
                placeholder="Networks"
                required
              />
            </div>

            <div className="form-group">
              <label htmlFor="expiresMinutes">Expires minutes</label>
              <input
                id="expiresMinutes"
                type="number"
                name="expiresMinutes"
                value={sessionForm.expiresMinutes}
                onChange={handleSessionInputChange}
                min="1"
                max="1440"
                required
              />
            </div>

            <button type="submit" className="submit-btn" disabled={sessionLoading}>
              {sessionLoading ? 'Создание...' : 'Создать ссылку'}
            </button>

            <button type="button" className="copy-btn">Копировать ссылку</button>
          </form>

          {sessionError && (
            <div className="error-message">
              {sessionError}
            </div>
          )}

          {sessionResult && (
            <div className="session-result">
              <h3>Сессия создана успешно!</h3>
              
              <div className="result-item">
                <label>Join URL:</label>
                <div className="result-value clickable">
                  {sessionResult.join_url || 'Не получена'}
                </div>
              </div>

              <div className="result-item">
                <label>Invite token:</label>
                <div className="result-value clickable">
                  {sessionResult.invite_token || 'Не получена'}
                </div>
              </div>

              {sessionResult.session_id && (
                <div className="result-item">
                  <label>Session ID:</label>
                  <div className="result-value">
                    {sessionResult.session_id}
                  </div>
                </div>
              )}

              {sessionResult.created_at && (
                <div className="result-item">
                  <label>Создано:</label>
                  <div className="result-value">
                    {new Date(sessionResult.created_at).toLocaleString('ru-RU')}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Statistics Tab */}
      {activeTab === 'statistics' && (
        <div className="teacher-section">
          <h2>Преподаватель: статистика группы</h2>

          <form onSubmit={handleGetStats} className="teacher-form">
            <div className="form-group">
              <label htmlFor="statGroupId">Group ID</label>
              <input
                id="statGroupId"
                type="number"
                name="groupId"
                value={statsForm.groupId}
                onChange={handleStatsInputChange}
                min="1"
                required
              />
            </div>

            <div className="form-group">
              <label htmlFor="statSubjectId">Subject ID (опционально)</label>
              <input
                id="statSubjectId"
                type="number"
                name="subjectId"
                value={statsForm.subjectId || ''}
                onChange={handleStatsInputChange}
                min="1"
                placeholder="Оставьте пустым для всех предметов"
              />
            </div>

            <button type="submit" className="submit-btn" disabled={statsLoading}>
              {statsLoading ? 'Загрузка...' : 'Загрузить посещаемость'}
            </button>
          </form>

          {statsError && (
            <div className="error-message">
              {statsError}
            </div>
          )}

          {statsResult && (
            <div className="stats-result">
              <h3>Статистика посещаемости</h3>
              <table className="stats-table">
                <thead>
                  <tr>
                    <th>student_id</th>
                    <th>student_name</th>
                    <th>attended</th>
                    <th>total</th>
                    <th>%</th>
                  </tr>
                </thead>
                <tbody>
                  {statsResult.students && statsResult.students.length > 0 ? (
                    statsResult.students.map((student, idx) => (
                      <tr key={idx}>
                        <td>{student.student_id}</td>
                        <td>{student.student_name}</td>
                        <td>{student.attended}</td>
                        <td>{student.total}</td>
                        <td>{student.percentage ? (student.percentage * 100).toFixed(1) + '%' : 'N/A'}</td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan="5">Нет данных</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default TeacherAccount;
