import React, { useState } from 'react';

function LoginCard({ onLogin, onRegister, loading, error }) {
  const [isRegister, setIsRegister] = useState(false);
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [roleHash, setRoleHash] = useState('');

  const submit = (event) => {
    event.preventDefault();
    const cleanedLogin = login.trim();
    const cleanedRoleHash = roleHash.trim();

    if (isRegister) {
      onRegister(cleanedLogin, password, cleanedRoleHash);
      return;
    }

    onLogin(cleanedLogin, password, cleanedRoleHash);
  };

  return (
    <div className="auth-wrap">
      <form className="card auth-card" onSubmit={submit}>
        <h1>{isRegister ? 'Регистрация' : 'Вход в систему'}</h1>
        <p className="muted">Роль определяется только по role_hash. Пароль хэшируется SHA-256 в браузере.</p>

        <div className="row gap-sm">
          <button type="button" className={!isRegister ? 'btn btn-primary' : 'btn'} onClick={() => setIsRegister(false)}>
            Вход
          </button>
          <button type="button" className={isRegister ? 'btn btn-primary' : 'btn'} onClick={() => setIsRegister(true)}>
            Регистрация
          </button>
        </div>

        <label className="field">
          <span>Логин</span>
          <input value={login} onChange={(e) => setLogin(e.target.value)} required />
        </label>

        <label className="field">
          <span>Пароль</span>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        </label>

        <label className="field">
          <span>Role hash</span>
          <input
            placeholder="Например: STUDENT-HASH-2026"
            value={roleHash}
            onChange={(e) => setRoleHash(e.target.value)}
            required
          />
        </label>

        {error && <div className="error-box">{error}</div>}

        <button className="btn btn-primary btn-block" disabled={loading}>
          {loading ? 'Подождите...' : isRegister ? 'Зарегистрироваться' : 'Войти'}
        </button>
      </form>
    </div>
  );
}

export default LoginCard;
