import React, { useRef, useState } from 'react';
import api from '../../services/api';
import { getRoleLabel } from '../navigation';

const pickDisplayName = (user) => (
  user?.student_name || user?.teacher_name || user?.name || user?.login || 'Пользователь'
);

const getInitials = (name = '') =>
  name
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0])
    .join('')
    .toUpperCase() || 'П';

const profileRows = (user) => {
  const rows = [
    { label: 'ID студента', value: user?.student_id, compact: true },
    { label: 'ID преподавателя', value: user?.teacher_id, compact: true },
    { label: 'Email', value: user?.email || 'Не привязан', wide: true },
    { label: 'Должность', value: user?.job_title },
    { label: 'NFC-метка', value: user?.nfc_tag }
  ];

  if (user?.role !== 'teacher') {
    rows.splice(4, 0, { label: 'Кафедра', value: user?.lectern_id ? `ID ${user.lectern_id}` : '' });
  }

  return rows.filter((row) => row.value !== undefined && row.value !== null && row.value !== '');
};

const metricCards = (user) => {
  if (user?.role === 'teacher') {
    return [
      { label: 'Статус', value: getRoleLabel(user?.role) },
      { label: 'Логин', value: user?.login || '—' },
      { label: 'Кафедра', value: user?.lectern_id ? `ID ${user.lectern_id}` : 'Не указана' }
    ];
  }

  return [
    { label: 'Группа', value: user?.group_name || user?.group || '—' },
    { label: 'Статус', value: getRoleLabel(user?.role) },
    { label: 'Логин', value: user?.login || '—' }
  ];
};

const DefaultAvatar = () => (
  <svg className="profile-default-avatar" viewBox="0 0 160 160" role="img" aria-label="Фото профиля">
    <defs>
      <linearGradient id="avatarBlue" x1="24" y1="18" x2="136" y2="148" gradientUnits="userSpaceOnUse">
        <stop stopColor="#3f76ff" />
        <stop offset="1" stopColor="#17359f" />
      </linearGradient>
      <linearGradient id="avatarLight" x1="52" y1="40" x2="116" y2="132" gradientUnits="userSpaceOnUse">
        <stop stopColor="#ffffff" />
        <stop offset="1" stopColor="#dbe6ff" />
      </linearGradient>
    </defs>
    <circle cx="80" cy="80" r="74" fill="url(#avatarBlue)" />
    <circle cx="80" cy="80" r="66" fill="none" stroke="rgba(255,255,255,0.26)" strokeWidth="2" />
    <circle cx="80" cy="63" r="25" fill="url(#avatarLight)" />
    <path
      d="M34 132c7-29 25-45 46-45s39 16 46 45c-12 10-27 16-46 16s-34-6-46-16Z"
      fill="url(#avatarLight)"
    />
  </svg>
);

const TwoFaShieldIcon = () => (
  <svg viewBox="0 0 24 24" aria-hidden="true">
    <path d="M12 3 5.5 5.7v5.1c0 4.4 2.6 8.2 6.5 10.2 3.9-2 6.5-5.8 6.5-10.2V5.7L12 3Z" />
    <rect x="9" y="10.5" width="6" height="5" rx="1.2" />
    <path d="M10.5 10.5V9.2a1.5 1.5 0 0 1 3 0v1.3" />
  </svg>
);

const getTwoFaErrorMessage = (error, fallback) => {
  const message = api.getErrorMessage(error, fallback);
  const normalized = String(message).toLowerCase();

  if (normalized.includes('invalid totp code') || normalized.includes('invalid 2fa code')) {
    return 'Неверный код. Дождитесь нового кода в приложении и попробуйте ещё раз.';
  }

  if (normalized.includes('setup not initiated')) {
    return 'Сначала создайте новый QR-код для подключения.';
  }

  return message;
};

const getAvatarErrorMessage = (error) => {
  const message = api.getErrorMessage(error, 'Не удалось загрузить фото');
  const normalized = String(message).toLowerCase();

  if (normalized.includes('file type is not supported')) {
    return 'Поддерживаются только изображения JPEG, PNG и WebP.';
  }

  if (normalized.includes('between 1 byte and 5 mib')) {
    return 'Размер изображения должен быть не больше 5 MiB.';
  }

  if (normalized.includes('timeout')) {
    return 'Сервер не ответил на загрузку. Попробуйте выбрать изображение ещё раз.';
  }

  return message;
};

