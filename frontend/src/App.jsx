import React, { useEffect, useRef, useState } from 'react';
import api from './services/api';
import LoginPage from './components/LoginPage';
import Workspace from './ui/Workspace';
import LegalModal from './components/legal/LegalModal';
import CookieBanner from './components/legal/CookieBanner';
import useHashRoute from './hooks/useHashRoute';
import { getJoinInviteToken } from './utils/attendanceJoin';
import { getBrowserDeviceId, getBrowserLocation } from './utils/attendanceFraud';

function App() {
  const [token, setToken] = useState(() => sessionStorage.getItem('ejournal_token') || '');
  const [userData, setUserData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [pendingInvite, setPendingInvite] = useState(
    () => getJoinInviteToken() || sessionStorage.getItem('ejournal_pending_invite') || ''
  );
  const [attendanceNotice, setAttendanceNotice] = useState(null);
  const [attendanceSubmitting, setAttendanceSubmitting] = useState(false);
  const attendanceRequestRef = useRef(false);
  const attendanceContextRef = useRef({ token, pendingInvite });
  attendanceContextRef.current = { token, pendingInvite };
  const [legalModalOpen, setLegalModalOpen] = useState(false);
  const [legalInitialDoc, setLegalInitialDoc] = useState('privacy');
  const { route, navigate } = useHashRoute();

  const handleOpenLegal = (docId = 'privacy') => {
    setLegalInitialDoc(docId);
    setLegalModalOpen(true);
  };

  const handleLogout = () => {
    sessionStorage.removeItem('ejournal_token');
    setToken('');
    setUserData(null);
    setError('');
    navigate('/');
  };

  const handleUserUpdate = (updates) => {
    setUserData((current) => (current ? { ...current, ...updates } : current));
  };

  const handleRoleSwitch = async (role) => {
    if (!token) {
      throw new Error('Сессия истекла. Войдите снова.');
    }

    const result = await api.switchRole(token, role);
    const nextToken = result?.token || result?.access_token;
    if (!nextToken) {
      throw new Error('Сервер не вернул новую сессию. Попробуйте ещё раз.');
    }

    sessionStorage.setItem('ejournal_token', nextToken);
    // Clearing the old profile prevents a stale menu from being shown while
    // the profile for the newly signed active role is loaded by the effect.
    setUserData(null);
    setError('');
    setToken(nextToken);
    return result;
  };

  useEffect(() => {
    if (!token) {
      return undefined;
    }

    let cancelled = false;
    setLoading(true);
    api.getProfile(token)
      .then((profile) => {
        if (!cancelled) {
          setUserData(profile);
          setError('');
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(api.getErrorMessage(err, 'Не удалось загрузить профиль'));
          handleLogout();
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [token]);

  useEffect(() => {
    const invite = getJoinInviteToken();
    if (invite) {
      sessionStorage.setItem('ejournal_pending_invite', invite);
      setPendingInvite(invite);
    }
  }, [route]);

  const closePendingAttendance = () => {
    sessionStorage.removeItem('ejournal_pending_invite');
    setPendingInvite('');
    navigate(['teacher', 'head'].includes(userData?.role) ? '/teacher/attendance' : '/attendance');
  };

  const handleConfirmPendingAttendance = async () => {
    if (!token || !userData || !pendingInvite || attendanceRequestRef.current) return;

    attendanceRequestRef.current = true;
    setAttendanceSubmitting(true);
    setAttendanceNotice(null);
    try {
      // Keep the permission request directly attached to the user's click.
      const location = await getBrowserLocation();
      if (attendanceContextRef.current.token !== token || attendanceContextRef.current.pendingInvite !== pendingInvite) return;
      const result = await api.confirmAttendance(token, pendingInvite, {
        deviceId: getBrowserDeviceId(),
        ...location
      });
      if (attendanceContextRef.current.token !== token || attendanceContextRef.current.pendingInvite !== pendingInvite) return;

      if (result?.is_fraud) {
        const reason = result.fraud_reason === 'student is too far from lesson location'
          ? 'Вы находитесь слишком далеко от места занятия.'
          : 'Это устройство уже использовалось для отметки другого студента.';
        setAttendanceNotice({ type: 'error', text: `Отметка отклонена антифродом. ${reason}` });
      } else {
        setAttendanceNotice({ type: 'success', text: 'Посещение успешно отмечено!' });
      }
      closePendingAttendance();
    } catch (err) {
      if (attendanceContextRef.current.token !== token || attendanceContextRef.current.pendingInvite !== pendingInvite) return;
      if (api.getErrorMessage(err, '') === 'attendance already confirmed') {
        setAttendanceNotice({ type: 'success', text: 'Посещение уже отмечено.' });
        closePendingAttendance();
        return;
      }
      // Keep the invite so the student can change Safari settings and retry.
      setAttendanceNotice({
        type: 'error',
        text: api.getErrorMessage(err, 'Не удалось отметить посещение')
      });
    } finally {
      attendanceRequestRef.current = false;
      setAttendanceSubmitting(false);
    }
  };

  const handleLogin = async (login, password, twoFaCode = '') => {
    setLoading(true);
    setError('');
    try {
      const result = await api.login(login, password, twoFaCode);
      sessionStorage.setItem('ejournal_token', result.token);
      setToken(result.token);
      if (!pendingInvite) {
        navigate('/dashboard');
      }
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось войти'));
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (login, password, registrationCode, personalDataConsent, emailConsent = false) => {
    setLoading(true);
    setError('');
    try {
      if (!personalDataConsent) {
        throw new Error('Для регистрации необходимо подтвердить согласие на обработку персональных данных.');
      }
      const result = await api.register(login, password, registrationCode, {
        version: '2026-09-01',
        decision: 'accepted'
      });
      if (!result?.token) {
        throw new Error('Сервер не вернул сессию. Попробуйте войти с указанным логином и паролем.');
      }
      sessionStorage.setItem('ejournal_token', result.token);
      setToken(result.token);
      if (!pendingInvite) navigate('/dashboard');
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось зарегистрироваться'));
    } finally {
      setLoading(false);
    }
  };

  let body;

  if (!token) {
    body = (
      <LoginPage
        onLogin={handleLogin}
        onRegister={handleRegister}
        loading={loading}
        error={error}
        onOpenLegal={handleOpenLegal}
      />
    );
  } else if (!userData) {
    body = <div className="app-loading">Загрузка…</div>;
  } else {
    body = (
      <Workspace
        token={token}
        user={userData}
        route={route}
        navigate={navigate}
        onLogout={handleLogout}
        onUserUpdate={handleUserUpdate}
        onRoleSwitch={handleRoleSwitch}
        onOpenLegal={handleOpenLegal}
      />
    );
  }

  return (
    <>
      {attendanceNotice && (
        <div
          className={`attendance-notice attendance-notice--${attendanceNotice.type}`}
          role="status"
          onClick={() => setAttendanceNotice(null)}
        >
          <span>{attendanceNotice.text}</span>
          <button type="button" aria-label="Закрыть" onClick={() => setAttendanceNotice(null)}>×</button>
        </div>
      )}
      {token && userData && pendingInvite && (
        <div className="attendance-location-backdrop" role="presentation">
          <section
            className="attendance-location-dialog"
            role="dialog"
            aria-modal="true"
            aria-labelledby="attendance-location-title"
          >
            <h2 id="attendance-location-title">Подтверждение посещения</h2>
            <p>
              Для отметки нужна ваша геолокация. Нажмите кнопку ниже и разрешите браузеру
              использовать местоположение, когда появится системный запрос.
            </p>
            <p className="attendance-location-hint">
              Если доступ уже был запрещён: Настройки iPhone → Конфиденциальность и безопасность →
              Службы геолокации → Safari.
            </p>
            <div className="attendance-location-actions">
              <button type="button" className="attendance-location-cancel" onClick={closePendingAttendance} disabled={attendanceSubmitting}>
                Отмена
              </button>
              <button type="button" className="attendance-location-confirm" onClick={handleConfirmPendingAttendance} disabled={attendanceSubmitting}>
                {attendanceSubmitting ? 'Определяем местоположение…' : 'Разрешить и отметиться'}
              </button>
            </div>
          </section>
        </div>
      )}
      {body}
      <CookieBanner onOpenLegal={handleOpenLegal} />
      <LegalModal
        isOpen={legalModalOpen}
        onClose={() => setLegalModalOpen(false)}
        initialDoc={legalInitialDoc}
      />
    </>
  );
}

export default App;
