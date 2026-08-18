import React, { useEffect, useRef, useState } from 'react';
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

  if (normalized.includes('email_required')) {
    return 'Сначала привяжите email в настройках выше.';
  }

  if (normalized.includes('invalid or expired email confirmation code')) {
    return 'Неверный или истёкший код из письма.';
  }

  if (normalized.includes('invalid totp code') || normalized.includes('invalid 2fa code')) {
    return 'Неверный код. Дождитесь нового кода в приложении и попробуйте ещё раз.';
  }

  if (normalized.includes('setup not initiated')) {
    return 'Сначала подтвердите email для подключения.';
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
  const origW = image.naturalWidth;
  const origH = image.naturalHeight;

  if (origW <= 0 || origH <= 0) {
    throw new Error('Не удалось обработать выбранное изображение');
  }

  const MAX_DIM = 1600;
  let targetW = origW;
  let targetH = origH;

  if (origW > MAX_DIM || origH > MAX_DIM) {
    if (origW >= origH) {
      targetW = MAX_DIM;
      targetH = Math.round((origH * MAX_DIM) / origW);
    } else {
      targetH = MAX_DIM;
      targetW = Math.round((origW * MAX_DIM) / origH);
    }
  }

  const canvas = document.createElement('canvas');
  canvas.width = targetW;
  canvas.height = targetH;

  const context = canvas.getContext('2d');
  if (!context) {
    throw new Error('Не удалось подготовить холст изображения');
  }

  context.imageSmoothingEnabled = true;
  context.imageSmoothingQuality = 'high';
  context.drawImage(image, 0, 0, origW, origH, 0, 0, targetW, targetH);

  const preparedBlob = await canvasToBlob(canvas, 0.88);

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
	const [emailStage, setEmailStage] = useState('idle');
	const [emailBindCode, setEmailBindCode] = useState('');
  const [twoFaEnabled, setTwoFaEnabled] = useState(Boolean(user?.two_fa_enabled));
  const [twoFaStage, setTwoFaStage] = useState('idle'); // 'idle' | 'email_sent' | 'qr_ready'
  const [emailCode, setEmailCode] = useState('');
  const [twoFaSetup, setTwoFaSetup] = useState(null);
  const [twoFaCode, setTwoFaCode] = useState('');
  const [twoFaMessage, setTwoFaMessage] = useState('');
  const [twoFaError, setTwoFaError] = useState('');
  const [twoFaAction, setTwoFaAction] = useState('');
	const [agreement, setAgreement] = useState(null);
	const [agreementAction, setAgreementAction] = useState('');

	useEffect(() => {
		let cancelled = false;
		api.getCurrentAgreement(token)
			.then((status) => { if (!cancelled) setAgreement(status); })
			.catch(() => { if (!cancelled) setAgreement(null); });
		return () => { cancelled = true; };
	}, [token]);

  const handleSubmit = async (event) => {
    event.preventDefault();
    setMessage('');
    setError('');

    try {
      setSaving(true);
      const nextEmail = email.trim();
	  await api.requestEmailBind(token, nextEmail);
	  setEmailStage('confirmation');
	  setEmailBindCode('');
	  setMessage('Код подтверждения отправлен на новый email.');
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось сохранить email'));
    } finally {
      setSaving(false);
    }
  };

	const confirmEmail = async () => {
		setMessage('');
		setError('');
		if (!/^[0-9a-f]{6}$/i.test(emailBindCode.trim())) {
			setError('Введите шестизначный код из письма.');
			return;
		}
		try {
			setSaving(true);
			await api.confirmEmailBind(token, emailBindCode.trim());
			const nextEmail = email.trim();
			setSavedEmail(nextEmail);
			setEmailStage('idle');
			setEmailBindCode('');
			setMessage('Email подтверждён и привязан.');
		} catch (err) {
			setError(api.getErrorMessage(err, 'Не удалось подтвердить email'));
		} finally {
			setSaving(false);
		}
	};

  const startTwoFaSetup = async () => {
    setTwoFaMessage('');
    setTwoFaError('');

    if (!savedEmail) {
      setTwoFaError('Сначала привяжите email в настройках выше.');
      return;
    }

    try {
      setTwoFaAction('request_email');
      await api.request2FAEnable(token);
      setTwoFaStage('email_sent');
      setEmailCode('');
      setTwoFaMessage('Код подтверждения отправлен на вашу почту.');
    } catch (err) {
      setTwoFaError(getTwoFaErrorMessage(err, 'Не удалось отправить код подтверждения на email'));
    } finally {
      setTwoFaAction('');
    }
  };

  const verifyEmailCodeAndGenerate = async (event) => {
    event.preventDefault();
    setTwoFaMessage('');
    setTwoFaError('');

    if (!emailCode.trim()) {
      setTwoFaError('Введите код подтверждения из письма.');
      return;
    }

    try {
      setTwoFaAction('generate');
      const setup = await api.generate2FA(token, emailCode.trim());

      if (!setup?.qr_code || !setup?.secret) {
        throw new Error('Сервер не вернул данные для подключения 2FA');
      }

      setTwoFaSetup(setup);
      setTwoFaStage('qr_ready');
      setTwoFaCode('');
      setTwoFaMessage('Email подтверждён. Отсканируйте QR-код в мобильном приложении.');
    } catch (err) {
      setTwoFaError(getTwoFaErrorMessage(err, 'Не удалось подтвердить код из письма'));
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
      setTwoFaStage('idle');
      setTwoFaCode('');
      setEmailCode('');
	  setTwoFaMessage('Двухфакторная аутентификация включена. Войдите заново.');
	  sessionStorage.removeItem('ejournal_token');
	  window.setTimeout(() => window.location.reload(), 800);
    } catch (err) {
      setTwoFaError(getTwoFaErrorMessage(err, 'Не удалось подтвердить код 2FA'));
    } finally {
      setTwoFaAction('');
    }
  };

  const disableTwoFa = async () => {
	if (!/^\d{6}$/.test(twoFaCode)) {
		setTwoFaError('Введите текущий шестизначный код из приложения-аутентификатора.');
		return;
	}
    if (!window.confirm('Отключить двухфакторную аутентификацию?')) {
      return;
    }

    setTwoFaMessage('');
    setTwoFaError('');

    try {
      setTwoFaAction('disable');
	  await api.disable2FA(token, twoFaCode);
      setTwoFaEnabled(false);
      setTwoFaSetup(null);
      setTwoFaStage('idle');
      setTwoFaCode('');
      setEmailCode('');
	  setTwoFaMessage('Двухфакторная аутентификация отключена. Войдите заново.');
	  sessionStorage.removeItem('ejournal_token');
	  window.setTimeout(() => window.location.reload(), 800);
    } catch (err) {
      setTwoFaError(getTwoFaErrorMessage(err, 'Не удалось отключить 2FA'));
    } finally {
      setTwoFaAction('');
    }
  };

	const updateAgreement = async (decision) => {
		try {
			setAgreementAction(decision);
			const result = await api.recordAgreementDecision(token, decision, agreement?.version || '2026-08-01');
			setAgreement(result);
		} catch (err) {
			setError(api.getErrorMessage(err, 'Не удалось сохранить решение'));
		} finally {
			setAgreementAction('');
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
			{saving ? 'Отправляем...' : 'Подтвердить новый email'}
          </button>
		  {emailStage === 'confirmation' && (
			<div className="security-form">
			  <label>
				Код из письма
				<input value={emailBindCode} onChange={(event) => setEmailBindCode(event.target.value.trim().slice(0, 6))} maxLength={6} />
			  </label>
			  <button type="button" onClick={confirmEmail} disabled={saving}>Подтвердить email</button>
			</div>
		  )}
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
                : 'Для подключения требуется привязанный email и подтверждение по почте.'}
            </p>
          </div>

          <div className="twofa-controls">
            <span className={`twofa-state ${twoFaEnabled ? 'is-enabled' : ''}`}>
              {twoFaEnabled ? 'Включена' : 'Не включена'}
            </span>

            {twoFaEnabled ? (
			  <div>
				<input
				  type="text"
				  inputMode="numeric"
				  value={twoFaCode}
				  onChange={(event) => setTwoFaCode(event.target.value.replace(/\D/g, '').slice(0, 6))}
				  placeholder="Текущий код"
				  maxLength={6}
				/>
				<button type="button" className="twofa-button twofa-button--danger" onClick={disableTwoFa} disabled={Boolean(twoFaAction)}>
				  {twoFaAction === 'disable' ? 'Отключаем...' : 'Отключить'}
				</button>
			  </div>
            ) : (
              <button
                type="button"
                className="twofa-button twofa-button--primary"
                onClick={startTwoFaSetup}
                disabled={Boolean(twoFaAction) || !savedEmail}
                title={!savedEmail ? 'Сначала привяжите email выше' : ''}
              >
                {twoFaAction === 'request_email'
                  ? 'Отправка кода...'
                  : twoFaStage === 'email_sent'
                  ? 'Повторить отправку кода'
                  : twoFaSetup
                  ? 'Запросить новый QR-код'
                  : 'Подключить'}
              </button>
            )}
          </div>
        </div>

        {twoFaMessage && <div className="security-message security-message--success">{twoFaMessage}</div>}
        {twoFaError && <div className="security-message security-message--error">{twoFaError}</div>}

        {!twoFaEnabled && twoFaStage === 'email_sent' && (
          <div className="twofa-setup">
            <div className="twofa-setup-content" style={{ width: '100%' }}>
              <h4>Шаг 1 из 2: Подтверждение по Email</h4>
              <p style={{ color: 'var(--text-secondary, #64748b)', marginBottom: '16px' }}>
                Мы отправили письмо с кодом подтверждения на ваш email <strong>{savedEmail}</strong>.
              </p>

              <form className="twofa-verify-form" onSubmit={verifyEmailCodeAndGenerate}>
                <label htmlFor="email-code">Код из письма</label>
                <div style={{ display: 'flex', gap: '12px' }}>
                  <input
                    id="email-code"
                    type="text"
                    value={emailCode}
                    onChange={(event) => setEmailCode(event.target.value.trim())}
                    placeholder="Введите код из письма"
                    required
                  />
                  <button
                    type="submit"
                    className="twofa-button twofa-button--primary"
                    disabled={!emailCode.trim() || Boolean(twoFaAction)}
                  >
                    {twoFaAction === 'generate' ? 'Проверяем...' : 'Подтвердить и показать QR'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {!twoFaEnabled && twoFaStage === 'qr_ready' && twoFaSetup && (
          <div className="twofa-setup">
            <div className="twofa-qr">
              <img src={twoFaSetup.qr_code} alt="QR-код для подключения двухфакторной аутентификации" />
              <span>QR-код подключения</span>
            </div>

            <div className="twofa-setup-content">
              <h4>Шаг 2 из 2: Сканирование и активация</h4>
              <ol>
                <li>Отсканируйте QR-код в приложении-аутентификаторе (Google Authenticator, Yandex Key и др.).</li>
                <li>Введите текущий шестизначный код из приложения.</li>
              </ol>

              <div className="twofa-secret">
                <span>Ключ для ручной настройки</span>
                <div>
                  <code>{twoFaSetup.secret}</code>
                  <button type="button" onClick={copyTwoFaSecret}>Скопировать</button>
                </div>
              </div>

              <form className="twofa-verify-form" onSubmit={verifyTwoFa}>
                <label htmlFor="twofa-code">Код из приложения 2FA</label>
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
                    {twoFaAction === 'verify' ? 'Проверяем...' : 'Активировать 2FA'}
                  </button>
                </div>
                <small id="twofa-code-hint">Код обновляется в приложении каждые 30 секунд.</small>
              </form>
            </div>
          </div>
        )}
      </section>

      <div className="security-section-divider" />

      <button
        type="button"
        className="consent-revoke-button"
		disabled={Boolean(agreementAction)}
		onClick={() => updateAgreement(agreement?.accepted ? 'declined' : 'accepted')}
      >
		{agreement?.accepted
		  ? 'Отозвать согласие на обработку персональных данных'
		  : 'Дать согласие на обработку персональных данных'}
      </button>
	  <small>При отказе сервис продолжит работать, но ФИО в общем рейтинге будет скрыто.</small>
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
  const [isLightboxOpen, setIsLightboxOpen] = useState(false);
  const [imgError, setImgError] = useState(false);

  useEffect(() => {
    setImgError(false);
  }, [user?.avatar, avatarPreview]);

  useEffect(() => {
    if (!isLightboxOpen) return;
    const handleKeyDown = (event) => {
      if (event.key === 'Escape') {
        setIsLightboxOpen(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isLightboxOpen]);

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

  const hasValidAvatar = Boolean((avatarPreview || user?.avatar) && !imgError);

  return (
    <div className="profile-layout">
      <aside className="profile-photo-card">
        <div
          className={`profile-avatar ${hasValidAvatar ? 'is-clickable' : ''}`}
          aria-label={`Фото профиля: ${getInitials(displayName)}`}
          title={hasValidAvatar ? 'Нажмите для просмотра в полном размере' : ''}
          onClick={() => {
            if (hasValidAvatar) {
              setIsLightboxOpen(true);
            }
          }}
        >
          {hasValidAvatar ? (
            <img
              src={avatarPreview || user.avatar}
              alt={`Аватар пользователя ${displayName}`}
              onError={() => setImgError(true)}
            />
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

      {isLightboxOpen && hasValidAvatar && (
        <div
          className="avatar-lightbox-backdrop"
          onClick={() => setIsLightboxOpen(false)}
          role="dialog"
          aria-modal="true"
        >
          <div className="avatar-lightbox-content" onClick={(event) => event.stopPropagation()}>
            <button
              type="button"
              className="avatar-lightbox-close"
              onClick={() => setIsLightboxOpen(false)}
              aria-label="Закрыть просмотр"
            >
              ✕
            </button>
            <img
              src={avatarPreview || user?.avatar}
              alt={`Аватар пользователя ${displayName}`}
              className="avatar-lightbox-image"
            />
            <span className="avatar-lightbox-caption">{displayName}</span>
          </div>
        </div>
      )}
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
