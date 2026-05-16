import React from 'react';
import './ProfileDescription.css';

const ProfileDescription = ({ userData }) => {
  const displayName = userData?.name || userData?.login || 'Пользователь';

  return (
    <div className="pfp-description">
      <div className="pfp-description-inner">
        <div className="profile-info-grid">
          <div className="info-item">
            <span className="info-label">Имя</span>
            <span className="info-value">{displayName}</span>
          </div>
          <div className="info-item">
            <span className="info-label">Логин</span>
            <span className="info-value">{userData?.login || '—'}</span>
          </div>
          <div className="info-item">
            <span className="info-label">Роль</span>
            <span className="role-badge">{userData?.role || 'студент'}</span>
          </div>
          <div className="info-item">
            <span className="info-label">Группа</span>
            <span className="info-value">{userData?.group_name || userData?.group_id || '—'}</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ProfileDescription;