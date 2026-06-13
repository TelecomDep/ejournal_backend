import React, { useEffect, useMemo, useState } from 'react';
import api from '../services/api';
import './TeacherAccount.css';

const gradeTypes = [
  { value: 'current', label: 'Текущая работа' },
  { value: 'laboratory', label: 'Лабораторная' },
  { value: 'test', label: 'Тест' },
  { value: 'exam', label: 'Экзамен' },
  { value: 'project', label: 'Проект' }
];

const formatDateTime = (value) => {
  if (!value) {
    return 'не указан';
  }
  return new Date(value).toLocaleString('ru-RU');
};

const formatDateForAPI = (value) => {
  if (!value) {
    return undefined;
  }
  return new Date(`${value}T23:59:00`).toISOString();
};

// Возвращает список групп предмета в виде [{ id, name }].
// Бэкенд отдаёт поле groups; group_ids оставлен как фолбэк для совместимости.
const groupsOf = (subject) => {
  if (!subject) {
    return [];
  }
  if (Array.isArray(subject.groups) && subject.groups.length > 0) {
    return subject.groups
      .map((group) => ({ id: Number(group.id), name: group.name || `Группа ${group.id}` }))
      .filter((group) => Number.isFinite(group.id) && group.id > 0);
  }
  return (Array.isArray(subject.group_ids) ? subject.group_ids : [])
    .map((id) => ({ id: Number(id), name: `Группа ${id}` }))
    .filter((group) => Number.isFinite(group.id) && group.id > 0);
};

