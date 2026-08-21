import React, { useEffect, useMemo, useState } from 'react';
import QRCode from '../../components/QRCode';
import api from '../../services/api';

const gradeTypes = [
  { value: 'current', label: 'Текущая работа' },
  { value: 'laboratory', label: 'Лабораторная' },
  { value: 'test', label: 'Тест' },
  { value: 'exam', label: 'Экзамен' },
  { value: 'project', label: 'Проект' }
];

const formatDateTime = (value) => (value ? new Date(value).toLocaleString('ru-RU') : 'не указан');

const formatDateForAPI = (value) => (value ? new Date(`${value}T23:59:00`).toISOString() : undefined);

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

const SubjectBars = ({ items }) => {
  if (!items?.length) {
    return <p className="teacher-empty">Нет данных для диаграммы.</p>;
  }
  return (
    <div className="teacher-bars">
      {items.map((item) => {
        const pct = Math.max(0, Math.min(100, Math.round(Number(item.percent || item.score || 0))));
        return (
          <div className="teacher-bar-row" key={item.subject_id || item.subject_name}>
            <span title={item.subject_name}>{item.subject_name || `Предмет ${item.subject_id}`}</span>
            <div className="teacher-bar-track">
              <div className={`teacher-bar-fill is-${pctTone(pct)}`} style={{ width: `${pct}%` }} />
            </div>
            <strong>{pct}%</strong>
          </div>
        );
      })}
    </div>
  );
};

const TeacherSteps = ({ items }) => (
  <div className="teacher-steps">
    {items.map((item, index) => (
      <div className="teacher-step" key={item.title}>
        <span>{index + 1}</span>
        <strong>{item.title}</strong>
        <p>{item.text}</p>
      </div>
    ))}
  </div>
);

