import React from 'react';
import './PersonalAccount.css';

const PersonalAccount = ({ userData, onLogout }) => {
  const displayName = userData?.name || userData?.login || 'Пользователь';

  return (
    <section className="personal-account">
      <div className="account-header">
        <div className="account-title">
          <h1>Личный кабинет</h1>
          <p>Добро пожаловать, {displayName}</p>
        </div>
        <button className="logout-button" onClick={onLogout}>
          Выйти
        </button>
      </div>

      <div className="account-details">
        <div className="account-card">
          <h2>Информация пользователя</h2>
          <p>
            <strong>Имя:</strong> {displayName}
          </p>
          <p>
            <strong>Логин:</strong> {userData?.login || 'не указан'}
          </p>
          <p>
            <strong>Роль:</strong> {userData?.role || 'не указана'}
          </p>
          <p>
            <strong>ID:</strong> {userData?.user_id || 'не указан'}
          </p>
        </div>
      </div>
    </section>
  );
};

export default PersonalAccount;
