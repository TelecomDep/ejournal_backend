import React, { useState } from 'react';
import api from '../services/api';
import './ProfileDescription.css';

const ProfileDescription = ({ userData, token, onUserDataUpdated }) => {
  const displayName = userData?.name || userData?.login || 'Пользователь';
  const [email, setEmail] = useState(userData?.email || '');
  const [editing, setEditing] = useState(false);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  const handleSave = async (e) => {
    e.preventDefault();
    setLoading(true);
    setMessage('');
    setError('');
    try {
      await api.updateEmail(token, email.trim());
      if (onUserDataUpdated) {
        onUserDataUpdated({ email: email.trim() });
      }
      setEditing(false);
      setMessage('Email успешно обновлен');
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось привязать email'));
    } finally {
      setLoading(false);
    }
  };

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

        <div className="profile-email-section" style={{ marginTop: '24px', borderTop: '1px solid rgba(255,255,255,0.1)', paddingTop: '16px' }}>
          <span className="info-label">Привязка почты</span>
          {editing ? (
            <form onSubmit={handleSave} style={{ display: 'flex', gap: '8px', marginTop: '8px', alignItems: 'center' }}>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="example@mail.ru"
                required
                style={{
                  background: '#1a1a1a',
                  border: '1px solid rgba(255,255,255,0.2)',
                  borderRadius: '6px',
                  color: '#fff',
                  padding: '6px 12px',
                  fontSize: '14px',
                  flex: 1
                }}
              />
              <button
                type="submit"
                disabled={loading}
                style={{
                  background: '#2c44b8',
                  color: '#fff',
                  border: 'none',
                  borderRadius: '6px',
                  padding: '6px 12px',
                  cursor: 'pointer',
                  fontSize: '14px'
                }}
              >
                {loading ? 'Сохранение...' : 'Сохранить'}
              </button>
              <button
                type="button"
                onClick={() => { setEditing(false); setEmail(userData?.email || ''); setError(''); setMessage(''); }}
                style={{
                  background: 'transparent',
                  color: 'rgba(255,255,255,0.6)',
                  border: '1px solid rgba(255,255,255,0.2)',
                  borderRadius: '6px',
                  padding: '6px 12px',
                  cursor: 'pointer',
                  fontSize: '14px'
                }}
              >
                Отмена
              </button>
            </form>
          ) : (
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '8px' }}>
              <span className="info-value">{userData?.email || 'Почта не привязана'}</span>
              <button
                type="button"
                onClick={() => setEditing(true)}
                style={{
                  background: 'rgba(44, 68, 184, 0.2)',
                  color: '#2c44b8',
                  border: 'none',
                  borderRadius: '6px',
                  padding: '6px 12px',
                  cursor: 'pointer',
                  fontSize: '13px',
                  fontWeight: 500
                }}
              >
                {userData?.email ? 'Изменить' : 'Привязать'}
              </button>
            </div>
          )}
          {message && <p style={{ color: '#4caf50', fontSize: '12px', marginTop: '8px', marginBottom: 0 }}>{message}</p>}
          {error && <p style={{ color: '#f44336', fontSize: '12px', marginTop: '8px', marginBottom: 0 }}>{error}</p>}
        </div>
      </div>
    </div>
  );
};

export default ProfileDescription;