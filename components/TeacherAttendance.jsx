import React, { useState } from 'react';
import './TeacherPanels.css';

const TeacherAttendance = ({ data, sendMessage, teacherId }) => {
  const [lessonName, setLessonName] = useState('');
  const [subjectId, setSubjectId] = useState('1');
  const [expiresMinutes, setExpiresMinutes] = useState('30');
  const [attendanceLink, setAttendanceLink] = useState(null);
  const [selectedGroup, setSelectedGroup] = useState('1');

  const handleCreateLink = () => {
    sendMessage({
      action: 'createAttendanceLink',
      data: {
        teacher_id: teacherId,
        subject_id: parseInt(subjectId),
        lesson_name: lessonName,
        expires_minutes: parseInt(expiresMinutes),
        group_id: parseInt(selectedGroup)
      }
    });
    
    // Имитация получения ссылки
    setAttendanceLink({
      url: `https://attendance.example.com/join/${Math.random().toString(36).substr(2, 9)}`,
      lesson_id: Math.floor(Math.random() * 1000),
      expires_at: new Date(Date.now() + parseInt(expiresMinutes) * 60000).toLocaleString()
    });
  };

  const copyToClipboard = () => {
    if (attendanceLink?.url) {
      navigator.clipboard.writeText(attendanceLink.url);
    }
  };

  return (
    <div className="teacher-panel">
      <h2 className="panel-title">Управление посещаемостью</h2>
      
      <div className="panel-grid">
        <div className="panel-card">
          <div className="pfp-block-inner-dark">
            <h3>Создать ссылку для отметки</h3>
            
            <div className="form-group">
              <label>Предмет ID</label>
              <input 
                type="number" 
                value={subjectId}
                onChange={(e) => setSubjectId(e.target.value)}
                className="dark-input"
                min="1"
              />
            </div>
            
            <div className="form-group">
              <label>Название занятия</label>
              <input 
                type="text" 
                value={lessonName}
                onChange={(e) => setLessonName(e.target.value)}
                className="dark-input"
                placeholder="Лекция по программированию"
              />
            </div>
            
            <div className="form-row">
              <div className="form-group">
                <label>Группа ID</label>
                <select 
                  value={selectedGroup}
                  onChange={(e) => setSelectedGroup(e.target.value)}
                  className="dark-input"
                >
                  <option value="1">Группа 101</option>
                  <option value="2">Группа 102</option>
                  <option value="3">Группа 103</option>
                </select>
              </div>
              
              <div className="form-group">
                <label>Действует (мин)</label>
                <input 
                  type="number" 
                  value={expiresMinutes}
                  onChange={(e) => setExpiresMinutes(e.target.value)}
                  className="dark-input"
                  min="5"
                  max="180"
                />
              </div>
            </div>
            
            <button onClick={handleCreateLink} className="action-btn">
              Создать ссылку
            </button>
            
            {attendanceLink && (
              <div className="link-result">
                <div className="link-url">{attendanceLink.url}</div>
                <div className="link-info">
                  <span>ID занятия: {attendanceLink.lesson_id}</span>
                  <span>Истекает: {attendanceLink.expires_at}</span>
                </div>
                <button onClick={copyToClipboard} className="copy-btn">
                  Копировать ссылку
                </button>
              </div>
            )}
          </div>
        </div>
        
        <div className="panel-card">
          <div className="pfp-block-inner-dark">
            <h3>История посещений</h3>
            <div className="attendance-list">
              {data && data.length > 0 ? (
                data.map((record, index) => (
                  <div key={index} className="attendance-item">
                    <div className="attendance-info">
                      <span className="attendance-name">{record.student_name || `Студент ${index + 1}`}</span>
                      <span className="attendance-date">{record.date || '01.01.2024'}</span>
                    </div>
                    <span className={`attendance-status ${record.present ? 'present' : 'absent'}`}>
                      {record.present ? '✓' : '✗'}
                    </span>
                  </div>
                ))
              ) : (
                <div className="empty-state">Нет записей о посещаемости</div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TeacherAttendance;