const loadAvatarImage = (file) => new Promise((resolve, reject) => {
  const objectUrl = URL.createObjectURL(file);
  const image = new Image();

  image.onload = () => {
    URL.revokeObjectURL(objectUrl);
    resolve(image);
  };
  image.onerror = () => {
    URL.revokeObjectURL(objectUrl);
    reject(new Error('Не удалось прочитать выбранное изображение'));
  };
  image.src = objectUrl;
});

const canvasToBlob = (canvas, quality) => new Promise((resolve, reject) => {
  canvas.toBlob((blob) => {
    if (blob) {
      resolve(blob);
      return;
    }
    reject(new Error('Не удалось подготовить изображение'));
  }, 'image/webp', quality);
});

const prepareAvatarFile = async (file) => {
  const image = await loadAvatarImage(file);
  const sourceSize = Math.min(image.naturalWidth, image.naturalHeight);
  const sourceX = Math.round((image.naturalWidth - sourceSize) / 2);
  const sourceY = Math.round((image.naturalHeight - sourceSize) / 2);
  const canvas = document.createElement('canvas');
  const context = canvas.getContext('2d');
  const targetBytes = 11 * 1024;
  const variants = [
    [192, 0.82],
    [192, 0.66],
    [160, 0.72],
    [160, 0.56],
    [128, 0.64],
    [128, 0.46],
    [96, 0.5]
  ];
  let preparedBlob = null;

  if (!context || sourceSize <= 0) {
    throw new Error('Не удалось обработать выбранное изображение');
  }

  for (const [size, quality] of variants) {
    canvas.width = size;
    canvas.height = size;
    context.clearRect(0, 0, size, size);
    context.drawImage(image, sourceX, sourceY, sourceSize, sourceSize, 0, 0, size, size);
    preparedBlob = await canvasToBlob(canvas, quality);

    if (preparedBlob.size <= targetBytes) {
      break;
    }
  }

  if (!preparedBlob) {
    throw new Error('Не удалось подготовить изображение');
  }

  return new File([preparedBlob], 'avatar.webp', {
    type: 'image/webp',
    lastModified: Date.now()
  });
};

