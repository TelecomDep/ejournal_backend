import React from 'react';
import './ProfileDescription.css';

const ProfileDescription = ({ userData }) => {
  const displayName = userData?.name || userData?.login || 'Пользователь';

  return (
    <div className="pfp-description">
      <div className="pfp-block-inner">
        <h2>Личный профиль</h2>
        <p>
          <strong>Имя:</strong> {displayName}
        </p>
        <p>
          <strong>Логин:</strong> {userData?.login || 'не указан'}
        </p>
        <p>
          <strong>Роль:</strong> {userData?.role || 'не указана'}
        </p>
      </div>
    </div>
  );
};

export default ProfileDescription;