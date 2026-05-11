import React, { useState } from 'react';
import './LoginPage.css';

const LoginPage = ({ onLogin, onRegister, loading, error }) => {
  const [isRegister, setIsRegister] = useState(false);
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [registrationCode, setRegistrationCode] = useState('');

  const handleSubmit = async (event) => {
    event.preventDefault();

    if (isRegister) {
      onRegister(login.trim(), password, registrationCode.trim());
    } else {
      onLogin(login.trim(), password);
    }
  };

  return (
    <div className="login-page">
      <div className="login-card">
        <h1>{isRegister ? 'Регистрация' : 'Вход в личный кабинет'}</h1>
        
        <div className="toggle-buttons">
          <button
            type="button"
            className={!isRegister ? 'active' : ''}
            onClick={() => setIsRegister(false)}
          >
            Вход
          </button>
          <button
            type="button"
            className={isRegister ? 'active' : ''}
            onClick={() => setIsRegister(true)}
          >
            Регистрация
          </button>
        </div>

        <form className="login-form" onSubmit={handleSubmit}>
          <label className="form-label">
            <span className="label-text">Логин</span>
            <input
              type="text"
              value={login}
              onChange={(e) => setLogin(e.target.value)}
              placeholder="Введите логин"
              required
              className="form-input"
            />
          </label>

          <label className="form-label">
            <span className="label-text">Пароль</span>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Введите пароль"
              required
              className="form-input"
            />
          </label>

          {isRegister && (
            <label className="form-label">
              <span className="label-text">Код регистрации</span>
              <input
                type="text"
                value={registrationCode}
                onChange={(e) => setRegistrationCode(e.target.value)}
                placeholder="Введите код из БД"
                required
                className="form-input"
              />
            </label>
          )}

          {error && <div className="login-error">{error}</div>}

          <button type="submit" disabled={loading} className="submit-btn">
            {loading ? (
              <span className="loading-spinner" />
            ) : null}
            <span>
              {loading 
                ? (isRegister ? 'Регистрация...' : 'Выполняется вход...') 
                : (isRegister ? 'Зарегистрироваться' : 'Войти')
              }
            </span>
          </button>
        </form>
      </div>
    </div>
  );
};

export default LoginPage;