const TeacherPage = ({ token, section = 'attendance' }) => {
  const [subjects, setSubjects] = useState([]);
  const [subjectsLoading, setSubjectsLoading] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  const [sessionForm, setSessionForm] = useState({
    subjectId: 0,
    groupIds: [],
    lessonName: 'Занятие',
    expiresMinutes: 20
  });
  const [sessionLoading, setSessionLoading] = useState(false);
  const [sessionResult, setSessionResult] = useState(null);

  const [statsSelection, setStatsSelection] = useState({ subjectId: 0, groupId: 0 });
  const [statsLoading, setStatsLoading] = useState(false);
  const [statsResult, setStatsResult] = useState(null);

  const [gradeSelection, setGradeSelection] = useState({ subjectId: 0, groupId: 0, studentId: 0 });
  const [roster, setRoster] = useState([]);
  const [rosterLoading, setRosterLoading] = useState(false);
  const [gradeItemsResult, setGradeItemsResult] = useState(null);
  const [gradesLoading, setGradesLoading] = useState(false);
  const [studentSheet, setStudentSheet] = useState(null);
  const [studentRadar, setStudentRadar] = useState([]);
  const [radarLoading, setRadarLoading] = useState(false);
  const [itemForm, setItemForm] = useState({
    title: 'Лабораторная работа 1',
    maxScore: 10,
    itemType: 'laboratory',
    deadline: ''
  });
  const [gradeForm, setGradeForm] = useState({
    itemId: '',
    score: 0,
    comment: '',
    sessionDate: ''
  });

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
          setGradeSelection((current) => ({
            subjectId: current.subjectId || subjectId,
            groupId: current.groupId || groupId,
            studentId: current.studentId || 0
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
  const gradeSubject = useMemo(
    () => subjects.find((subject) => Number(subject.subject_id) === Number(gradeSelection.subjectId)),
    [subjects, gradeSelection.subjectId]
  );
  const gradeGroups = useMemo(() => groupsOf(gradeSubject), [gradeSubject]);
  const gradeItems = gradeItemsResult?.items || [];
  const gradeItemsTotal = useMemo(
    () => gradeItems.reduce((sum, item) => sum + Number(item.max_score || 0), 0),
    [gradeItems]
  );
  const selectedStudent = useMemo(
    () => roster.find((student) => Number(student.student_id) === Number(gradeSelection.studentId)),
    [roster, gradeSelection.studentId]
  );

  useEffect(() => {
    if (!gradeSelection.subjectId || !gradeSelection.groupId) {
      setRoster([]);
      return undefined;
    }

    let active = true;
    setRosterLoading(true);
    api.getGroupStats(token, gradeSelection.groupId, gradeSelection.subjectId)
      .then((response) => {
        if (!active) return;
        const students = Array.isArray(response?.students) ? response.students : [];
        setRoster(students);
        setGradeSelection((current) => {
          const stillValid = students.some((student) => Number(student.student_id) === Number(current.studentId));
          return {
            ...current,
            studentId: stillValid ? current.studentId : (students[0]?.student_id || 0)
          };
        });
      })
      .catch(() => {
        if (active) setRoster([]);
      })
      .finally(() => {
        if (active) setRosterLoading(false);
      });

    return () => {
      active = false;
    };
  }, [token, gradeSelection.subjectId, gradeSelection.groupId]);

  const loadGradeItems = async (subjectId = gradeSelection.subjectId, silent = false) => {
    if (!silent) clearFeedback();
    if (!subjectId) return;
    setGradesLoading(true);
    try {
      const response = await api.getTeacherGradeItems(token, asNumber(subjectId));
      setGradeItemsResult(response);
      setGradeForm((current) => {
        const items = response?.items || [];
        const stillValid = items.some((item) => String(item.item_id) === String(current.itemId));
        return stillValid ? current : { ...current, itemId: items[0] ? String(items[0].item_id) : '' };
      });
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось загрузить контрольные точки'));
    } finally {
      setGradesLoading(false);
    }
  };

  useEffect(() => {
    if (gradeSelection.subjectId) {
      loadGradeItems(gradeSelection.subjectId, true);
    }
  }, [gradeSelection.subjectId]);

  useEffect(() => {
    const studentId = asNumber(gradeSelection.studentId);
    if (!token || studentId <= 0) {
      setStudentRadar([]);
      return undefined;
    }

    let active = true;
    setRadarLoading(true);
    api.getTeacherStudentPerformanceRadar(token, studentId)
      .then((response) => {
        if (active) setStudentRadar(Array.isArray(response?.subjects) ? response.subjects : []);
      })
      .catch(() => {
        if (active) setStudentRadar([]);
      })
      .finally(() => {
        if (active) setRadarLoading(false);
      });
    return () => {
      active = false;
    };
  }, [token, gradeSelection.studentId]);

  const updateSessionSubject = (subjectId) => {
    const selected = subjects.find((subject) => Number(subject.subject_id) === subjectId);
    const groups = groupsOf(selected);
    setSessionForm((current) => ({
      ...current,
      subjectId,
      groupIds: groups.map((group) => group.id)
    }));
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
      const response = await api.createAttendanceLink(
        token,
        asNumber(sessionForm.subjectId),
        sessionForm.groupIds,
        sessionForm.lessonName.trim() || 'Занятие',
        asNumber(sessionForm.expiresMinutes)
      );
      setSessionResult(response);
      setMessage('QR и ссылка для отметки посещаемости созданы.');
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось создать ссылку посещаемости'));
    } finally {
      setSessionLoading(false);
    }
  };

  const copyAttendanceLink = async () => {
    if (!sessionResult?.join_url) return;
    try {
      await navigator.clipboard.writeText(sessionResult.join_url);
      setMessage('Ссылка скопирована.');
      setError('');
    } catch {
      setError('Не удалось скопировать ссылку автоматически.');
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

  const createGradeItem = async (event) => {
    event.preventDefault();
    clearFeedback();
    const title = itemForm.title.trim();
    if (!gradeSelection.subjectId || !title) {
      setError('Выберите предмет и введите название работы.');
      return;
    }

    setGradesLoading(true);
    try {
      await api.createGradeItem(token, {
        subject_id: asNumber(gradeSelection.subjectId),
        title,
        max_score: asNumber(itemForm.maxScore),
        item_type: itemForm.itemType,
        deadline: formatDateForAPI(itemForm.deadline)
      });
      setMessage(`Контрольная точка «${title}» добавлена.`);
      await loadGradeItems(gradeSelection.subjectId, true);
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось добавить контрольную точку'));
    } finally {
      setGradesLoading(false);
    }
  };

  const saveGrade = async (event) => {
    event.preventDefault();
    clearFeedback();
    if (!gradeSelection.studentId || !gradeForm.itemId) {
      setError('Выберите студента и контрольную точку.');
      return;
    }

    setGradesLoading(true);
    try {
      const payload = {
        student_id: asNumber(gradeSelection.studentId),
        item_id: asNumber(gradeForm.itemId),
        score: asNumber(gradeForm.score)
      };
      if (gradeForm.comment.trim()) payload.comment = gradeForm.comment.trim();
      await api.saveStudentGrade(token, payload);
      setMessage('Оценка сохранена.');
      if (studentSheet && Number(studentSheet.student_id) === payload.student_id) {
        await loadStudentSheet(null, true);
      }
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось сохранить оценку'));
    } finally {
      setGradesLoading(false);
    }
  };

  const loadStudentSheet = async (event, silent = false) => {
    if (event) event.preventDefault();
    if (!silent) clearFeedback();
    if (!gradeSelection.studentId || !gradeSelection.subjectId) {
      setError('Выберите предмет, группу и студента.');
      return;
    }

    setGradesLoading(true);
    try {
      const response = await api.getTeacherStudentGrades(
        token,
        asNumber(gradeSelection.studentId),
        asNumber(gradeSelection.subjectId)
      );
      setStudentSheet(response);
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось загрузить ведомость студента'));
    } finally {
      setGradesLoading(false);
    }
  };

  const selectStatsSubject = (subjectId) => {
    const subject = subjects.find((item) => Number(item.subject_id) === subjectId);
    setStatsSelection({ subjectId, groupId: groupsOf(subject)[0]?.id || 0 });
  };

  const selectGradeSubject = (subjectId) => {
    const subject = subjects.find((item) => Number(item.subject_id) === subjectId);
    setGradeSelection({ subjectId, groupId: groupsOf(subject)[0]?.id || 0, studentId: 0 });
    setStudentSheet(null);
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

          {sessionResult && (
            <div className="teacher-result-grid teacher-result-grid--qr">
              <div className="teacher-qr-card">
                <QRCode value={sessionResult.join_url} size={220} />
                <strong>QR для студентов</strong>
                <p>Покажите этот код на экране. Студент сканирует его телефоном и отмечается на паре.</p>
              </div>
              <div className="teacher-link-card">
                <span>Ссылка для студентов</span>
                <strong>{sessionResult.join_url || 'не получена'}</strong>
              </div>
              <StatCard label="Дата занятия" value={formatDateTime(sessionResult.created_at || new Date())} />
              <StatCard label="Истекает" value={formatDateTime(sessionResult.expires_at)} />
            </div>
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
        <div className="teacher-section-card">
          <header className="teacher-section-head">
            <div>
              <span>Оценки</span>
              <h1>БРС: работы и баллы</h1>
              <p>Сначала создайте работу с максимальным баллом, затем выберите студента и поставьте ему результат.</p>
            </div>
          </header>

          <TeacherSteps
            items={[
              { title: 'Выберите контекст', text: 'Предмет, учебную группу и студента, которому нужно поставить баллы.' },
              { title: 'Создайте работу', text: 'Лабораторная, тест, проект или другая контрольная точка с максимумом баллов.' },
              { title: 'Поставьте результат', text: 'Выберите работу из списка, внесите баллы и при необходимости добавьте комментарий.' }
            ]}
          />

          <div className="teacher-form-grid">
            <label>
              Предмет
              <select
                value={gradeSelection.subjectId}
                onChange={(event) => selectGradeSubject(asNumber(event.target.value))}
                disabled={subjectsLoading || noSubjects}
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
                value={gradeSelection.groupId}
                onChange={(event) => {
                  setGradeSelection((current) => ({ ...current, groupId: asNumber(event.target.value), studentId: 0 }));
                  setStudentSheet(null);
                }}
                disabled={!gradeGroups.length}
              >
                <option value="">{gradeGroups.length ? 'Выберите группу' : 'Нет групп'}</option>
                {gradeGroups.map((group) => (
                  <option key={group.id} value={group.id}>{group.name}</option>
                ))}
              </select>
            </label>
            <label>
              Студент
              <select
                value={gradeSelection.studentId}
                onChange={(event) => {
                  setGradeSelection((current) => ({ ...current, studentId: asNumber(event.target.value) }));
                  setStudentSheet(null);
                }}
                disabled={rosterLoading || !roster.length}
              >
                <option value="">{rosterLoading ? 'Загрузка...' : 'Выберите студента'}</option>
                {roster.map((student) => (
                  <option key={student.student_id} value={student.student_id}>{student.student_name}</option>
                ))}
              </select>
            </label>
          </div>

          <div className="teacher-grade-layout">
            <form className="teacher-tool-card" onSubmit={createGradeItem}>
              <h2>1. Создать работу</h2>
              <p className="teacher-muted">Работа появится в списке ниже и станет доступна для выставления баллов студентам.</p>
              <label>
                Название
                <input value={itemForm.title} onChange={(event) => setItemForm((current) => ({ ...current, title: event.target.value }))} required />
              </label>
              <label>
                Максимум баллов
                <input type="number" min="1" max="100" value={itemForm.maxScore} onChange={(event) => setItemForm((current) => ({ ...current, maxScore: asNumber(event.target.value) }))} required />
              </label>
              <label>
                Тип работы
                <select value={itemForm.itemType} onChange={(event) => setItemForm((current) => ({ ...current, itemType: event.target.value }))}>
                  {gradeTypes.map((type) => <option key={type.value} value={type.value}>{type.label}</option>)}
                </select>
              </label>
              <label>
                Срок сдачи
                <input type="date" value={itemForm.deadline} onChange={(event) => setItemForm((current) => ({ ...current, deadline: event.target.value }))} />
              </label>
              <button type="submit" className="teacher-primary" disabled={gradesLoading || !gradeSelection.subjectId}>Добавить</button>
            </form>

            <form className="teacher-tool-card" onSubmit={saveGrade}>
              <h2>2. Поставить баллы студенту</h2>
              <p className="teacher-muted">{selectedStudent ? selectedStudent.student_name : 'Выберите студента в панели выше'}</p>
              <label>
                Контрольная точка
                <select value={gradeForm.itemId} onChange={(event) => setGradeForm((current) => ({ ...current, itemId: event.target.value }))} required>
                  <option value="">Выберите работу</option>
                  {gradeItems.map((item) => <option key={item.item_id} value={item.item_id}>{item.title} · {item.max_score} б.</option>)}
                </select>
              </label>
              <label>
                Баллы
                <input type="number" min="0" value={gradeForm.score} onChange={(event) => setGradeForm((current) => ({ ...current, score: asNumber(event.target.value) }))} required />
              </label>
              <label>
                Дата занятия
                <input type="date" value={gradeForm.sessionDate} onChange={(event) => setGradeForm((current) => ({ ...current, sessionDate: event.target.value }))} />
              </label>
              <label>
                Комментарий
                <textarea value={gradeForm.comment} onChange={(event) => setGradeForm((current) => ({ ...current, comment: event.target.value }))} />
              </label>
              <button type="submit" className="teacher-primary" disabled={gradesLoading || !gradeItems.length || !gradeSelection.studentId}>Сохранить оценку</button>
            </form>
          </div>

          <div className="teacher-grade-dashboard">
            <section className="teacher-panel">
              <div className="teacher-panel-head">
                <h2>Работы предмета</h2>
                <span>{gradeItemsTotal}/100 баллов</span>
              </div>
              <div className="teacher-work-list">
                {gradeItems.length ? gradeItems.map((item) => (
                  <button
                    type="button"
                    className={String(item.item_id) === String(gradeForm.itemId) ? 'is-selected' : ''}
                    key={item.item_id}
                    onClick={() => setGradeForm((current) => ({ ...current, itemId: String(item.item_id) }))}
                  >
                    <span>{item.title}</span>
                    <strong>{item.max_score} б.</strong>
                    <small>{formatDateTime(item.deadline)}</small>
                  </button>
                )) : <p className="teacher-empty">Контрольные точки пока не загружены.</p>}
              </div>
            </section>

            <section className="teacher-panel">
              <div className="teacher-panel-head">
                <h2>Ведомость студента</h2>
                <button type="button" className="teacher-secondary" onClick={loadStudentSheet} disabled={gradesLoading || !gradeSelection.studentId}>Показать</button>
              </div>
              {radarLoading ? <p className="teacher-empty">Загрузка диаграммы...</p> : <SubjectBars items={studentRadar} />}
              {studentSheet ? (
                <>
                  <div className="teacher-stat-grid is-three">
                    <StatCard label="Набрано" value={studentSheet.summary?.current_score || 0} />
                    <StatCard label="Максимум" value={studentSheet.summary?.total_max || 0} />
                    <StatCard label="Ожидалось к сроку" value={studentSheet.summary?.passed_max || 0} />
                  </div>
                  <div className="teacher-table-wrap">
                    <table className="teacher-table">
                      <thead>
                        <tr>
                          <th>Работа</th>
                          <th>Баллы</th>
                          <th>Максимум</th>
                          <th>Дата</th>
                        </tr>
                      </thead>
                      <tbody>
                        {studentSheet.grades?.length ? studentSheet.grades.map((grade) => (
                          <tr key={grade.item_id}>
                            <td>{grade.title}</td>
                            <td>{grade.score}</td>
                            <td>{grade.max_score}</td>
                            <td>{formatDateTime(grade.graded_at)}</td>
                          </tr>
                        )) : <tr><td colSpan={4}>Оценок пока нет</td></tr>}
                      </tbody>
                    </table>
                  </div>
                </>
              ) : (
                <p className="teacher-empty">Выберите студента и нажмите «Показать».</p>
              )}
            </section>
          </div>
        </div>
      )}
    </section>
  );
};

export default TeacherPage;
