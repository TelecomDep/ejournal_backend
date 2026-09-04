import React, { useEffect, useState } from 'react';
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

  useEffect(() => {
    if (!token || !userData || !pendingInvite) {
      return undefined;
    }

    let cancelled = false;
    (async () => {
      const location = await getBrowserLocation();
      if (cancelled) return null;
      if (!location) {
        throw new Error('Для отметки посещения необходимо разрешить доступ к геолокации.');
      }
      return api.confirmAttendance(token, pendingInvite, {
        deviceId: getBrowserDeviceId(),
        ...(location || {})
      });
    })()
      .then((result) => {
        if (!cancelled) {
          if (result?.is_fraud) {
            const reason = result.fraud_reason === 'student is too far from lesson location'
              ? 'Вы находитесь слишком далеко от места занятия.'
              : 'Это устройство уже использовалось для отметки другого студента.';
            setAttendanceNotice({ type: 'error', text: `Отметка отклонена антифродом. ${reason}` });
          } else {
            setAttendanceNotice({ type: 'success', text: 'Посещение успешно отмечено!' });
          }
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setAttendanceNotice({
            type: 'error',
            text: api.getErrorMessage(err, 'Не удалось отметить посещение')
          });
        }
      })
      .finally(() => {
        if (cancelled) {
          return;
        }
        sessionStorage.removeItem('ejournal_pending_invite');
        setPendingInvite('');
        navigate(['teacher', 'head'].includes(userData.role) ? '/teacher/attendance' : '/attendance');
      });

    return () => {
      cancelled = true;
    };
  }, [token, userData, pendingInvite]);

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
      const result = await api.register(login, password, registrationCode);
      sessionStorage.setItem('ejournal_token', result.token);
      setToken(result.token);
      navigate('/dashboard');
      api.recordAgreementDecision(
        result.token,
        personalDataConsent ? 'accepted' : 'declined',
        '2026-09-01'
      ).catch(() => {
        // Consent logging is best-effort and must not block registration.
      });
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