const TeacherAccount = ({ userData, onLogout, token }) => {
  const displayName = userData?.teacher_name || userData?.name || userData?.login || 'Преподаватель';

  const [activeTab, setActiveTab] = useState('attendance');
  const [sessionForm, setSessionForm] = useState({
    subjectId: 0,
    groupIds: [],
    lessonName: 'Занятие',
    expiresMinutes: 20
  });
  // Выбор для вкладки «Статистика группы»: предмет + группа.
  const [statsSelection, setStatsSelection] = useState({ subjectId: 0, groupId: 0 });
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
    sessionId: ''
  });
  // Единый каскадный выбор для вкладки «Оценки»: предмет → группа → студент.
  const [gradeSelection, setGradeSelection] = useState({ subjectId: 0, groupId: 0, studentId: 0 });
  const [roster, setRoster] = useState([]);
  const [rosterLoading, setRosterLoading] = useState(false);

  const [teacherSubjects, setTeacherSubjects] = useState([]);
  const [subjectsLoading, setSubjectsLoading] = useState(false);

  const [sessionLoading, setSessionLoading] = useState(false);
  const [statsLoading, setStatsLoading] = useState(false);
  const [gradesLoading, setGradesLoading] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [sessionResult, setSessionResult] = useState(null);
  const [statsResult, setStatsResult] = useState(null);
  const [gradeItemsResult, setGradeItemsResult] = useState(null);
  const [studentSheet, setStudentSheet] = useState(null);

  const gradeItems = gradeItemsResult?.items || [];
  const selectedSessionSubject = useMemo(
    () => teacherSubjects.find((subject) => Number(subject.subject_id) === Number(sessionForm.subjectId)),
    [teacherSubjects, sessionForm.subjectId]
  );
  const sessionGroups = useMemo(() => groupsOf(selectedSessionSubject), [selectedSessionSubject]);
  const gradeItemsTotal = useMemo(
    () => gradeItems.reduce((sum, item) => sum + Number(item.max_score || 0), 0),
    [gradeItems]
  );

  const gradeSubject = useMemo(
    () => teacherSubjects.find((subject) => Number(subject.subject_id) === Number(gradeSelection.subjectId)),
    [teacherSubjects, gradeSelection.subjectId]
  );
  const gradeGroups = useMemo(() => groupsOf(gradeSubject), [gradeSubject]);
  const statsSubject = useMemo(
    () => teacherSubjects.find((subject) => Number(subject.subject_id) === Number(statsSelection.subjectId)),
    [teacherSubjects, statsSelection.subjectId]
  );
  const statsGroups = useMemo(() => groupsOf(statsSubject), [statsSubject]);
  const selectedStudent = useMemo(
    () => roster.find((student) => Number(student.student_id) === Number(gradeSelection.studentId)),
    [roster, gradeSelection.studentId]
  );

  const parseNumber = (value) => {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  };

  // Загрузка предметов преподавателя + инициализация выборов по первому предмету.
  useEffect(() => {
    let active = true;

    const loadTeacherSubjects = async () => {
      setSubjectsLoading(true);
      try {
        const response = await api.getTeacherSubjects(token);
        const subjects = Array.isArray(response?.subjects) ? response.subjects : [];
        if (!active) {
          return;
        }
        setTeacherSubjects(subjects);

        if (subjects.length > 0) {
          const first = subjects[0];
          const firstSubjectID = Number(first.subject_id);
          const firstGroups = groupsOf(first);
          const firstGroupID = firstGroups[0]?.id || 0;

          setSessionForm((current) => ({
            ...current,
            subjectId: current.subjectId || firstSubjectID,
            groupIds: current.groupIds?.length ? current.groupIds : firstGroups.map((group) => group.id)
          }));
          setGradeSelection((current) => ({
            subjectId: current.subjectId || firstSubjectID,
            groupId: current.groupId || firstGroupID,
            studentId: current.studentId || 0
          }));
          setStatsSelection((current) => ({
            subjectId: current.subjectId || firstSubjectID,
            groupId: current.groupId || firstGroupID
          }));
        }
      } catch (err) {
        if (active) {
          setError(api.getErrorMessage(err, 'Не удалось загрузить предметы преподавателя'));
        }
      } finally {
        if (active) {
          setSubjectsLoading(false);
        }
      }
    };

    loadTeacherSubjects();
    return () => {
      active = false;
    };
  }, [token]);

  // Подгрузка состава группы (студентов) при смене предмета/группы во вкладке «Оценки».
  useEffect(() => {
    if (!gradeSelection.subjectId || !gradeSelection.groupId) {
      setRoster([]);
      return undefined;
    }

    let active = true;
    setRosterLoading(true);
    api.getGroupStats(token, gradeSelection.groupId, gradeSelection.subjectId)
      .then((response) => {
        if (!active) {
          return;
        }
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
        if (active) {
          setRoster([]);
        }
      })
      .finally(() => {
        if (active) {
          setRosterLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, [token, gradeSelection.subjectId, gradeSelection.groupId]);

  // Контрольные точки зависят только от предмета — перегружаем при его смене.
  useEffect(() => {
    if (!gradeSelection.subjectId) {
      return;
    }
    handleLoadGradeItems(gradeSelection.subjectId, true);
  }, [gradeSelection.subjectId]);

  const clearFeedback = () => {
    setError('');
    setMessage('');
  };

  const handleSessionInputChange = (event) => {
    const { name, value } = event.target;
    if (name === 'subjectId') {
      const selectedSubjectID = parseNumber(value);
      const selectedSubject = teacherSubjects.find((subject) => Number(subject.subject_id) === selectedSubjectID);
      const nextGroupIDs = groupsOf(selectedSubject).map((group) => group.id);
      setSessionForm((current) => ({
        ...current,
        subjectId: selectedSubjectID,
        groupIds: nextGroupIDs
      }));
      return;
    }

    setSessionForm((current) => ({
      ...current,
      [name]: name === 'expiresMinutes' ? parseNumber(value) : value
    }));
  };

  // Включение/выключение группы чекбоксом во вкладке «Посещаемость».
  const handleSessionGroupToggle = (groupId) => {
    setSessionForm((current) => {
      const exists = current.groupIds.includes(groupId);
      return {
        ...current,
        groupIds: exists
          ? current.groupIds.filter((id) => id !== groupId)
          : [...current.groupIds, groupId]
      };
    });
  };

  // Каскад вкладки «Статистика группы».
  const handleStatsSubjectChange = (event) => {
    const subjectId = parseNumber(event.target.value);
    const subject = teacherSubjects.find((item) => Number(item.subject_id) === subjectId);
    const firstGroup = groupsOf(subject)[0]?.id || 0;
    setStatsSelection({ subjectId, groupId: firstGroup });
  };

  const handleStatsGroupChange = (event) => {
    setStatsSelection((current) => ({ ...current, groupId: parseNumber(event.target.value) }));
  };

  // Каскад вкладки «Оценки».
  const handleGradeSubjectChange = (event) => {
    const subjectId = parseNumber(event.target.value);
    const subject = teacherSubjects.find((item) => Number(item.subject_id) === subjectId);
    const firstGroup = groupsOf(subject)[0]?.id || 0;
    setGradeSelection({ subjectId, groupId: firstGroup, studentId: 0 });
  };

  const handleGradeGroupChange = (event) => {
    setGradeSelection((current) => ({ ...current, groupId: parseNumber(event.target.value), studentId: 0 }));
  };

  const handleStudentChange = (event) => {
    setGradeSelection((current) => ({ ...current, studentId: parseNumber(event.target.value) }));
  };

  const handleItemInputChange = (event) => {
    const { name, value } = event.target;
    setItemForm((current) => ({
      ...current,
      [name]: name === 'maxScore' ? parseNumber(value) : value
    }));
  };

  const handleGradeInputChange = (event) => {
    const { name, value } = event.target;
    setGradeForm((current) => ({
      ...current,
      [name]: name === 'score' ? parseNumber(value) : value
    }));
  };

  const handleCreateSession = async (event) => {
    event.preventDefault();
    clearFeedback();
    setSessionResult(null);
    setSessionLoading(true);

    try {
      if (!parseNumber(sessionForm.subjectId)) {
        setError('Выберите предмет из списка ваших пар.');
        return;
      }
      if (!sessionForm.groupIds.length) {
        setError('Отметьте хотя бы одну группу.');
        return;
      }
      const groupNames = sessionGroups
        .filter((group) => sessionForm.groupIds.includes(group.id))
        .map((group) => group.name)
        .join(', ') || sessionForm.groupIds.join(', ');
      const confirmText = `Открыть отметку посещаемости «${sessionForm.lessonName}» для групп: ${groupNames} на ${sessionForm.expiresMinutes} мин?`;
      if (!window.confirm(confirmText)) {
        return;
      }
      const response = await api.createAttendanceLink(
        token,
        sessionForm.subjectId,
        sessionForm.groupIds,
        sessionForm.lessonName,
        sessionForm.expiresMinutes
      );
      setSessionResult(response);
      setMessage('Ссылка для отметки посещаемости создана.');
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось создать ссылку посещаемости'));
    } finally {
      setSessionLoading(false);
    }
  };

  const handleGetStats = async (event) => {
    event.preventDefault();
    clearFeedback();
    setStatsResult(null);

    if (!statsSelection.subjectId) {
      setError('Выберите предмет.');
      return;
    }
    if (!statsSelection.groupId) {
      setError('Выберите группу.');
      return;
    }

    setStatsLoading(true);
    try {
      const response = await api.getGroupPerformance(
        token,
        parseNumber(statsSelection.groupId),
        parseNumber(statsSelection.subjectId)
      );
      setStatsResult(response);
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось загрузить сводку по группе'));
    } finally {
      setStatsLoading(false);
    }
  };

  const handleLoadGradeItems = async (subjectId = gradeSelection.subjectId, silent = false) => {
    if (!silent) {
      clearFeedback();
    }
    setGradesLoading(true);

    try {
      const response = await api.getTeacherGradeItems(token, parseNumber(subjectId));
      setGradeItemsResult(response);
      if (response?.items?.length) {
        setGradeForm((current) => {
          const stillValid = response.items.some((item) => String(item.item_id) === String(current.itemId));
          return stillValid ? current : { ...current, itemId: String(response.items[0].item_id) };
        });
      } else {
        setGradeForm((current) => ({ ...current, itemId: '' }));
      }
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось загрузить контрольные точки'));
    } finally {
      setGradesLoading(false);
    }
  };

  const handleCreateGradeItem = async (event) => {
    event.preventDefault();
    clearFeedback();

    if (!gradeSelection.subjectId) {
      setError('Сначала выберите предмет.');
      return;
    }

    const title = itemForm.title.trim();
    if (!title) {
      setError('Введите название контрольной точки.');
      return;
    }
    const confirmText = `Создать контрольную точку «${title}» (${parseNumber(itemForm.maxScore)} б.) по предмету «${gradeSubject ? gradeSubject.subject_name : ''}»?`;
    if (!window.confirm(confirmText)) {
      return;
    }

    setGradesLoading(true);
    try {
      await api.createGradeItem(token, {
        subject_id: parseNumber(gradeSelection.subjectId),
        title,
        max_score: parseNumber(itemForm.maxScore),
        item_type: itemForm.itemType,
        deadline: formatDateForAPI(itemForm.deadline)
      });
      setMessage(`Контрольная точка «${title}» добавлена.`);
      await handleLoadGradeItems(gradeSelection.subjectId, true);
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось добавить контрольную точку'));
    } finally {
      setGradesLoading(false);
    }
  };

  const handleSaveGrade = async (event) => {
    event.preventDefault();
    clearFeedback();

    if (!gradeSelection.studentId) {
      setError('Выберите студента из группы.');
      return;
    }
    if (!gradeForm.itemId) {
      setError('Выберите контрольную точку.');
      return;
    }

    const selectedItem = gradeItems.find((item) => String(item.item_id) === String(gradeForm.itemId));
    const studentName = selectedStudent ? selectedStudent.student_name : `студенту #${gradeSelection.studentId}`;
    const workTitle = selectedItem ? selectedItem.title : 'работу';
    const confirmText = `Поставить ${studentName} ${parseNumber(gradeForm.score)} б. за «${workTitle}»?`;
    if (!window.confirm(confirmText)) {
      return;
    }

    setGradesLoading(true);
    try {
      const payload = {
        student_id: parseNumber(gradeSelection.studentId),
        item_id: parseNumber(gradeForm.itemId),
        score: parseNumber(gradeForm.score)
      };
      if (gradeForm.comment.trim()) {
        payload.comment = gradeForm.comment.trim();
      }
      if (gradeForm.sessionId) {
        payload.session_id = parseNumber(gradeForm.sessionId);
      }

      await api.saveStudentGrade(token, payload);
      setMessage(`Оценка сохранена: ${studentName} — ${parseNumber(gradeForm.score)} б. за «${workTitle}».`);
      if (studentSheet && Number(studentSheet.student_id) === payload.student_id) {
        await handleLoadStudentSheet(null, true);
      }
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось сохранить оценку'));
    } finally {
      setGradesLoading(false);
    }
  };

  const handleLoadStudentSheet = async (event, silent = false) => {
    if (event) {
      event.preventDefault();
    }
    if (!silent) {
      clearFeedback();
    }

    if (!gradeSelection.studentId || !gradeSelection.subjectId) {
      setError('Выберите предмет, группу и студента.');
      return;
    }

    setGradesLoading(true);
    try {
      const response = await api.getTeacherStudentGrades(
        token,
        parseNumber(gradeSelection.studentId),
        parseNumber(gradeSelection.subjectId)
      );
      setStudentSheet(response);
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось загрузить ведомость студента'));
    } finally {
      setGradesLoading(false);
    }
  };

  const copyAttendanceLink = async () => {
    if (!sessionResult?.join_url) {
      return;
    }
    try {
      await navigator.clipboard.writeText(sessionResult.join_url);
      setMessage('Ссылка скопирована.');
    } catch {
      setError('Не удалось скопировать ссылку автоматически.');
    }
  };

  const noSubjects = !subjectsLoading && teacherSubjects.length === 0;

  return (
    <div className="teacher-account">
      <header className="teacher-header">
        <div className="teacher-info">
          <h1>Панель преподавателя</h1>
          <p className="teacher-subtitle">
            {displayName} · роль: {userData?.role || 'преподаватель'} · ID: {userData?.teacher_id || userData?.user_id || userData?.id}
          </p>
        </div>
        <div className="teacher-actions">
          <button className="logout-btn" onClick={onLogout}>Выйти</button>
        </div>
      </header>

      {(message || error) && (
        <div className={error ? 'feedback feedback-error' : 'feedback feedback-success'}>
          {error || message}
        </div>
      )}

      <nav className="teacher-tabs" aria-label="Разделы преподавателя">
        <button className={`tab-btn ${activeTab === 'attendance' ? 'active' : ''}`} onClick={() => setActiveTab('attendance')}>
          Посещаемость
        </button>
        <button className={`tab-btn ${activeTab === 'statistics' ? 'active' : ''}`} onClick={() => setActiveTab('statistics')}>
          Статистика группы
        </button>
        <button className={`tab-btn ${activeTab === 'grades' ? 'active' : ''}`} onClick={() => setActiveTab('grades')}>
          Оценки
        </button>
      </nav>

      {activeTab === 'attendance' && (
        <section className="teacher-section">
          <div className="section-heading">
            <h2>Ссылка для отметки посещаемости</h2>
            <p>Создайте короткую ссылку, по которой студенты отметятся на текущем занятии.</p>
          </div>

          <form onSubmit={handleCreateSession} className="teacher-form">
            <label className="form-group" htmlFor="subjectId">
              Пара (предмет)
              <select
                id="subjectId"
                name="subjectId"
                value={sessionForm.subjectId}
                onChange={handleSessionInputChange}
                required
                disabled={subjectsLoading || teacherSubjects.length === 0}
              >
                <option value="">
                  {subjectsLoading ? 'Загрузка пар...' : 'Выберите пару'}
                </option>
                {teacherSubjects.map((subject) => (
                  <option key={subject.subject_id} value={subject.subject_id}>
                    {subject.subject_name} (ID: {subject.subject_id})
                  </option>
                ))}
              </select>
            </label>
            {noSubjects && (
              <p className="helper-text">
                Для вашего профиля пока не найдено пар в расписании. Обратитесь к администратору.
              </p>
            )}
            <div className="form-group form-group-wide">
              <span>Группы</span>
              {sessionGroups.length > 0 ? (
                <div className="group-checklist">
                  {sessionGroups.map((group) => (
                    <label
                      key={group.id}
                      className={`group-chip ${sessionForm.groupIds.includes(group.id) ? 'checked' : ''}`}
                    >
                      <input
                        type="checkbox"
                        checked={sessionForm.groupIds.includes(group.id)}
                        onChange={() => handleSessionGroupToggle(group.id)}
                      />
                      {group.name}
                    </label>
                  ))}
                </div>
              ) : (
                <p className="helper-text">У выбранной пары нет групп в расписании.</p>
              )}
            </div>
            <label className="form-group" htmlFor="lessonName">
              Название занятия
              <input id="lessonName" type="text" name="lessonName" value={sessionForm.lessonName} onChange={handleSessionInputChange} placeholder="Практика по сетям" required />
            </label>
            <label className="form-group" htmlFor="expiresMinutes">
              Ссылка активна, минут
              <input id="expiresMinutes" type="number" name="expiresMinutes" value={sessionForm.expiresMinutes} onChange={handleSessionInputChange} min="1" max="180" required />
            </label>

            <div className="form-actions">
              <button type="submit" className="submit-btn" disabled={sessionLoading}>
                {sessionLoading ? 'Создаем...' : 'Создать ссылку'}
              </button>
              <button type="button" className="secondary-btn" onClick={copyAttendanceLink} disabled={!sessionResult?.join_url}>
                Скопировать ссылку
              </button>
            </div>
          </form>

          {sessionResult && (
            <div className="result-panel">
              <h3>Сессия создана</h3>
              <div className="result-grid">
                <div>
                  <span>Ссылка для студентов</span>
                  <strong className="mono-value">{sessionResult.join_url || 'не получена'}</strong>
                </div>
                <div>
                  <span>ID занятия</span>
                  <strong>{sessionResult.lesson_id || 'не указан'}</strong>
                </div>
                <div>
                  <span>Истекает</span>
                  <strong>{formatDateTime(sessionResult.expires_at)}</strong>
                </div>
              </div>
            </div>
          )}
        </section>
      )}

      {activeTab === 'statistics' && (
        <section className="teacher-section">
          <div className="section-heading">
            <h2>Сводка по группе</h2>
            <p>Выберите предмет и группу — увидите посещаемость и успеваемость каждого студента в одной таблице.</p>
          </div>

          <form onSubmit={handleGetStats} className="teacher-form compact-form">
            <label className="form-group" htmlFor="statsSubject">
              Предмет
              <select
                id="statsSubject"
                value={statsSelection.subjectId}
                onChange={handleStatsSubjectChange}
                disabled={subjectsLoading || teacherSubjects.length === 0}
                required
              >
                <option value="">{subjectsLoading ? 'Загрузка...' : 'Выберите предмет'}</option>
                {teacherSubjects.map((subject) => (
                  <option key={subject.subject_id} value={subject.subject_id}>
                    {subject.subject_name}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-group" htmlFor="statsGroup">
              Группа
              <select
                id="statsGroup"
                value={statsSelection.groupId}
                onChange={handleStatsGroupChange}
                disabled={statsGroups.length === 0}
                required
              >
                <option value="">{statsGroups.length ? 'Выберите группу' : 'Нет групп'}</option>
                {statsGroups.map((group) => (
                  <option key={group.id} value={group.id}>
                    {group.name}
                  </option>
                ))}
              </select>
            </label>
            <div className="form-actions">
              <button type="submit" className="submit-btn" disabled={statsLoading || !statsSelection.subjectId || !statsSelection.groupId}>
                {statsLoading ? 'Загружаем...' : 'Показать сводку'}
              </button>
            </div>
          </form>

          {statsResult && (
            <>
              <div className="result-panel">
                <h3>
                  {statsResult.group_name || `Группа ${statsResult.group_id}`}
                  {statsResult.subject_name ? ` · ${statsResult.subject_name}` : ''}
                </h3>
                <div className="score-summary">
                  <div>
                    <span>Студентов</span>
                    <strong>{statsResult.summary?.students_count ?? 0}</strong>
                  </div>
                  <div>
                    <span>Занятий проведено</span>
                    <strong>{statsResult.summary?.sessions_count ?? 0}</strong>
                  </div>
                  <div>
                    <span>Средняя посещаемость</span>
                    <strong>{Number(statsResult.summary?.avg_attendance_percent || 0).toFixed(0)}%</strong>
                  </div>
                  <div>
                    <span>Средняя успеваемость</span>
                    <strong>{Number(statsResult.summary?.avg_grade_percent || 0).toFixed(0)}%</strong>
                  </div>
                </div>
              </div>

              <div className="table-wrap">
                <table className="stats-table">
                  <thead>
                    <tr>
                      <th>ФИО</th>
                      <th>Посещаемость</th>
                      <th>Успеваемость</th>
                      <th>Баллы</th>
                    </tr>
                  </thead>
                  <tbody>
                    {statsResult.students?.length > 0 ? (
                      statsResult.students.map((student) => {
                        const attendance = Number(student.attendance_percent);
                        const grade = Number(student.grade_percent);
                        return (
                          <tr key={student.student_id}>
                            <td>{student.student_name}</td>
                            <td>
                              {student.attended_sessions}/{student.total_sessions}
                              {' · '}
                              <span className={`badge ${attendance >= 75 ? 'badge-ok' : attendance >= 50 ? 'badge-warn' : 'badge-bad'}`}>
                                {Number.isFinite(attendance) ? `${attendance.toFixed(0)}%` : '—'}
                              </span>
                            </td>
                            <td>
                              <span className={`badge ${grade >= 75 ? 'badge-ok' : grade >= 50 ? 'badge-warn' : 'badge-bad'}`}>
                                {Number.isFinite(grade) ? `${grade.toFixed(0)}%` : '—'}
                              </span>
                            </td>
                            <td>{student.current_score} / {student.total_max}</td>
                          </tr>
                        );
                      })
                    ) : (
                      <tr><td colSpan={4}>В группе пока нет студентов</td></tr>
                    )}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </section>
      )}

      {activeTab === 'grades' && (
        <section className="teacher-section">
          <div className="section-heading">
            <h2>Оценки и балльно-рейтинговая система</h2>
            <p>Создавайте контрольные точки до 100 баллов по предмету и выставляйте результаты студентам.</p>
          </div>

          <div className="selection-bar">
            <label className="form-group" htmlFor="gradeSubject">
              Предмет
              <select
                id="gradeSubject"
                value={gradeSelection.subjectId}
                onChange={handleGradeSubjectChange}
                disabled={subjectsLoading || teacherSubjects.length === 0}
              >
                <option value="">{subjectsLoading ? 'Загрузка...' : 'Выберите предмет'}</option>
                {teacherSubjects.map((subject) => (
                  <option key={subject.subject_id} value={subject.subject_id}>
                    {subject.subject_name}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-group" htmlFor="gradeGroup">
              Группа
              <select
                id="gradeGroup"
                value={gradeSelection.groupId}
                onChange={handleGradeGroupChange}
                disabled={gradeGroups.length === 0}
              >
                <option value="">{gradeGroups.length ? 'Выберите группу' : 'Нет групп'}</option>
                {gradeGroups.map((group) => (
                  <option key={group.id} value={group.id}>
                    {group.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-group" htmlFor="gradeStudent">
              Студент
              <select
                id="gradeStudent"
                value={gradeSelection.studentId}
                onChange={handleStudentChange}
                disabled={rosterLoading || roster.length === 0}
              >
                <option value="">
                  {rosterLoading ? 'Загрузка студентов...' : (roster.length ? 'Выберите студента' : 'Сначала выберите группу')}
                </option>
                {roster.map((student) => (
                  <option key={student.student_id} value={student.student_id}>
                    {student.student_name} (ID: {student.student_id})
                  </option>
                ))}
              </select>
            </label>
          </div>
          {noSubjects && (
            <p className="helper-text">
              Для вашего профиля пока не найдено пар в расписании. Обратитесь к администратору.
            </p>
          )}

          <div className="grades-layout">
            <form onSubmit={handleCreateGradeItem} className="tool-panel">
              <h3>Новая контрольная точка</h3>
              <p className="helper-text">
                Предмет: {gradeSubject ? gradeSubject.subject_name : 'не выбран'}
              </p>
              <label className="form-group" htmlFor="gradeTitle">
                Название
                <input id="gradeTitle" type="text" name="title" value={itemForm.title} onChange={handleItemInputChange} required />
              </label>
              <label className="form-group" htmlFor="gradeMaxScore">
                Максимум баллов
                <input id="gradeMaxScore" type="number" name="maxScore" min="1" max="100" value={itemForm.maxScore} onChange={handleItemInputChange} required />
              </label>
              <label className="form-group" htmlFor="gradeItemType">
                Тип работы
                <select id="gradeItemType" name="itemType" value={itemForm.itemType} onChange={handleItemInputChange}>
                  {gradeTypes.map((type) => (
                    <option key={type.value} value={type.value}>{type.label}</option>
                  ))}
                </select>
              </label>
              <label className="form-group" htmlFor="gradeDeadline">
                Срок сдачи
                <input id="gradeDeadline" type="date" name="deadline" value={itemForm.deadline} onChange={handleItemInputChange} />
              </label>
              <div className="form-actions">
                <button type="submit" className="submit-btn" disabled={gradesLoading || !gradeSelection.subjectId}>Добавить</button>
                <button type="button" className="secondary-btn" onClick={() => handleLoadGradeItems()} disabled={gradesLoading}>Обновить список</button>
              </div>
            </form>

            <form onSubmit={handleSaveGrade} className="tool-panel">
              <h3>Выставить баллы</h3>
              {selectedStudent ? (
                <div className="target-chip">
                  Оценка для: <strong>{selectedStudent.student_name}</strong>
                  <span className="target-chip-meta">ID {selectedStudent.student_id}{gradeSubject ? ` · ${gradeSubject.subject_name}` : ''}</span>
                </div>
              ) : (
                <p className="helper-text">Выберите студента в панели вверху, чтобы выставить ему баллы.</p>
              )}
              <label className="form-group" htmlFor="gradeItemId">
                Контрольная точка
                <select id="gradeItemId" name="itemId" value={gradeForm.itemId} onChange={handleGradeInputChange} required>
                  <option value="">Выберите работу</option>
                  {gradeItems.map((item) => (
                    <option key={item.item_id} value={item.item_id}>
                      {item.title} · {item.max_score} б.
                    </option>
                  ))}
                </select>
              </label>
              <label className="form-group" htmlFor="gradeScore">
                Баллы
                <input id="gradeScore" type="number" name="score" min="0" value={gradeForm.score} onChange={handleGradeInputChange} required />
              </label>
              <label className="form-group" htmlFor="gradeSessionId">
                ID занятия
                <input id="gradeSessionId" type="number" name="sessionId" min="1" value={gradeForm.sessionId} onChange={handleGradeInputChange} placeholder="необязательно" />
              </label>
              <label className="form-group form-group-wide" htmlFor="gradeComment">
                Комментарий
                <textarea id="gradeComment" name="comment" value={gradeForm.comment} onChange={handleGradeInputChange} placeholder="Например: защита сдана с замечаниями" />
              </label>
              <div className="form-actions">
                <button type="submit" className="submit-btn" disabled={gradesLoading || !gradeItems.length || !gradeSelection.studentId}>Сохранить оценку</button>
              </div>
            </form>
          </div>

          <div className="grades-dashboard">
            <div className="list-panel">
              <div className="list-panel-header">
                <h3>Контрольные точки</h3>
                <span>{gradeItemsTotal}/100 баллов</span>
              </div>
              {gradeItems.length > 0 ? (
                <div className="grade-items-list">
                  {gradeItems.map((item) => (
                    <button
                      type="button"
                      className={`grade-item-row ${String(item.item_id) === String(gradeForm.itemId) ? 'selected' : ''}`}
                      key={item.item_id}
                      onClick={() => setGradeForm((current) => ({ ...current, itemId: String(item.item_id) }))}
                    >
                      <span>{item.title}</span>
                      <strong>{item.max_score} б.</strong>
                      <small>{formatDateTime(item.deadline)}</small>
                    </button>
                  ))}
                </div>
              ) : (
                <p className="empty-state">Выберите предмет, чтобы увидеть контрольные точки.</p>
              )}
            </div>

            <div className="list-panel">
              <form onSubmit={handleLoadStudentSheet} className="sheet-toolbar">
                <h3>Ведомость студента</h3>
                <p className="helper-text">
                  {selectedStudent ? selectedStudent.student_name : 'студент не выбран'}
                  {gradeSubject ? ` · ${gradeSubject.subject_name}` : ''}
                </p>
                <button type="submit" className="secondary-btn" disabled={gradesLoading || !gradeSelection.studentId}>Показать</button>
              </form>

              {studentSheet ? (
                <>
                  <div className="score-summary">
                    <div title="Сумма баллов, которые студент уже набрал">
                      <span>Набрано баллов</span>
                      <strong>{studentSheet.summary?.current_score || 0}</strong>
                    </div>
                    <div title="Максимум баллов по всем работам предмета (план на семестр)">
                      <span>Максимум по предмету</span>
                      <strong>{studentSheet.summary?.total_max || 0}</strong>
                    </div>
                    <div title="Максимум баллов по работам, срок сдачи которых уже прошёл — сравните с «Набрано», чтобы понять отставание">
                      <span>Ожидалось к сроку</span>
                      <strong>{studentSheet.summary?.passed_max || 0}</strong>
                    </div>
                  </div>
                  <p className="helper-text">
                    «Ожидалось к сроку» — сколько баллов уже можно было набрать по работам с прошедшим дедлайном.
                    Если «Набрано» заметно меньше — у студента отставание.
                  </p>
                  <div className="table-wrap">
                    <table className="stats-table">
                      <thead>
                        <tr>
                          <th>Работа</th>
                          <th>Баллы</th>
                          <th>Максимум</th>
                          <th>Дата</th>
                        </tr>
                      </thead>
                      <tbody>
                        {studentSheet.grades?.length > 0 ? (
                          studentSheet.grades.map((grade) => (
                            <tr key={grade.item_id}>
                              <td>{grade.title}</td>
                              <td>{grade.score}</td>
                              <td>{grade.max_score}</td>
                              <td>{formatDateTime(grade.graded_at)}</td>
                            </tr>
                          ))
                        ) : (
                          <tr><td colSpan={4}>Оценок пока нет</td></tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                </>
              ) : (
                <p className="empty-state">Выберите студента вверху и нажмите «Показать».</p>
              )}
            </div>
          </div>
        </section>
      )}
    </div>
  );
};

export default TeacherAccount;
