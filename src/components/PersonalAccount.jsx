import React from 'react';
import './PersonalAccount.css';

const PersonalAccount = ({ userData, onLogout }) => {
  const displayName = userData?.name || userData?.login || 'Пользователь';
  const login = userData?.login || 'не указан';
  const role = userData?.role || 'не указана';
  const userId = userData?.user_id || 'не указан';
  const group = userData?.group_name || userData?.group || 'не указана';
  const email = userData?.email || 'не указан';

  return (
    <section className="personal-account">
      <div className="account-header">
        <div className="account-title">
          <h1>Личный кабинет</h1>
          <p>Добро пожаловать, {displayName}</p>
        </div>
        <button className="logout-button" onClick={onLogout}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
            <polyline points="16,17 21,12 16,7" />
            <line x1="21" y1="12" x2="9" y2="12" />
          </svg>
          Выйти
        </button>
      </div>

      <div className="account-details">
        <div className="account-card">
          <div className="card-label">Основная информация</div>
          <h2>Профиль пользователя</h2>
          
          <div className="info-grid">
            <div className="info-item">
              <span className="info-label">Имя</span>
              <span className="info-value name-value">{displayName}</span>
            </div>
            
            <div className="info-item">
              <span className="info-label">Логин</span>
              <span className="info-value">{login}</span>
            </div>
            
            <div className="info-item">
              <span className="info-label">Роль</span>
              <span className="info-value role-badge">{role}</span>
            </div>
            
            <div className="info-item">
              <span className="info-label">ID пользователя</span>
              <span className="info-value id-value">{userId}</span>
            </div>
            
            <div className="info-item">
              <span className="info-label">Группа</span>
              <span className="info-value">{group}</span>
            </div>
            
            <div className="info-item">
              <span className="info-label">Email</span>
              <span className="info-value">{email}</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};

export default PersonalAccount;