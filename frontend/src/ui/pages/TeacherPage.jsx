import React, { useEffect, useMemo, useRef, useState } from 'react';
import QRCode from '../../components/QRCode';
import api from '../../services/api';
import AttendanceLiveTable from '../components/AttendanceLiveTable';
import TeacherGradebook from '../components/TeacherGradebook';
import { getBrowserLocation } from '../../utils/attendanceFraud';

const formatDateTime = (value) => (value ? new Date(value).toLocaleString('ru-RU') : 'не указан');

const asNumber = (value) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
};

const groupsOf = (subject) => {
  if (!subject) return [];
  if (Array.isArray(subject.groups) && subject.groups.length > 0) {
    return subject.groups
      .map((group) => ({ id: Number(group.id), name: group.name || `Группа ${group.id}` }))
      .filter((group) => Number.isFinite(group.id) && group.id > 0);
  }
  return (Array.isArray(subject.group_ids) ? subject.group_ids : [])
    .map((id) => ({ id: Number(id), name: `Группа ${id}` }))
    .filter((group) => Number.isFinite(group.id) && group.id > 0);
};

const pctTone = (value) => {
  const pct = Number(value) || 0;
  if (pct >= 75) return 'good';
  if (pct >= 50) return 'warn';
  return 'bad';
};

const StatCard = ({ label, value, hint = '' }) => (
  <div className="teacher-stat-card">
    <span>{label}</span>
    <strong>{value}</strong>
    {hint && <small>{hint}</small>}
  </div>
);

const ProgressCell = ({ value }) => {
  const pct = Math.max(0, Math.min(100, Math.round(Number(value) || 0)));
  return (
    <div className="teacher-progress-cell">
      <div className="teacher-progress-track">
        <div className={`teacher-progress-value is-${pctTone(pct)}`} style={{ width: `${pct}%` }} />
      </div>
      <strong>{pct}%</strong>
    </div>
  );
};

/**
 * Safari still exposes prefixed fullscreen members that are absent from
 * TypeScript's standard DOM declarations.
 * @typedef {Document & {
 *   webkitFullscreenElement?: Element | null,
 *   webkitExitFullscreen?: () => Promise<void> | void
 * }} WebkitFullscreenDocument
 * @typedef {HTMLElement & {
 *   webkitRequestFullscreen?: () => Promise<void> | void
 * }} WebkitFullscreenElement
 */

/** @type {WebkitFullscreenDocument} */
const fullscreenDocument = document;

const getFullscreenElement = () => fullscreenDocument.fullscreenElement || fullscreenDocument.webkitFullscreenElement;

const leaveFullscreen = async () => {
  const exitFullscreen = fullscreenDocument.exitFullscreen || fullscreenDocument.webkitExitFullscreen;
  if (!getFullscreenElement() || !exitFullscreen) return;
  try {
    await Promise.resolve(exitFullscreen.call(fullscreenDocument));
  } catch {
    // The CSS fullscreen fallback is closed by local state below.
  }
};

