import React, { useState, useEffect } from 'react';
import SibLogo from './SibLogo';
import LegalFooter from './legal/LegalFooter';
import api from '../services/api';
import './LoginPage.css';

const LoginPage = ({ onLogin, onRegister, loading, error, onOpenLegal }) => {
  const [view, setView] = useState('login'); // 'login' | 'register' | 'forgot' | 'reset'
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [registrationCode, setRegistrationCode] = useState('');
  const [personalDataConsent, setPersonalDataConsent] = useState(false);
  const [emailNotificationsConsent, setEmailNotificationsConsent] = useState(false);
  const [twoFaCode, setTwoFaCode] = useState('');

  // Password reset fields
  const [identity, setIdentity] = useState('');
  const [resetToken, setResetToken] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [registerConfirmPassword, setRegisterConfirmPassword] = useState('');

  // Keyboard state detection (Caps Lock & Russian layout)
  const [capsLockActive, setCapsLockActive] = useState(false);

  const handlePasswordKeyEvent = (event) => {
    if (event.getModifierState) {
      setCapsLockActive(event.getModifierState('CapsLock'));
    }
  };

  const hasRussianInPassword = (
    /[а-яА-ЯёЁ]/.test(password) ||
    (view === 'register' && /[а-яА-ЯёЁ]/.test(registerConfirmPassword)) ||
    (view === 'reset' && (/[а-яА-ЯёЁ]/.test(newPassword) || /[а-яА-ЯёЁ]/.test(confirmPassword)))
  );

  // Internal flow messages & errors
  const [localError, setLocalError] = useState('');
  const [localMessage, setLocalMessage] = useState('');
  const [localLoading, setLocalLoading] = useState(false);

  useEffect(() => {
    const checkHashRoute = () => {
      const hash = window.location.hash;
      const search = window.location.search;

      if (hash.startsWith('#/reset-password')) {
        const match = hash.match(/token=([^&]+)/);
        if (match) {
          setResetToken(match[1]);
          setView('reset');
          setLocalError('');
          setLocalMessage('');
        }
      } else {
        const fullUrl = hash + search;
        const codeMatch = fullUrl.match(/(?:code|invite)=([^&]+)/i);
        if (codeMatch && codeMatch[1]) {
          const inviteCode = decodeURIComponent(codeMatch[1]);
          setRegistrationCode(inviteCode);
          setView('register');
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
      if (!login.trim()) {
        setLocalError('Пожалуйста, введите логин');
        return;
      }
      if (!registrationCode.trim()) {
        setLocalError('Пожалуйста, введите код регистрации (инвайт-код)');
        return;
      }
      if (password.length < 8) {
        setLocalError('Пароль должен содержать минимум 8 символов');
        return;
      }
      if (password !== registerConfirmPassword) {
        setLocalError('Пароли не совпадают');
        return;
      }
      if (!personalDataConsent) {
        setLocalError('Для регистрации необходимо подтвердить согласие на обработку персональных данных');
        return;
      }
      onRegister(login.trim(), password, registrationCode.trim(), personalDataConsent, emailNotificationsConsent);
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
      if (newPassword.length < 8) {
        setLocalError('Пароль должен содержать минимум 8 символов');
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

  const renderKeyboardWarning = () => {
    if (!capsLockActive && !hasRussianInPassword) return null;
    return (
      <div className="keyboard-warning-container" role="alert">
        {capsLockActive && (
          <div className="keyboard-warning-item keyboard-warning--caps">
            <svg className="keyboard-warning-svg" viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
              <path d="M12 4l-6 6h4v8h4v-8h4l-6-6zM5 20h14v2H5v-2z" />
            </svg>
            <span>Внимание: включен <strong>Caps Lock</strong></span>
          </div>
        )}
        {hasRussianInPassword && (
          <div className="keyboard-warning-item keyboard-warning--layout">
            <svg className="keyboard-warning-svg" viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z" />
            </svg>
            <span>Внимание: обнаружена <strong>русская раскладка</strong> (в пароле есть кириллица)</span>
          </div>
        )}
      </div>
    );
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
            {view === 'login' && (
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
                    onKeyDown={handlePasswordKeyEvent}
                    onKeyUp={handlePasswordKeyEvent}
                    placeholder="Введите пароль"
                    required
                  />
                </label>

                {renderKeyboardWarning()}

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
              <>
                <label>
                  Логин
                  <input
                    type="text"
                    value={login}
                    onChange={(e) => setLogin(e.target.value)}
                    placeholder="Придумайте логин"
                    required
                  />
                </label>

                <label>
                  Код регистрации
                  <input
                    type="text"
                    value={registrationCode}
                    onChange={(e) => setRegistrationCode(e.target.value)}
                    placeholder="Введите инвайт-код из деканата или кафедры"
                    required
                  />
                </label>

                <label>
                  Пароль
                  <input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    onKeyDown={handlePasswordKeyEvent}
                    onKeyUp={handlePasswordKeyEvent}
                    placeholder="Придумайте пароль (от 8 символов)"
                    required
                  />
                </label>

                <label>
                  Подтверждение пароля
                  <input
                    type="password"
                    value={registerConfirmPassword}
                    onChange={(e) => setRegisterConfirmPassword(e.target.value)}
                    onKeyDown={handlePasswordKeyEvent}
                    onKeyUp={handlePasswordKeyEvent}
                    placeholder="Повторите пароль"
                    required
                  />
                </label>

                {renderKeyboardWarning()}

                <div className="registration-consents-group">
                  <div className="registration-consent-item">
                    <input
                      id="personal-data-consent"
                      type="checkbox"
                      checked={personalDataConsent}
                      onChange={(event) => {
                        setPersonalDataConsent(event.target.checked);
                        if (event.target.checked) setLocalError('');
                      }}
                      required
                    />
                    <label htmlFor="personal-data-consent">
                      Я даю согласие на{' '}
                      <button
                        type="button"
                        className="legal-inline-link"
                        onClick={() => onOpenLegal && onOpenLegal('privacy')}
                      >
                        обработку персональных данных
                      </button>{' '}
                      (152-ФЗ) и принимаю условия{' '}
                      <button
                        type="button"
                        className="legal-inline-link"
                        onClick={() => onOpenLegal && onOpenLegal('terms')}
                      >
                        Пользовательского соглашения
                      </button>
                      <span className="required-star" title="Обязательно">*</span>
                    </label>
                  </div>

                  <div className="registration-consent-item is-optional">
                    <input
                      id="email-notifications-consent"
                      type="checkbox"
                      checked={emailNotificationsConsent}
                      onChange={(event) => setEmailNotificationsConsent(event.target.checked)}
                    />
                    <label htmlFor="email-notifications-consent">
                      Я согласен(а) получать системные email-уведомления об успеваемости и расписании (опционально)
                    </label>
                  </div>
                </div>
              </>
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
                    onKeyDown={handlePasswordKeyEvent}
                    onKeyUp={handlePasswordKeyEvent}
                    placeholder="Новый пароль (от 8 символов)"
                    required
                  />
                </label>

                <label>
                  Подтверждение пароля
                  <input
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    onKeyDown={handlePasswordKeyEvent}
                    onKeyUp={handlePasswordKeyEvent}
                    placeholder="Повторите новый пароль"
                    required
                  />
                </label>

                {renderKeyboardWarning()}
              </>
            )}
          </div>

          {(error || localError) && error !== 'requires_2fa' && <div className="login-error">{error || localError}</div>}
          {localMessage && <div className="login-success">{localMessage}</div>}

          <button
            type="submit"
            disabled={loading || localLoading}
          >
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

      <LegalFooter onOpenLegal={onOpenLegal} className="login-page-footer" />
    </div>
  );
};

export default LoginPage;
