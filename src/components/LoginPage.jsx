import React, { useState } from 'react';
import './LoginPage.css';

const LoginPage = ({ onLogin, onRegister, loading, error }) => {
  const [isRegister, setIsRegister] = useState(false);
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [roleHash, setRoleHash] = useState('');
  const [inviteCode, setInviteCode] = useState('');

  const handleSubmit = async (event) => {
    event.preventDefault();
    const cleanedRoleHash = roleHash.trim();

    if (isRegister) {
      onRegister(login.trim(), password, cleanedRoleHash, inviteCode.trim());
    } else {
      onLogin(login.trim(), password, cleanedRoleHash);
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

          <label>
            Код доступа
            <input
              type="text"
              value={roleHash}
              onChange={(e) => setRoleHash(e.target.value)}
              placeholder="Введите хэш"
              required
            />
          </label>

          {isRegister && (
            <label>
              Пригласительный код
              <input
                type="text"
                value={inviteCode}
                onChange={(e) => setInviteCode(e.target.value)}
                placeholder="Для регистрации студента по приглашению"
              />
            </label>
          )}

          {error && <div className="login-error">{error}</div>}

          <button type="submit" disabled={loading}>
            {loading ? (isRegister ? 'Регистрация...' : 'Выполняется вход...') : (isRegister ? 'Зарегистрироваться' : 'Войти')}
          </button>
        </form>
      </div>
    </div>
  );
};

export default LoginPage;