const SecurityPanel = ({ user, token }) => {
  const [savedEmail, setSavedEmail] = useState(user?.email || '');
  const [email, setEmail] = useState(user?.email || '');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [twoFaEnabled, setTwoFaEnabled] = useState(Boolean(user?.two_fa_enabled));
  const [twoFaSetup, setTwoFaSetup] = useState(null);
  const [twoFaCode, setTwoFaCode] = useState('');
  const [twoFaMessage, setTwoFaMessage] = useState('');
  const [twoFaError, setTwoFaError] = useState('');
  const [twoFaAction, setTwoFaAction] = useState('');

  const handleSubmit = async (event) => {
    event.preventDefault();
    setMessage('');
    setError('');

    try {
      setSaving(true);
      const nextEmail = email.trim();
      await api.updateEmail(token, nextEmail);
      setSavedEmail(nextEmail);
      setMessage('Email сохранён. Его можно использовать для восстановления доступа.');
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось сохранить email'));
    } finally {
      setSaving(false);
    }
  };

  const startTwoFaSetup = async () => {
    setTwoFaMessage('');
    setTwoFaError('');

    try {
      setTwoFaAction('generate');
      const setup = await api.generate2FA(token);

      if (!setup?.qr_code || !setup?.secret) {
        throw new Error('Сервер не вернул данные для подключения 2FA');
      }

      setTwoFaSetup(setup);
      setTwoFaCode('');
    } catch (err) {
      setTwoFaError(getTwoFaErrorMessage(err, 'Не удалось создать QR-код для 2FA'));
    } finally {
      setTwoFaAction('');
    }
  };

  const verifyTwoFa = async (event) => {
    event.preventDefault();
    setTwoFaMessage('');
    setTwoFaError('');

    if (!/^\d{6}$/.test(twoFaCode)) {
      setTwoFaError('Введите шестизначный код из приложения-аутентификатора.');
      return;
    }

    try {
      setTwoFaAction('verify');
      await api.verify2FA(token, twoFaCode);
      setTwoFaEnabled(true);
      setTwoFaSetup(null);
      setTwoFaCode('');
      setTwoFaMessage('Двухфакторная аутентификация включена.');
    } catch (err) {
      setTwoFaError(getTwoFaErrorMessage(err, 'Не удалось подтвердить код 2FA'));
    } finally {
      setTwoFaAction('');
    }
  };

  const disableTwoFa = async () => {
    if (!window.confirm('Отключить двухфакторную аутентификацию?')) {
      return;
    }

    setTwoFaMessage('');
    setTwoFaError('');

    try {
      setTwoFaAction('disable');
      await api.disable2FA(token);
      setTwoFaEnabled(false);
      setTwoFaSetup(null);
      setTwoFaCode('');
      setTwoFaMessage('Двухфакторная аутентификация отключена.');
    } catch (err) {
      setTwoFaError(getTwoFaErrorMessage(err, 'Не удалось отключить 2FA'));
    } finally {
      setTwoFaAction('');
    }
  };

  const copyTwoFaSecret = async () => {
    try {
      await navigator.clipboard.writeText(twoFaSetup.secret);
      setTwoFaError('');
      setTwoFaMessage('Ключ настройки скопирован.');
    } catch (err) {
      setTwoFaMessage('');
      setTwoFaError('Не удалось скопировать ключ. Выделите и скопируйте его вручную.');
    }
  };

  return (
    <section className="profile-card profile-security">
      <div className="profile-card-header">
        <div>
          <span>Безопасность</span>
          <h2>Доступ к аккаунту</h2>
          <p>Привяжите email, чтобы восстановление пароля работало без обращения к администратору.</p>
        </div>
      </div>

      <div className="security-grid">
        <div className="security-status-card">
          <span className={`security-status-dot ${savedEmail ? 'is-ok' : ''}`} />
          <div>
            <strong>{savedEmail ? 'Email привязан' : 'Email не привязан'}</strong>
            <p>{savedEmail || 'Укажите почту для восстановления доступа.'}</p>
          </div>
        </div>

        <form className="security-form" onSubmit={handleSubmit}>
          <label>
            Email
            <input
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="name@example.com"
              required
            />
          </label>

          {message && <div className="security-message security-message--success">{message}</div>}
          {error && <div className="security-message security-message--error">{error}</div>}

          <button type="submit" disabled={saving}>
            {saving ? 'Сохраняем...' : 'Сохранить email'}
          </button>
        </form>
      </div>

      <div className="security-section-divider" />

      <section className={`twofa-section ${twoFaEnabled ? 'is-enabled' : ''}`} aria-labelledby="twofa-title">
        <div className="twofa-summary">
          <span className="twofa-icon"><TwoFaShieldIcon /></span>

          <div className="twofa-summary-copy">
            <span>Дополнительная защита</span>
            <h3 id="twofa-title">Двухфакторная аутентификация</h3>
            <p>
              {twoFaEnabled
                ? 'При следующем входе потребуется одноразовый код из приложения-аутентификатора.'
                : 'Защитите аккаунт одноразовым кодом, который меняется каждые 30 секунд.'}
            </p>
          </div>

          <div className="twofa-controls">
            <span className={`twofa-state ${twoFaEnabled ? 'is-enabled' : ''}`}>
              {twoFaEnabled ? 'Включена' : 'Не включена'}
            </span>

            {twoFaEnabled ? (
              <button
                type="button"
                className="twofa-button twofa-button--danger"
                onClick={disableTwoFa}
                disabled={Boolean(twoFaAction)}
              >
                {twoFaAction === 'disable' ? 'Отключаем...' : 'Отключить'}
              </button>
            ) : (
              <button
                type="button"
                className="twofa-button twofa-button--primary"
                onClick={startTwoFaSetup}
                disabled={Boolean(twoFaAction)}
              >
                {twoFaAction === 'generate' ? 'Создаём QR-код...' : twoFaSetup ? 'Обновить QR-код' : 'Подключить'}
              </button>
            )}
          </div>
        </div>

        {twoFaMessage && <div className="security-message security-message--success">{twoFaMessage}</div>}
        {twoFaError && <div className="security-message security-message--error">{twoFaError}</div>}

        {!twoFaEnabled && twoFaSetup && (
          <div className="twofa-setup">
            <div className="twofa-qr">
              <img src={twoFaSetup.qr_code} alt="QR-код для подключения двухфакторной аутентификации" />
              <span>QR-код подключения</span>
            </div>

            <div className="twofa-setup-content">
              <h4>Подтверждение подключения</h4>
              <ol>
                <li>Отсканируйте QR-код в приложении-аутентификаторе.</li>
                <li>Введите текущий шестизначный код.</li>
              </ol>

              <div className="twofa-secret">
                <span>Ключ для ручной настройки</span>
                <div>
                  <code>{twoFaSetup.secret}</code>
                  <button type="button" onClick={copyTwoFaSecret}>Скопировать</button>
                </div>
              </div>

              <form className="twofa-verify-form" onSubmit={verifyTwoFa}>
                <label htmlFor="twofa-code">Код подтверждения</label>
                <div>
                  <input
                    id="twofa-code"
                    type="text"
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    maxLength={6}
                    value={twoFaCode}
                    onChange={(event) => setTwoFaCode(event.target.value.replace(/\D/g, '').slice(0, 6))}
                    placeholder="000000"
                    aria-describedby="twofa-code-hint"
                  />
                  <button
                    type="submit"
                    className="twofa-button twofa-button--primary"
                    disabled={twoFaCode.length !== 6 || Boolean(twoFaAction)}
                  >
                    {twoFaAction === 'verify' ? 'Проверяем...' : 'Подтвердить'}
                  </button>
                </div>
                <small id="twofa-code-hint">Код обновляется в приложении каждые 30 секунд.</small>
              </form>
            </div>
          </div>
        )}
      </section>
    </section>
  );
};

