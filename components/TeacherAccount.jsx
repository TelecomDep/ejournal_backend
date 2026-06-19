import React, { useState, useEffect } from 'react';
import ProfileSquare from './ProfileSquare';
import TeacherPanel from './TeacherPanel';
import TeacherAttendance from './TeacherAttendance';
import TeacherGrades from './TeacherGrades';
import TeacherStats from './TeacherStats';
import DataTable from './DataTable';
import InfoCard from './InfoCard';
import useWebSocket from '../hooks/useWebSocket';
import './TeacherAccount.css';

const TeacherAccount = ({ userData, onLogout }) => {
  const [activeTab, setActiveTab] = useState('attendance');
  const [studentsData, setStudentsData] = useState([]);
  const [attendanceData, setAttendanceData] = useState([]);
  const [statsData, setStatsData] = useState(null);
  const [gradesData, setGradesData] = useState(null);
  
  const { sendMessage, lastMessage, connectionStatus } = useWebSocket('ws://localhost:8080/ws');

  useEffect(() => {
    if (connectionStatus === 'connected') {
      sendMessage({
        action: 'getTeacherData',
        data: { teacherId: userData?.teacher_id }
      });
      
      sendMessage({
        action: 'getStudentsList',
        data: { groupId: userData?.group_id }
      });
      
      sendMessage({
        action: 'getAttendanceStats',
        data: { teacherId: userData?.teacher_id }
      });
    }
  }, [connectionStatus, sendMessage, userData]);

  useEffect(() => {
    if (lastMessage) {
      try {
        const response = JSON.parse(lastMessage);
        if (response.ok) {
          switch (response.action) {
            case 'getTeacherData':
              // Обработка данных преподавателя
              break;
            case 'getStudentsList':
              setStudentsData(response.result || []);
              break;
            case 'getAttendanceStats':
              setAttendanceData(response.result || []);
              break;
            case 'getStatsData':
              setStatsData(response.result);
              break;
            case 'getGradesData':
              setGradesData(response.result);
              break;
            default:
              break;
          }
        }
      } catch (error) {
        console.error('Parse error:', error);
      }
    }
  }, [lastMessage]);

  const statsCards = [
    { 
      id: 1, 
      title: 'Студентов', 
      value: studentsData.length.toString(), 
      color: '#6B8ED4' 
    },
    { 
      id: 2, 
      title: 'Посещаемость', 
      value: statsData?.attendanceRate || '92%', 
      color: '#5E3E9F' 
    },
    { 
      id: 3, 
      title: 'Успеваемость', 
      value: statsData?.performanceRate || '85%', 
      color: '#4A7BC8' 
    }
  ];

  return (
    <div className="teacher-container">
      {/* Профиль преподавателя */}
      <div className="teacher-profile-section">
        <ProfileSquare userData={userData} />
        <div className="teacher-info-block">
          <div className="pfp-block-inner">
            <h1 className="teacher-name">
              {userData?.teacher_name || userData?.name || 'Преподаватель'}
            </h1>
            <div className="teacher-details-grid">
              <div className="detail-item">
                <span className="detail-label">Должность</span>
                <span className="detail-value">
                  {userData?.position || 'Преподаватель'}
                </span>
              </div>
              <div className="detail-item">
                <span className="detail-label">Кафедра</span>
                <span className="detail-value">
                  {userData?.department || 'Не указана'}
                </span>
              </div>
              <div className="detail-item">
                <span className="detail-label">Email</span>
                <span className="detail-value">
                  {userData?.email || 'Не указан'}
                </span>
              </div>
              <div className="detail-item">
                <span className="detail-label">Телефон</span>
                <span className="detail-value">
                  {userData?.phone || 'Не указан'}
                </span>
              </div>
            </div>
            <button className="logout-btn" onClick={onLogout}>
              Выйти из системы
            </button>
          </div>
        </div>
      </div>

      {/* Навигация */}
      <div className="teacher-nav">
        <button 
          className={`nav-btn ${activeTab === 'attendance' ? 'active' : ''}`}
          onClick={() => setActiveTab('attendance')}
        >
          Посещаемость
        </button>
        <button 
          className={`nav-btn ${activeTab === 'grades' ? 'active' : ''}`}
          onClick={() => setActiveTab('grades')}
        >
          Оценки
        </button>
        <button 
          className={`nav-btn ${activeTab === 'stats' ? 'active' : ''}`}
          onClick={() => setActiveTab('stats')}
        >
          Статистика
        </button>
        <button 
          className={`nav-btn ${activeTab === 'students' ? 'active' : ''}`}
          onClick={() => setActiveTab('students')}
        >
          Студенты
        </button>
      </div>

      {/* Карточки статистики */}
      <div className="teacher-stats-row">
        {statsCards.map((card) => (
          <InfoCard
            key={card.id}
            title={card.title}
            value={card.value}
            color={card.color}
          />
        ))}
      </div>

      {/* Контент вкладок */}
      <div className="teacher-content">
        {activeTab === 'attendance' && (
          <TeacherAttendance 
            data={attendanceData}
            sendMessage={sendMessage}
            teacherId={userData?.teacher_id}
          />
        )}
        
        {activeTab === 'grades' && (
          <TeacherGrades 
            data={gradesData}
            sendMessage={sendMessage}
            teacherId={userData?.teacher_id}
          />
        )}
        
        {activeTab === 'stats' && (
          <TeacherStats 
            data={statsData}
            studentsData={studentsData}
          />
        )}
        
        {activeTab === 'students' && (
          <DataTable 
            data={studentsData}
            type="students"
            title="Список студентов группы"
          />
        )}
      </div>
    </div>
  );
};

export default TeacherAccount;