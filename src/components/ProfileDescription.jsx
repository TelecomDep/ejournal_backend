import React from 'react';
import './ProfileDescription.css';

const ProfileDescription = ({ userData }) => {
  const displayName = userData?.name || 'Пользователь';
  const login = userData?.login || 'Не указан';
  const role = userData?.role || 'Не указана';
  const group = userData?.group_name || userData?.group || 'Не указана';
  const email = userData?.email || 'Не указан';

  return (
    <div className="pfp-description">
      <div className="pfp-block-inner">
        <h2 className="profile-name">{displayName}</h2>
        
        <div className="profile-info-grid">
          <div className="info-item">
            <span className="info-label">Логин</span>
            <span className="info-value">{login}</span>
          </div>
          
          <div className="info-item">
            <span className="info-label">Роль</span>
            <span className="info-value role-badge">{role}</span>
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
  );
};

export default ProfileDescription;