const NotificationToggle = ({ title, description, checked, onChange }) => (
  <label className="notification-toggle">
    <span>
      <strong>{title}</strong>
      <small>{description}</small>
    </span>
    <input
      type="checkbox"
      checked={checked}
      onChange={(event) => onChange(event.target.checked)}
    />
    <span className="notification-switch" aria-hidden="true" />
  </label>
);

const NotificationsPanel = () => {
  const [settings, setSettings] = useState({
    grades: true,
    schedule: true,
    attendance: true,
    system: false
  });

  const updateSetting = (key, value) => {
    setSettings((current) => ({ ...current, [key]: value }));
  };

  return (
    <section className="profile-card profile-notifications">
      <div className="profile-card-header">
        <div>
          <span>Уведомления</span>
          <h2>Настройки оповещений</h2>
          <p>Пока настройки сохраняются только на экране. Позже подключим их к backend.</p>
        </div>
      </div>

      <div className="notification-list">
        <NotificationToggle
          title="Новые оценки"
          description="Показывать уведомления, когда преподаватель выставил или обновил оценку."
          checked={settings.grades}
          onChange={(value) => updateSetting('grades', value)}
        />
        <NotificationToggle
          title="Изменения расписания"
          description="Сообщать о переносах, заменах и новых парах."
          checked={settings.schedule}
          onChange={(value) => updateSetting('schedule', value)}
        />
        <NotificationToggle
          title="Посещаемость"
          description="Напоминать об отметке по QR и показывать результат отметки."
          checked={settings.attendance}
          onChange={(value) => updateSetting('attendance', value)}
        />
        <NotificationToggle
          title="Системные сообщения"
          description="Получать редкие служебные уведомления от системы."
          checked={settings.system}
          onChange={(value) => updateSetting('system', value)}
        />
      </div>

      <div className="notification-history">
        <div className="notification-history-header">
          <h3>Последние уведомления</h3>
          <span>0 новых</span>
        </div>

        <div className="notification-empty">
          <strong>Пока нет уведомлений</strong>
          <p>Здесь появятся новые оценки, изменения расписания и результаты отметки посещаемости.</p>
        </div>
      </div>
    </section>
  );
};

