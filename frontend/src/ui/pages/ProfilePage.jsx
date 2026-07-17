import React, { useState } from 'react';
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

const SecurityPanel = ({ user, token }) => {
  const [savedEmail, setSavedEmail] = useState(user?.email || '');
  const [email, setEmail] = useState(user?.email || '');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

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

const ProfileInfoPanel = ({ user, displayName, rows, metrics }) => (
  <div className="profile-layout">
    <aside className="profile-photo-card">
      <div className="profile-avatar" aria-label={`Фото профиля: ${getInitials(displayName)}`}>
        {user?.avatar ? (
          <img src={user.avatar} alt="" />
        ) : (
          <DefaultAvatar />
        )}
      </div>

      <button type="button" className="profile-photo-button">Изменить фото</button>
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

const ProfileTabPanel = ({ tab, children }) => (
  <div key={tab} className="profile-tab-panel">
    {children}
  </div>
);

const ProfilePage = ({ user, token }) => {
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
          <ProfileInfoPanel user={user} displayName={displayName} rows={rows} metrics={metrics} />
        )}

        {activeTab === 'security' && <SecurityPanel user={user} token={token} />}

        {activeTab === 'notifications' && <NotificationsPanel />}
      </ProfileTabPanel>
    </section>
  );
};

export default ProfilePage;
