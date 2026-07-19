import React, { useState } from 'react';
import api from '../services/api';
import './ProfileDescription.css';

const ProfileDescription = ({ userData, token, onUserDataUpdated }) => {
  const displayName = userData?.name || userData?.login || 'Пользователь';
  // Email state
  const [email, setEmail] = useState(userData?.email || '');
  const [editingEmail, setEditingEmail] = useState(false);
  const [confirmingEmail, setConfirmingEmail] = useState(false);
  const [emailCode, setEmailCode] = useState('');

  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  
  // 2FA state
  const [twoFaEnabled, setTwoFaEnabled] = useState(userData?.is_2fa_enabled || false);
  const [setupTwoFa, setSetupTwoFa] = useState(false);
  const [twoFaCode, setTwoFaCode] = useState('');
  const [qrCodeData, setQrCodeData] = useState('');

  const handleEmailRequest = async (e) => {
    e.preventDefault();
    setLoading(true);
    setMessage('');
    setError('');
    try {
      await api.requestEmailBind(token, email.trim());
      setConfirmingEmail(true);
      setMessage('Код подтверждения отправлен на почту');
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось запросить привязку'));
    } finally {
      setLoading(false);
    }
  };

  const handleEmailConfirm = async (e) => {
    e.preventDefault();
    setLoading(true);
    setMessage('');
    setError('');
    try {
      await api.confirmEmailBind(token, emailCode.trim());
      if (onUserDataUpdated) {
        onUserDataUpdated({ email: email.trim() });
      }
      setEditingEmail(false);
      setConfirmingEmail(false);
      setMessage('Email успешно привязан');
    } catch (err) {
      setError(api.getErrorMessage(err, 'Неверный код подтверждения'));
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
          {editingEmail ? (
            confirmingEmail ? (
              <form onSubmit={handleEmailConfirm} style={{ display: 'flex', gap: '8px', marginTop: '8px', alignItems: 'center' }}>
                <input
                  type="text"
                  value={emailCode}
                  onChange={(e) => setEmailCode(e.target.value)}
                  placeholder="Код из письма"
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
                    background: '#4caf50',
                    color: '#fff',
                    border: 'none',
                    borderRadius: '6px',
                    padding: '6px 12px',
                    cursor: 'pointer',
                    fontSize: '14px'
                  }}
                >
                  {loading ? '...' : 'Подтвердить'}
                </button>
                <button
                  type="button"
                  onClick={() => { setEditingEmail(false); setConfirmingEmail(false); setError(''); setMessage(''); }}
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
              <form onSubmit={handleEmailRequest} style={{ display: 'flex', gap: '8px', marginTop: '8px', alignItems: 'center' }}>
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
                  {loading ? 'Отправка...' : 'Отправить код'}
                </button>
                <button
                  type="button"
                  onClick={() => { setEditingEmail(false); setEmail(userData?.email || ''); setError(''); setMessage(''); }}
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
            )
          ) : (
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '8px' }}>
              <span className="info-value">{userData?.email || 'Почта не привязана'}</span>
              <button
                type="button"
                onClick={() => setEditingEmail(true)}
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

        <div className="profile-2fa-section" style={{ marginTop: '24px', borderTop: '1px solid rgba(255,255,255,0.1)', paddingTop: '16px' }}>
          <span className="info-label">Двухфакторная аутентификация (2FA)</span>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '8px' }}>
            <span className="info-value">{twoFaEnabled ? 'Включена' : 'Отключена'}</span>
            {!twoFaEnabled && !setupTwoFa && (
              <button
                type="button"
                onClick={async () => {
                  try {
                    const result = await api.generate2FA(token);
                    setQrCodeData(result.qr_code);
                    setSetupTwoFa(true);
                  } catch (err) {
                    setError(api.getErrorMessage(err, 'Не удалось начать настройку 2FA'));
                  }
                }}
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
                Включить
              </button>
            )}
            {twoFaEnabled && (
              <button
                type="button"
                onClick={async () => {
                  try {
                    await api.disable2FA(token);
                    setTwoFaEnabled(false);
                    setMessage('2FA успешно отключена');
                  } catch (err) {
                    setError(api.getErrorMessage(err, 'Не удалось отключить 2FA'));
                  }
                }}
                style={{
                  background: 'rgba(244, 67, 54, 0.2)',
                  color: '#f44336',
                  border: 'none',
                  borderRadius: '6px',
                  padding: '6px 12px',
                  cursor: 'pointer',
                  fontSize: '13px',
                  fontWeight: 500
                }}
              >
                Отключить
              </button>
            )}
          </div>
          
          {setupTwoFa && (
            <div style={{ marginTop: '16px', background: 'rgba(0,0,0,0.2)', padding: '12px', borderRadius: '8px' }}>
              <p style={{ fontSize: '13px', color: 'rgba(255,255,255,0.8)', marginBottom: '12px' }}>
                Отсканируйте QR-код в мобильном приложении:
              </p>
              <div style={{ width: '150px', height: '150px', margin: '0 auto 16px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                {qrCodeData ? <img src={qrCodeData} alt="2FA QR Code" style={{width: '100%', height: '100%'}} /> : <span style={{color: '#fff'}}>Загрузка...</span>}
              </div>
              <div style={{ display: 'flex', gap: '8px' }}>
                <input
                  type="text"
                  placeholder="Код подтверждения"
                  value={twoFaCode}
                  onChange={(e) => setTwoFaCode(e.target.value)}
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
                  type="button"
                  onClick={async () => {
                    try {
                      await api.verify2FA(token, twoFaCode.trim());
                      setTwoFaEnabled(true);
                      setSetupTwoFa(false);
                      setTwoFaCode('');
                      setMessage('2FA успешно включена');
                    } catch (err) {
                      setError(api.getErrorMessage(err, 'Неверный код 2FA'));
                    }
                  }}
                  style={{
                    background: '#4caf50',
                    color: '#fff',
                    border: 'none',
                    borderRadius: '6px',
                    padding: '6px 12px',
                    cursor: 'pointer',
                    fontSize: '14px'
                  }}
                >
                  Подтвердить
                </button>
                <button
                  type="button"
                  onClick={() => setSetupTwoFa(false)}
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
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ProfileDescription;