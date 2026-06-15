import React, { useRef, useState } from 'react';
import api from '../services/api';
import ProfileSquare from '../components/ProfileSquare';
import ProfileDescription from '../components/ProfileDescription';
import './ProfilePage.css';

const ProfilePage = ({ token, userData, onAvatarUpdated }) => {
  const inputRef = useRef(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');

  const handleFile = async (event) => {
    const file = event.target.files && event.target.files[0];
    if (!file) {
      return;
    }
    setError('');
    setUploading(true);
    try {
      const url = await api.uploadAvatar(token, file);
      if (onAvatarUpdated) {
        onAvatarUpdated(url);
      }
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось загрузить фото'));
    } finally {
      setUploading(false);
      if (inputRef.current) {
        inputRef.current.value = '';
      }
    }
  };

  return (
    <section className="profile-page">
      <div className="profile-photo-card">
        <ProfileSquare userData={userData} />
        <input ref={inputRef} type="file" accept="image/*" hidden onChange={handleFile} />
        <button
          type="button"
          className="profile-upload-btn"
          onClick={() => inputRef.current && inputRef.current.click()}
          disabled={uploading}
        >
          {uploading ? 'Загрузка…' : 'Изменить фото'}
        </button>
        {error ? <p className="profile-upload-error">{error}</p> : null}
      </div>
      <div className="profile-details">
        <ProfileDescription userData={userData} />
      </div>
    </section>
  );
};

export default ProfilePage;
