import React from 'react';
import './ProfileSquare.css';

const ProfileSquare = ({ userData }) => {
  return (
    <div className="pfp-square">
      <div className="pfp-block-inner">
        {userData?.avatar ? (
          <img src={userData.avatar} alt="Profile" className="profile-img" />
        ) : (
          <div className="profile-initials">
            {userData?.name?.charAt(0) || 'U'}
          </div>
        )}
      </div>
    </div>
  );
};

export default ProfileSquare;