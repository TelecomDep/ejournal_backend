import React, { useState, useEffect, useCallback } from 'react';
import ProfileSquare from './components/ProfileSquare';
import ProfileDescription from './components/ProfileDescription';
import SlidersUnder from './components/SlidersUnder';
import Calendar from './components/Calendar';
import InfoCard from './components/InfoCard';
import DataTable from './components/DataTable';
import PersonalAccount from './components/PersonalAccount';
import TeacherAccount from './components/TeacherAccount';
import LoginPage from './components/LoginPage';
import useWebSocket from './hooks/useWebSocket';
import './App.css';

function App() {
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [userRole, setUserRole] = useState('student'); // 'student' или 'teacher'
  const [userData, setUserData] = useState(null);
  const [studentsData, setStudentsData] = useState([]);
  const [attendanceData, setAttendanceData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  
  const { sendMessage, lastMessage, connectionStatus } = useWebSocket('ws://localhost:8080/ws');

  // Эффект для запроса данных после подключения WebSocket
  useEffect(() => {
    if (connectionStatus === 'connected' && isLoggedIn) {
      // Запрос данных пользователя
      sendMessage({
        action: 'getUserData',
        data: { userId: userData?.user_id }
      });
      
      // Запрос списка студентов
      sendMessage({
        action: 'getStudentsList',
        data: { groupId: userData?.group_id }
      });
      
      // Запрос посещаемости
      sendMessage({
        action: 'getAttendance',
        data: { userId: userData?.user_id }
      });

      // Для преподавателя запрашиваем дополнительные данные
      if (userRole === 'teacher') {
        sendMessage({
          action: 'getTeacherStats',
          data: { teacherId: userData?.teacher_id }
        });
        
        sendMessage({
          action: 'getTeacherGrades',
          data: { teacherId: userData?.teacher_id }
        });
      }
    }
  }, [connectionStatus, isLoggedIn, userData?.user_id, userData?.teacher_id]);

  // Обработка входящих сообщений WebSocket
  useEffect(() => {
    if (lastMessage) {
      try {
        const response = JSON.parse(lastMessage);
        
        if (response.ok) {
          switch (response.action) {
            case 'getUserData':
              setUserData(prev => ({
                ...prev,
                ...response.result
              }));
              break;
              
            case 'getStudentsList':
              setStudentsData(response.result || []);
              break;
              
            case 'getAttendance':
              setAttendanceData(response.result || []);
              break;
              
            case 'getTeacherStats':
              setUserData(prev => ({
                ...prev,
                stats: response.result
              }));
              break;
              
            case 'getTeacherGrades':
              setUserData(prev => ({
                ...prev,
                grades: response.result
              }));
              break;
              
            default:
              console.log('Unhandled action:', response.action);
              break;
          }
        } else {
          console.error('Server error:', response.error);
          setError(response.error || 'Ошибка сервера');
        }
      } catch (parseError) {
        console.error('Failed to parse message:', parseError);
      }
    }
  }, [lastMessage]);

  // Обработка входа
  const handleLogin = useCallback((login, password) => {
    setLoading(true);
    setError('');
    
    // Имитация запроса к серверу
    setTimeout(() => {
      // В реальном приложении здесь был бы API-запрос
      const mockUserData = {
        name: 'Иван Петров',
        login: login,
        role: userRole,
        group: userRole === 'student' ? 'A-101' : undefined,
        group_name: userRole === 'student' ? 'Группа A-101' : undefined,
        email: 'ivan@example.com',
        phone: '+7 (999) 123-45-67',
        user_id: Math.floor(Math.random() * 1000),
        teacher_id: userRole === 'teacher' ? Math.floor(Math.random() * 1000) : undefined,
        department: userRole === 'teacher' ? 'Информатика' : undefined,
        position: userRole === 'teacher' ? 'Доцент' : undefined,
        avatar: null,
        status: 'В сети'
      };
      
      setUserData(mockUserData);
      setIsLoggedIn(true);
      setLoading(false);
    }, 1000);
  }, [userRole]);

  // Обработка регистрации
  const handleRegister = useCallback((login, password, registrationCode) => {
    setLoading(true);
    setError('');
    
    // Имитация регистрации
    setTimeout(() => {
      // В реальном приложении здесь был бы API-запрос
      handleLogin(login, password);
    }, 1500);
  }, [handleLogin]);

  // Обработка выхода
  const handleLogout = useCallback(() => {
    setIsLoggedIn(false);
    setUserData(null);
    setStudentsData([]);
    setAttendanceData([]);
    setError('');
  }, []);

  // Данные для инфо-карточек
  const cardData = [
    { 
      id: 1, 
      title: 'Успеваемость', 
      value: userData?.performance || '85%', 
      color: '#6B8ED4' 
    },
    { 
      id: 2, 
      title: 'Посещаемость', 
      value: userData?.attendance || '92%', 
      color: '#5E3E9F' 
    },
    { 
      id: 3, 
      title: 'Рейтинг', 
      value: userData?.rating || '78%', 
      color: '#4A7BC8' 
    }
  ];

  // Если не авторизован - показываем страницу входа
  if (!isLoggedIn) {
    return (
      <LoginPage 
        onLogin={handleLogin} 
        onRegister={handleRegister}
        loading={loading}
        error={error}
      />
    );
  }

  // Интерфейс преподавателя
  if (userRole === 'teacher') {
    return (
      <TeacherAccount 
        userData={userData} 
        onLogout={handleLogout}
        sendMessage={sendMessage}
        connectionStatus={connectionStatus}
      />
    );
  }

  // Студенческий интерфейс
  return (
    <div className="contentContainer">
      {/* Секция профиля */}
      <div className="app-profile-section">
        <ProfileSquare userData={userData} />
        <ProfileDescription userData={userData} />
      </div>
      
      {/* Виджеты */}
      <SlidersUnder />
      <Calendar />
      
      {/* Инфо-карточки */}
      <div className="app-cards-container">
        {cardData.map((card) => (
          <InfoCard
            key={card.id}
            title={card.title}
            value={card.value}
            color={card.color}
          />
        ))}
      </div>
      
      {/* Личный кабинет */}
      <PersonalAccount userData={userData} onLogout={handleLogout} />
      
      {/* Таблицы */}
      <div className="app-tables-container">
        <DataTable
          data={studentsData}
          type="students"
          title="Список студентов"
        />
        
        <DataTable
          data={attendanceData}
          type="attendance"
          title="Посещаемость"
        />
      </div>
      
      {/* Индикатор соединения */}
      <div className={`connection-status ${connectionStatus}`}>
        <div className="status-dot" />
        <span>
          {connectionStatus === 'connected' && 'Подключено'}
          {connectionStatus === 'disconnected' && 'Отключено'}
          {connectionStatus === 'error' && 'Ошибка соединения'}
        </span>
      </div>
    </div>
  );
}

export default App;