const ProfileInfoPanel = ({ user, token, displayName, rows, metrics, onAvatarChange }) => {
  const fileInputRef = useRef(null);
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const [avatarMessage, setAvatarMessage] = useState('');
  const [avatarError, setAvatarError] = useState('');
  const [avatarPreview, setAvatarPreview] = useState('');

  const handleAvatarSelect = async (event) => {
    const input = event.currentTarget;
    const file = input.files?.[0];

    if (!file) {
      return;
    }

    setAvatarMessage('');
    setAvatarError('');

    const extension = file.name.split('.').pop()?.toLowerCase();
    const supportedExtensions = ['jpg', 'jpeg', 'png', 'webp'];

    if (!extension || !supportedExtensions.includes(extension)) {
      setAvatarError('Поддерживаются только изображения JPEG, PNG и WebP.');
      input.value = '';
      return;
    }

    if (file.size === 0 || file.size > 5 * 1024 * 1024) {
      setAvatarError('Размер изображения должен быть от 1 байта до 5 MiB.');
      input.value = '';
      return;
    }

    let previewUrl = '';

    try {
      setUploadingAvatar(true);
      const preparedFile = await prepareAvatarFile(file);
      previewUrl = URL.createObjectURL(preparedFile);
      setAvatarPreview(previewUrl);

      const avatar = await api.uploadAvatar(token, preparedFile);
      onAvatarChange(avatar);
      setAvatarMessage('Фото профиля обновлено.');
    } catch (error) {
      setAvatarError(getAvatarErrorMessage(error));
    } finally {
      setUploadingAvatar(false);
      setAvatarPreview('');
      if (previewUrl) {
        URL.revokeObjectURL(previewUrl);
      }
      input.value = '';
    }
  };

  return (
    <div className="profile-layout">
      <aside className="profile-photo-card">
        <div className="profile-avatar" aria-label={`Фото профиля: ${getInitials(displayName)}`}>
          {avatarPreview || user?.avatar ? (
            <img src={avatarPreview || user.avatar} alt={`Аватар пользователя ${displayName}`} />
          ) : (
            <DefaultAvatar />
          )}
        </div>

        <div className="profile-photo-actions">
          <input
            ref={fileInputRef}
            className="profile-photo-input"
            type="file"
            accept=".jpg,.jpeg,.png,.webp,image/jpeg,image/png,image/webp"
            onChange={handleAvatarSelect}
          />

          <button
            type="button"
            className="profile-photo-button"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploadingAvatar}
          >
            {uploadingAvatar ? 'Загружаем...' : user?.avatar ? 'Изменить фото' : 'Загрузить фото'}
          </button>

          <small className="profile-photo-note">JPEG, PNG или WebP, до 5 MiB</small>
          {avatarMessage && <span className="profile-photo-message is-success" role="status">{avatarMessage}</span>}
          {avatarError && <span className="profile-photo-message is-error" role="alert">{avatarError}</span>}
        </div>
      </aside>

      <section className="profile-card">
        <div className="profile-card-header">
          <div>
            <span>ФИО</span>
            <h2>{displayName}</h2>
            <p>{user?.group_name || user?.job_title || getRoleLabel(user?.role)}</p>
          </div>
        </div>

        <div className="profile-metrics" aria-label="Краткая информация">
          {metrics.map((metric) => (
            <div className={`profile-metric ${metric.wide ? 'profile-metric--wide' : ''}`} key={metric.label}>
              <span>{metric.label}</span>
              <strong>{metric.value}</strong>
            </div>
          ))}
        </div>

        <div className="profile-details" aria-label="Данные профиля">
          {rows.map((row) => (
            <div
              className={`profile-detail ${row.compact ? 'profile-detail--compact' : ''} ${row.wide ? 'profile-detail--wide' : ''}`}
              key={row.label}
            >
              <span>{row.label}</span>
              <strong>{row.value}</strong>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
};

const ProfileTabPanel = ({ tab, children }) => (
  <div key={tab} className="profile-tab-panel">
    {children}
  </div>
);

const ProfilePage = ({ user, token, onUserUpdate }) => {
  const [activeTab, setActiveTab] = useState('profile');
  const displayName = pickDisplayName(user);
  const rows = profileRows(user);
  const metrics = metricCards(user);

  return (
    <section className="profile-page">
      <header className="profile-heading">
        <h1>Профиль</h1>
      </header>

      <nav className="profile-tabs" aria-label="Разделы профиля">
        <button
          type="button"
          className={activeTab === 'profile' ? 'is-active' : ''}
          onClick={() => setActiveTab('profile')}
        >
          Мой профиль
        </button>
        <button
          type="button"
          className={activeTab === 'security' ? 'is-active' : ''}
          onClick={() => setActiveTab('security')}
        >
          Безопасность
        </button>
        <button
          type="button"
          className={activeTab === 'notifications' ? 'is-active' : ''}
          onClick={() => setActiveTab('notifications')}
        >
          Уведомления
        </button>
      </nav>

      <ProfileTabPanel tab={activeTab}>
        {activeTab === 'profile' && (
          <ProfileInfoPanel
            user={user}
            token={token}
            displayName={displayName}
            rows={rows}
            metrics={metrics}
            onAvatarChange={(avatar) => onUserUpdate({ avatar })}
          />
        )}

        {activeTab === 'security' && <SecurityPanel user={user} token={token} />}

        {activeTab === 'notifications' && <NotificationsPanel />}
      </ProfileTabPanel>
    </section>
  );
};

export default ProfilePage;
