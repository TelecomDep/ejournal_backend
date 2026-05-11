import React from 'react';
import './ProfileSquare.css';

const ProfileSquare = ({ userData }) => {
  const getInitials = () => {
    if (!userData?.name) return '?';
    const parts = userData.name.split(' ');
    return parts.length > 1 
      ? `${parts[0][0]}${parts[1][0]}`.toUpperCase()
      : userData.name[0].toUpperCase();
  };

  const getFullName = () => {
    return userData?.name || 'Пользователь';
  };

  return (
    <div className="pfp-square">
      <div className="pfp-block-inner">
        {userData?.avatar ? (
          <img 
            src={userData.avatar} 
            alt={getFullName()} 
            className="profile-img" 
          />
        ) : (
          <div className="profile-initials">
            <span className="initials-text">{getInitials()}</span>
            <span className="profile-status">
              {userData?.status || 'В сети'}
            </span>
          </div>
        )}
      </div>
    </div>
  );
};

export default ProfileSquare;