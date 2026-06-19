import React from 'react';
import './TeacherPanels.css';

const TeacherStats = ({ data, studentsData }) => {
  const stats = data || {
    totalStudents: 30,
    averageAttendance: 87,
    averagePerformance: 78,
    topStudents: 5,
    riskStudents: 3
  };

  return (
    <div className="teacher-panel">
      <h2 className="panel-title">Статистика группы</h2>
      
      <div className="stats-grid">
        <div className="stat-card">
          <div className="pfp-block-inner-dark">
            <div className="stat-icon">📊</div>
            <div className="stat-value">{stats.totalStudents}</div>
            <div className="stat-label">Всего студентов</div>
          </div>
        </div>
        
        <div className="stat-card">
          <div className="pfp-block-inner-dark">
            <div className="stat-icon">✅</div>
            <div className="stat-value">{stats.averageAttendance}%</div>
            <div className="stat-label">Средняя посещаемость</div>
          </div>
        </div>
        
        <div className="stat-card">
          <div className="pfp-block-inner-dark">
            <div className="stat-icon">📈</div>
            <div className="stat-value">{stats.averagePerformance}%</div>
            <div className="stat-label">Средняя успеваемость</div>
          </div>
        </div>
        
        <div className="stat-card">
          <div className="pfp-block-inner-dark">
            <div className="stat-icon">⭐</div>
            <div className="stat-value">{stats.topStudents}</div>
            <div className="stat-label">Отличников</div>
          </div>
        </div>
        
        <div className="stat-card">
          <div className="pfp-block-inner-dark">
            <div className="stat-icon">⚠️</div>
            <div className="stat-value">{stats.riskStudents}</div>
            <div className="stat-label">В зоне риска</div>
          </div>
        </div>
      </div>
      
      <div className="panel-card full-width">
        <div className="pfp-block-inner-dark">
          <h3>Успеваемость по месяцам</h3>
          <div className="chart-placeholder">
            {[65, 72, 78, 82, 85, 88].map((value, index) => (
              <div key={index} className="chart-bar-container">
                <div className="chart-bar-label">Месяц {index + 1}</div>
                <div className="chart-bar-wrapper">
                  <div 
                    className="chart-bar" 
                    style={{ width: `${value}%` }}
                  />
                  <span className="chart-value">{value}%</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

export default TeacherStats;