const TeacherPage = ({ token, section = 'attendance' }) => {
  const [subjects, setSubjects] = useState([]);
  const [subjectsLoading, setSubjectsLoading] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  const [sessionForm, setSessionForm] = useState({
    subjectId: 0,
    groupIds: [],
    lessonName: 'Занятие (Практика)',
    lessonType: 'Практика',
    expiresMinutes: 20
  });
  const [sessionLoading, setSessionLoading] = useState(false);
  const saveActiveSession = (session) => {
    setSessionResult(session);
    if (session) {
      try {
        localStorage.setItem('active_teacher_session', JSON.stringify(session));
      } catch {}
    } else {
      try {
        localStorage.removeItem('active_teacher_session');
      } catch {}
    }
  };

  const [sessionResult, setSessionResult] = useState(() => {
    try {
      const saved = localStorage.getItem('active_teacher_session');
      if (saved) {
        const parsed = JSON.parse(saved);
        if (parsed?.expires_at && new Date(parsed.expires_at).getTime() > Date.now()) {
          return parsed;
        }
      }
    } catch {}
    return null;
  });
  const [sessionRecoveryLoading, setSessionRecoveryLoading] = useState(false);
  const [finishingSession, setFinishingSession] = useState(false);

  useEffect(() => {
    let active = true;
    setSessionRecoveryLoading(true);
    api.getTeacherActiveAttendanceSession(token)
      .then((res) => {
        if (!active) return;
        if (res?.active && res?.session) {
          saveActiveSession(res.session);
        } else if (res && !res.active) {
          saveActiveSession(null);
        }
      })
      .catch((err) => {
        console.warn('Could not recover active session:', err);
      })
      .finally(() => {
        if (active) setSessionRecoveryLoading(false);
      });

    return () => {
      active = false;
    };
  }, [token]);

  const handleFinishSession = async () => {
    const sessionId = Number(sessionResult?.session_id || sessionResult?.lesson_id || sessionResult?.id || 0);
    if (!Number.isFinite(sessionId) || sessionId <= 0) {
      setError('Не удалось определить идентификатор активного занятия. Обновите страницу и повторите попытку.');
      return;
    }
    if (!window.confirm('Завершить занятие досрочно? Новые отметки студентов станут недоступны.')) return;
    setFinishingSession(true);
    clearFeedback();
    try {
      await api.finishAttendanceSession(token, sessionId);
      setQrExpanded(false);
      await leaveFullscreen();
      setMessage('Занятие завершено. Новые отметки студентов недоступны.');
      saveActiveSession(null);
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось завершить занятие'));
    } finally {
      setFinishingSession(false);
    }
  };

  const [statsSelection, setStatsSelection] = useState({ subjectId: 0, groupId: 0 });
  const [statsLoading, setStatsLoading] = useState(false);
  const [statsResult, setStatsResult] = useState(null);

  const clearFeedback = () => {
    setError('');
    setMessage('');
  };

  useEffect(() => {
    let active = true;
    setSubjectsLoading(true);
    api.getTeacherSubjects(token)
      .then((response) => {
        if (!active) return;
        const nextSubjects = Array.isArray(response?.subjects) ? response.subjects : [];
        setSubjects(nextSubjects);
        if (nextSubjects.length > 0) {
          const first = nextSubjects[0];
          const subjectId = Number(first.subject_id);
          const groups = groupsOf(first);
          const groupId = groups[0]?.id || 0;
          setSessionForm((current) => ({
            ...current,
            subjectId: current.subjectId || subjectId,
            groupIds: current.groupIds.length ? current.groupIds : groups.map((group) => group.id)
          }));
          setStatsSelection((current) => ({
            subjectId: current.subjectId || subjectId,
            groupId: current.groupId || groupId
          }));
        }
      })
      .catch((err) => {
        if (active) setError(api.getErrorMessage(err, 'Не удалось загрузить предметы преподавателя'));
      })
      .finally(() => {
        if (active) setSubjectsLoading(false);
      });
    return () => {
      active = false;
    };
  }, [token]);

  useEffect(() => {
    if (section !== 'attendance') return undefined;

    let active = true;
    setSessionRecoveryLoading(true);
    api.getTeacherActiveAttendanceSession(token)
      .then((response) => {
        if (!active || !response?.active || !response?.session) return;
        setSessionResult((current) => current || response.session);
      })
      .catch((err) => {
        if (active) setError(api.getErrorMessage(err, 'Не удалось восстановить активное занятие'));
      })
      .finally(() => {
        if (active) setSessionRecoveryLoading(false);
      });

    return () => {
      active = false;
    };
  }, [section, token]);

  const sessionSubject = useMemo(
    () => subjects.find((subject) => Number(subject.subject_id) === Number(sessionForm.subjectId)),
    [subjects, sessionForm.subjectId]
  );
  const sessionGroups = useMemo(() => groupsOf(sessionSubject), [sessionSubject]);
  const statsSubject = useMemo(
    () => subjects.find((subject) => Number(subject.subject_id) === Number(statsSelection.subjectId)),
    [subjects, statsSelection.subjectId]
  );
  const statsGroups = useMemo(() => groupsOf(statsSubject), [statsSubject]);
  const updateSessionSubject = (subjectId) => {
    const selected = subjects.find((subject) => Number(subject.subject_id) === subjectId);
    const groups = groupsOf(selected);
    setSessionForm((current) => {
      const type = current.lessonType || 'Факультатив';
      const subName = selected?.subject_name || 'Занятие';
      return {
        ...current,
        subjectId,
        groupIds: groups.map((group) => group.id),
        lessonName: `${subName} (${type})`
      };
    });
  };

  const toggleSessionGroup = (groupId) => {
    setSessionForm((current) => ({
      ...current,
      groupIds: current.groupIds.includes(groupId)
        ? current.groupIds.filter((id) => id !== groupId)
        : [...current.groupIds, groupId]
    }));
  };

  const createSession = async (event) => {
    event.preventDefault();
    clearFeedback();
    setSessionResult(null);
    if (!sessionForm.subjectId) {
      setError('Выберите предмет.');
      return;
    }
    if (!sessionForm.groupIds.length) {
      setError('Отметьте хотя бы одну группу.');
      return;
    }

    setSessionLoading(true);
    try {
      const location = await getBrowserLocation();
      const response = await api.createAttendanceLink(
        token,
        asNumber(sessionForm.subjectId),
        sessionForm.groupIds,
        sessionForm.lessonName.trim() || 'Занятие',
        asNumber(sessionForm.expiresMinutes),
        sessionForm.lessonType || 'Практика',
        location
      );
      saveActiveSession(response);
      setMessage('QR и ссылка для отметки посещаемости созданы.');
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось создать ссылку посещаемости'));
    } finally {
      setSessionLoading(false);
    }
  };

  const resolveCurrentSiteJoinUrl = (rawJoinUrl) => {
    if (!rawJoinUrl) return "";
    try {
      const currentOrigin = window.location.origin;
      if (rawJoinUrl.includes("#/attendance/join")) {
        const hashPart = rawJoinUrl.substring(rawJoinUrl.indexOf("#/attendance/join"));
        return `${currentOrigin}/${hashPart}`;
      }
      if (rawJoinUrl.includes("token=")) {
        const tokenPart = rawJoinUrl.substring(rawJoinUrl.indexOf("token="));
        return `${currentOrigin}/#/attendance/join?${tokenPart}`;
      }
      return rawJoinUrl;
    } catch {
      return rawJoinUrl;
    }
  };

  const [dynamicNonce, setDynamicNonce] = useState(() => Math.random().toString(36).substring(2, 10));
  const [dynamicTs, setDynamicTs] = useState(() => Math.floor(Date.now() / 1000));
  const [countdown, setCountdown] = useState(4);
  const [qrExpanded, setQrExpanded] = useState(false);
  const qrWorkspaceRef = useRef(null);

  useEffect(() => {
    if (!sessionResult?.join_url) return;

    const interval = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          setDynamicTs(Math.floor(Date.now() / 1000));
          setDynamicNonce(Math.random().toString(36).substring(2, 10));
          return 4;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [sessionResult?.join_url]);

  const activeJoinUrl = resolveCurrentSiteJoinUrl(sessionResult?.join_url);

  const dynamicQrUrl = useMemo(() => {
    if (!activeJoinUrl) return "";
    const separator = activeJoinUrl.includes("?") ? "&" : "?";
    return `${activeJoinUrl}${separator}ts=${dynamicTs}&nonce=${dynamicNonce}`;
  }, [activeJoinUrl, dynamicTs, dynamicNonce]);

  useEffect(() => {
    if (!qrExpanded) return undefined;

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    const closeOnEscape = (event) => {
      if (event.key !== 'Escape') return;
      setQrExpanded(false);
      leaveFullscreen();
    };
    window.addEventListener('keydown', closeOnEscape);

    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener('keydown', closeOnEscape);
    };
  }, [qrExpanded]);

  useEffect(() => {
    const syncFullscreenState = () => {
      if (!getFullscreenElement()) setQrExpanded(false);
    };
    document.addEventListener('fullscreenchange', syncFullscreenState);
    document.addEventListener('webkitfullscreenchange', syncFullscreenState);
    return () => {
      document.removeEventListener('fullscreenchange', syncFullscreenState);
      document.removeEventListener('webkitfullscreenchange', syncFullscreenState);
    };
  }, []);

  useEffect(() => {
    if (!activeJoinUrl) setQrExpanded(false);
  }, [activeJoinUrl]);

  const expandQr = async () => {
    setQrExpanded(true);
    /** @type {WebkitFullscreenElement | null} */
    const qrWorkspace = qrWorkspaceRef.current;
    const requestFullscreen = qrWorkspace?.requestFullscreen || qrWorkspace?.webkitRequestFullscreen;
    if (requestFullscreen) {
      try {
        await Promise.resolve(requestFullscreen.call(qrWorkspace));
      } catch {
        // CSS fallback keeps the QR expanded when native fullscreen is unavailable.
      }
    }
  };

  const closeExpandedQr = async () => {
    setQrExpanded(false);
    await leaveFullscreen();
  };

  const copyAttendanceLink = async () => {
    if (!activeJoinUrl) return;
    try {
      await navigator.clipboard.writeText(activeJoinUrl);
      setMessage("Ссылка скопирована.");
      setError("");
    } catch {
      setError("Не удалось скопировать ссылку автоматически.");
    }
  };

  const loadStats = async (event) => {
    event.preventDefault();
    clearFeedback();
    setStatsResult(null);
    if (!statsSelection.subjectId || !statsSelection.groupId) {
      setError('Выберите предмет и группу.');
      return;
    }

    setStatsLoading(true);
    try {
      const response = await api.getGroupPerformance(
        token,
        asNumber(statsSelection.groupId),
        asNumber(statsSelection.subjectId)
      );
      setStatsResult(response);
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось загрузить сводку по группе'));
    } finally {
      setStatsLoading(false);
    }
  };

  const selectStatsSubject = (subjectId) => {
    const subject = subjects.find((item) => Number(item.subject_id) === subjectId);
    setStatsSelection({ subjectId, groupId: groupsOf(subject)[0]?.id || 0 });
  };

  const noSubjects = !subjectsLoading && subjects.length === 0;

  return (
    <section className="teacher-page">
      {(message || error) && (
        <div className={`teacher-feedback ${error ? 'is-error' : 'is-success'}`}>
          {error || message}
        </div>
      )}

      {section === 'attendance' && (
        <div className="teacher-section-card">
          <header className="teacher-section-head">
            <div>
              <span>Посещаемость</span>
              <h1>QR для отметки посещаемости</h1>
              <p>Выберите предмет и учебные группы на паре. Система создаст QR и ссылку с ограниченным временем действия.</p>
            </div>
          </header>

          <form className="teacher-form-grid" onSubmit={createSession}>
            <label>
              Тип занятия
              <select
                value={sessionForm.lessonType}
                onChange={(event) => {
                  const val = event.target.value;
                  setSessionForm((current) => ({
                    ...current,
                    lessonType: val,
                    lessonName: val === 'Факультатив'
                      ? (sessionSubject?.subject_name ? `${sessionSubject.subject_name} (Факультатив)` : 'Факультатив')
                      : (sessionSubject?.subject_name ? `${sessionSubject.subject_name} (${val})` : `Занятие (${val})`)
                  }));
                }}
              >
                <option value="Практика">Практика</option>
                <option value="Лекция">Лекция</option>
                <option value="Лабораторная работа">Лабораторная работа</option>
                <option value="Факультатив">Факультатив (в любое время)</option>
              </select>
            </label>

            <label>
              Предмет
              <select
                value={sessionForm.subjectId}
                onChange={(event) => updateSessionSubject(asNumber(event.target.value))}
                disabled={subjectsLoading || noSubjects}
                required
              >
                <option value="">{subjectsLoading ? 'Загрузка...' : 'Выберите предмет'}</option>
                {subjects.map((subject) => (
                  <option key={subject.subject_id} value={subject.subject_id}>{subject.subject_name}</option>
                ))}
              </select>
            </label>

            <label>
              Название занятия
              <input
                type="text"
                value={sessionForm.lessonName}
                onChange={(event) => setSessionForm((current) => ({ ...current, lessonName: event.target.value }))}
                required
              />
            </label>

            <label>
              Активна, минут
              <input
                type="number"
                min="1"
                max="180"
                value={sessionForm.expiresMinutes}
                onChange={(event) => setSessionForm((current) => ({ ...current, expiresMinutes: asNumber(event.target.value) }))}
                required
              />
            </label>

            <div className="teacher-field teacher-field-wide">
              <span>Учебные группы на этой паре</span>
              <div className="teacher-chip-list">
                {sessionGroups.map((group) => (
                  <label className={`teacher-chip ${sessionForm.groupIds.includes(group.id) ? 'is-checked' : ''}`} key={group.id}>
                    <input
                      type="checkbox"
                      checked={sessionForm.groupIds.includes(group.id)}
                      onChange={() => toggleSessionGroup(group.id)}
                    />
                    <span className="teacher-chip-check" aria-hidden="true" />
                    <span className="teacher-chip-text">{group.name}</span>
                  </label>
                ))}
                {!sessionGroups.length && <p className="teacher-empty">Для выбранного предмета нет привязанных учебных групп.</p>}
              </div>
            </div>

            <div className="teacher-actions">
              <button type="submit" className="teacher-primary" disabled={sessionLoading || noSubjects}>
                {sessionLoading ? 'Создаём...' : 'Создать QR'}
              </button>
              <button type="button" className="teacher-secondary" onClick={copyAttendanceLink} disabled={!sessionResult?.join_url}>
                Скопировать
              </button>
            </div>
          </form>

          {sessionRecoveryLoading && !sessionResult && (
            <div className="teacher-session-loading">Проверяем активное занятие...</div>
          )}

          {sessionResult && (
            <>
              <div className={`teacher-result-grid ${activeJoinUrl ? 'teacher-result-grid--qr' : 'teacher-result-grid--session'}`}>
                {activeJoinUrl ? (
                  <>
                    <div
                      ref={qrWorkspaceRef}
                      className={`teacher-qr-workspace ${qrExpanded ? 'is-expanded' : ''}`}
                      role={qrExpanded ? 'dialog' : undefined}
                      aria-modal={qrExpanded ? 'true' : undefined}
                      aria-label={qrExpanded ? 'QR-код и ручная отметка посещаемости на весь экран' : undefined}
                    >
                      {qrExpanded && <div className="teacher-qr-editor"><AttendanceLiveTable token={token} session={sessionResult} /></div>}
                      <div className="teacher-qr-card">
                        {qrExpanded && (
                          <button
                            type="button"
                            className="teacher-qr-close-button"
                            onClick={closeExpandedQr}
                            aria-label="Закрыть полноэкранный QR-код"
                          >
                            <span aria-hidden="true">×</span> Закрыть
                          </button>
                        )}
                        <QRCode value={dynamicQrUrl} size={220} />
                        <button
                          type="button"
                          className="teacher-qr-expand-button"
                          onClick={expandQr}
                          aria-label="Развернуть QR-код на весь экран"
                        >
                          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                            <polyline points="15 3 21 3 21 9" />
                            <polyline points="9 21 3 21 3 15" />
                            <line x1="21" y1="3" x2="14" y2="10" />
                            <line x1="3" y1="21" x2="10" y2="14" />
                          </svg>
                          <span>Развернуть</span>
                        </button>
                        <div className="teacher-qr-dynamic-badge">
                          <span className="teacher-qr-pulse" />
                          <span>Динамический QR: ротация через {countdown} сек</span>
                        </div>
                        <strong>QR для проектора / экрана</strong>
                        <p>Код автоматически меняется каждые 4 секунды с защитой от фото и пересылки.</p>
                      </div>
                    </div>
                  </>
                ) : (
                  <div className="teacher-active-session-card">
                    <strong>Активное занятие восстановлено</strong>
                    <span>Таблица продолжает обновляться после перезагрузки страницы.</span>
                  </div>
                )}
                <StatCard label="Дата занятия" value={formatDateTime(sessionResult.created_at || new Date())} />
                <StatCard label="Истекает" value={formatDateTime(sessionResult.expires_at)} />
                <div className="teacher-finish-row">
                  <button
                    type="button"
                    className="teacher-danger"
                    onClick={handleFinishSession}
                    disabled={finishingSession}
                  >
                    {finishingSession ? 'Завершаем...' : 'Завершить занятие'}
                  </button>
                  <span>
                    Досрочное завершение пары сразу блокирует новые отметки студентов.
                  </span>
                </div>
              </div>

            </>
          )}
        </div>
      )}

      {section === 'statistics' && (
        <div className="teacher-section-card">
          <header className="teacher-section-head">
            <div>
              <span>Аналитика</span>
              <h1>Сводка по группе</h1>
              <p>Посещаемость и успеваемость студентов в разрезе предмета.</p>
            </div>
          </header>

          <div className="teacher-explain">
            <strong>Как работает аналитика</strong>
            <p>Вы выбираете предмет и учебную группу. Посещаемость считается как доля отмеченных занятий, успеваемость - как набранные баллы от максимума по работам БРС.</p>
          </div>

          <form className="teacher-form-grid is-compact" onSubmit={loadStats}>
            <label>
              Предмет
              <select
                value={statsSelection.subjectId}
                onChange={(event) => selectStatsSubject(asNumber(event.target.value))}
                disabled={subjectsLoading || noSubjects}
                required
              >
                <option value="">{subjectsLoading ? 'Загрузка...' : 'Выберите предмет'}</option>
                {subjects.map((subject) => (
                  <option key={subject.subject_id} value={subject.subject_id}>{subject.subject_name}</option>
                ))}
              </select>
            </label>
            <label>
              Учебная группа
              <select
                value={statsSelection.groupId}
                onChange={(event) => setStatsSelection((current) => ({ ...current, groupId: asNumber(event.target.value) }))}
                disabled={!statsGroups.length}
                required
              >
                <option value="">{statsGroups.length ? 'Выберите группу' : 'Нет групп'}</option>
                {statsGroups.map((group) => (
                  <option key={group.id} value={group.id}>{group.name}</option>
                ))}
              </select>
            </label>
            <div className="teacher-actions">
              <button type="submit" className="teacher-primary" disabled={statsLoading || !statsSelection.subjectId || !statsSelection.groupId}>
                {statsLoading ? 'Загружаем...' : 'Показать'}
              </button>
            </div>
          </form>

          {statsResult && (
            <>
              <div className="teacher-stat-grid">
                <StatCard label="Студентов" value={statsResult.summary?.students_count ?? 0} />
                <StatCard label="Занятий проведено" value={statsResult.summary?.sessions_count ?? 0} />
                <StatCard label="Средняя посещаемость" value={`${Number(statsResult.summary?.avg_attendance_percent || 0).toFixed(0)}%`} />
                <StatCard label="Средняя успеваемость" value={`${Number(statsResult.summary?.avg_grade_percent || 0).toFixed(0)}%`} />
              </div>

              <div className="teacher-table-wrap">
                <table className="teacher-table">
                  <thead>
                    <tr>
                      <th>ФИО</th>
                      <th>Посещаемость</th>
                      <th>Успеваемость</th>
                      <th>Баллы</th>
                    </tr>
                  </thead>
                  <tbody>
                    {statsResult.students?.length ? (
                      statsResult.students.map((student) => (
                        <tr key={student.student_id}>
                          <td>{student.student_name}</td>
                          <td><ProgressCell value={student.attendance_percent} /></td>
                          <td><ProgressCell value={student.grade_percent} /></td>
                          <td>{student.current_score} / {student.total_max}</td>
                        </tr>
                      ))
                    ) : (
                      <tr><td colSpan={4}>В группе пока нет студентов</td></tr>
                    )}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </div>
      )}

      {section === 'grades' && (
        <TeacherGradebook token={token} subjects={subjects} subjectsLoading={subjectsLoading} />
      )}
    </section>
  );
};

export default TeacherPage;
