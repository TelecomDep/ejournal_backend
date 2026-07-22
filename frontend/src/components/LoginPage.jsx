import React, { useState, useEffect } from 'react';
import SibLogo from './SibLogo';
import api from '../services/api';
import './LoginPage.css';

const LoginPage = ({ onLogin, onRegister, loading, error }) => {
  const [view, setView] = useState('login'); // 'login' | 'register' | 'forgot' | 'reset'
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [registrationCode, setRegistrationCode] = useState('');
  const [twoFaCode, setTwoFaCode] = useState('');

  // Password reset fields
  const [identity, setIdentity] = useState('');
  const [resetToken, setResetToken] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');

  // Internal flow messages & errors
  const [localError, setLocalError] = useState('');
  const [localMessage, setLocalMessage] = useState('');
  const [localLoading, setLocalLoading] = useState(false);

  useEffect(() => {
    const checkHashRoute = () => {
      const hash = window.location.hash;
      if (hash.startsWith('#/reset-password')) {
        const match = hash.match(/token=([^&]+)/);
        if (match) {
          setResetToken(match[1]);
          setView('reset');
          setLocalError('');
          setLocalMessage('');
        }
      }
    };

    checkHashRoute();
    window.addEventListener('hashchange', checkHashRoute);
    return () => window.removeEventListener('hashchange', checkHashRoute);
  }, []);

  const handleSubmit = async (event) => {
    event.preventDefault();
    setLocalError('');
    setLocalMessage('');

    if (view === 'register') {
      onRegister(login.trim(), password, registrationCode.trim());
    } else if (view === 'login') {
      onLogin(login.trim(), password, twoFaCode.trim());
    } else if (view === 'forgot') {
      if (!identity.trim()) {
        setLocalError('Пожалуйста, введите логин или email');
        return;
      }
      setLocalLoading(true);
      try {
        await api.forgotPassword(identity.trim());
        setLocalMessage('Ссылка для восстановления отправлена на почту (если аккаунт существует)');
        setIdentity('');
      } catch (err) {
        setLocalError(api.getErrorMessage(err, 'Не удалось отправить запрос'));
      } finally {
        setLocalLoading(false);
      }
    } else if (view === 'reset') {
      if (!resetToken.trim()) {
        setLocalError('Токен сброса пароля обязателен');
        return;
      }
      if (newPassword.length < 4) {
        setLocalError('Пароль должен содержать минимум 4 символа');
        return;
      }
      if (newPassword !== confirmPassword) {
        setLocalError('Пароли не совпадают');
        return;
      }
      setLocalLoading(true);
      try {
        await api.resetPassword(resetToken.trim(), newPassword);
        setLocalMessage('Пароль успешно изменен. Перенаправление на вход...');
        setNewPassword('');
        setConfirmPassword('');
        setResetToken('');
        window.location.hash = '/';
        setTimeout(() => {
          setView('login');
          setLocalMessage('');
        }, 2000);
      } catch (err) {
        setLocalError(api.getErrorMessage(err, 'Не удалось сбросить пароль'));
      } finally {
        setLocalLoading(false);
      }
    }
  };

  return (
    <div className="login-page">
      <div className="login-card">
        <div className="login-brand">
          <SibLogo size={56} withWordmark />
        </div>
        <h1>
          {view === 'register'
            ? 'Регистрация'
            : view === 'forgot'
            ? 'Восстановление доступа'
            : view === 'reset'
            ? 'Сброс пароля'
            : 'Вход в личный кабинет'}
        </h1>

        {(view === 'login' || view === 'register') && (
          <div className={`toggle-buttons ${view === 'register' ? 'is-register' : 'is-login'}`}>
            <span className="toggle-indicator" aria-hidden="true" />
            <button
              type="button"
              className={view === 'login' ? 'active' : ''}
              onClick={() => { setView('login'); setLocalError(''); setLocalMessage(''); }}
            >
              Вход
            </button>
            <button
              type="button"
              className={view === 'register' ? 'active' : ''}
              onClick={() => { setView('register'); setLocalError(''); setLocalMessage(''); }}
            >
              Регистрация
            </button>
          </div>
        )}

        <form className="login-form" onSubmit={handleSubmit}>
          <div key={view} className={`login-form-panel login-form-panel--${view}`}>
            {(view === 'login' || view === 'register') && (
              <>
                <label>
                  Логин
                  <input
                    type="text"
                    value={login}
                    onChange={(e) => setLogin(e.target.value)}
                    placeholder="Введите логин"
                    required
                  />
                </label>

                <label>
                  Пароль
                  <input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Введите пароль"
                    required
                  />
                </label>
              </>
            )}

            {view === 'login' && (
              <>
                {error === 'requires_2fa' && (
                  <label>
                    Код 2FA
                    <input
                      type="text"
                      value={twoFaCode}
                      onChange={(e) => setTwoFaCode(e.target.value)}
                      placeholder="Код из приложения"
                      required
                    />
                  </label>
                )}

                <div className="forgot-password-link">
                  <button
                    type="button"
                    className="forgot-password-button"
                    onClick={() => { setView('forgot'); setLocalError(''); setLocalMessage(''); }}
                  >
                    Забыли пароль?
                  </button>
                </div>
              </>
            )}

            {view === 'register' && (
              <label>
                Код регистрации
                <input
                  type="text"
                  value={registrationCode}
                  onChange={(e) => setRegistrationCode(e.target.value)}
                  placeholder="Введите код из БД"
                  required
                />
              </label>
            )}

            {view === 'forgot' && (
              <label>
                Логин или Email
                <input
                  type="text"
                  value={identity}
                  onChange={(e) => setIdentity(e.target.value)}
                  placeholder="Введите ваш логин или email"
                  required
                />
              </label>
            )}

            {view === 'reset' && (
              <>
                <label>
                  Токен восстановления
                  <input
                    type="text"
                    value={resetToken}
                    onChange={(e) => setResetToken(e.target.value)}
                    placeholder="Введите токен из письма"
                    required
                  />
                </label>

                <label>
                  Новый пароль
                  <input
                    type="password"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    placeholder="Новый пароль"
                    required
                  />
                </label>

                <label>
                  Подтверждение пароля
                  <input
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    placeholder="Повторите новый пароль"
                    required
                  />
                </label>
              </>
            )}
          </div>

          {(error || localError) && error !== 'requires_2fa' && <div className="login-error">{error || localError}</div>}
          {localMessage && <div className="login-success">{localMessage}</div>}

          <button type="submit" disabled={loading || localLoading}>
            {loading || localLoading
              ? (view === 'register' ? 'Регистрация...' : view === 'forgot' ? 'Отправка...' : view === 'reset' ? 'Сброс...' : 'Выполняется вход...')
              : (view === 'register' ? 'Зарегистрироваться' : view === 'forgot' ? 'Восстановить пароль' : view === 'reset' ? 'Сбросить пароль' : 'Войти')}
          </button>

          {(view === 'forgot' || view === 'reset') && (
            <button
              type="button"
              className="back-to-login-btn"
              onClick={() => { setView('login'); setLocalError(''); setLocalMessage(''); }}
            >
              Назад к входу
            </button>
          )}
        </form>
      </div>
    </div>
  );
};

export default LoginPage;
