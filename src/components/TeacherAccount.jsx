import React, { useMemo, useState } from 'react';
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

const TeacherAccount = ({ userData, onLogout, token }) => {
  const displayName = userData?.teacher_name || userData?.name || userData?.login || 'Преподаватель';

  const [activeTab, setActiveTab] = useState('attendance');
  const [sessionForm, setSessionForm] = useState({
    subjectId: 1,
    groupIds: [1],
    lessonName: 'Занятие',
    expiresMinutes: 20
  });
  const [statsForm, setStatsForm] = useState({
    groupId: 1,
    subjectId: ''
  });
  const [itemForm, setItemForm] = useState({
    subjectId: 1,
    title: 'Лабораторная работа 1',
    maxScore: 10,
    itemType: 'laboratory',
    deadline: ''
  });
  const [gradeForm, setGradeForm] = useState({
    studentId: 1,
    itemId: '',
    score: 0,
    comment: '',
    sessionId: ''
  });
  const [sheetForm, setSheetForm] = useState({
    studentId: 1,
    subjectId: 1
  });

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
  const gradeItemsTotal = useMemo(
    () => gradeItems.reduce((sum, item) => sum + Number(item.max_score || 0), 0),
    [gradeItems]
  );

  const clearFeedback = () => {
    setError('');
    setMessage('');
  };

  const parseNumber = (value) => {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  };

  const handleSessionInputChange = (event) => {
    const { name, value } = event.target;
    if (name === 'groupIds') {
      setSessionForm((current) => ({
        ...current,
        groupIds: value.split(',').map((id) => parseInt(id.trim(), 10)).filter((id) => !Number.isNaN(id))
      }));
      return;
    }

    setSessionForm((current) => ({
      ...current,
      [name]: name === 'expiresMinutes' || name === 'subjectId' ? parseNumber(value) : value
    }));
  };

  const handleStatsInputChange = (event) => {
    const { name, value } = event.target;
    setStatsForm((current) => ({
      ...current,
      [name]: value
    }));
  };

  const handleItemInputChange = (event) => {
    const { name, value } = event.target;
    setItemForm((current) => ({
      ...current,
      [name]: name === 'subjectId' || name === 'maxScore' ? parseNumber(value) : value
    }));
  };

  const handleGradeInputChange = (event) => {
    const { name, value } = event.target;
    setGradeForm((current) => ({
      ...current,
      [name]: name === 'studentId' || name === 'score' ? parseNumber(value) : value
    }));
  };

  const handleSheetInputChange = (event) => {
    const { name, value } = event.target;
    setSheetForm((current) => ({
      ...current,
      [name]: parseNumber(value)
    }));
  };

  const handleCreateSession = async (event) => {
    event.preventDefault();
    clearFeedback();
    setSessionResult(null);
    setSessionLoading(true);

    try {
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
    setStatsLoading(true);

    try {
      const subjectId = statsForm.subjectId === '' ? null : parseNumber(statsForm.subjectId);
      const response = await api.getGroupStats(token, parseNumber(statsForm.groupId), subjectId);
      setStatsResult(response);
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось загрузить посещаемость группы'));
    } finally {
      setStatsLoading(false);
    }
  };

  const handleLoadGradeItems = async (subjectId = itemForm.subjectId, silent = false) => {
    if (!silent) {
      clearFeedback();
    }
    setGradesLoading(true);

    try {
      const response = await api.getTeacherGradeItems(token, parseNumber(subjectId));
      setGradeItemsResult(response);
      if (response?.items?.length && !gradeForm.itemId) {
        setGradeForm((current) => ({ ...current, itemId: String(response.items[0].item_id) }));
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
    setGradesLoading(true);

    try {
      await api.createGradeItem(token, {
        subject_id: parseNumber(itemForm.subjectId),
        title: itemForm.title.trim(),
        max_score: parseNumber(itemForm.maxScore),
        item_type: itemForm.itemType,
        deadline: formatDateForAPI(itemForm.deadline)
      });
      setMessage('Контрольная точка добавлена.');
      await handleLoadGradeItems(itemForm.subjectId, true);
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось добавить контрольную точку'));
    } finally {
      setGradesLoading(false);
    }
  };

  const handleSaveGrade = async (event) => {
    event.preventDefault();
    clearFeedback();
    setGradesLoading(true);

    try {
      const payload = {
        student_id: parseNumber(gradeForm.studentId),
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
      setMessage('Оценка сохранена.');
      if (studentSheet?.student_id === payload.student_id) {
        await handleLoadStudentSheet(null, sheetForm.studentId, sheetForm.subjectId, true);
      }
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось сохранить оценку'));
    } finally {
      setGradesLoading(false);
    }
  };

  const handleLoadStudentSheet = async (event, studentId = sheetForm.studentId, subjectId = sheetForm.subjectId, silent = false) => {
    if (event) {
      event.preventDefault();
    }
    if (!silent) {
      clearFeedback();
    }
    setGradesLoading(true);

    try {
      const response = await api.getTeacherStudentGrades(token, parseNumber(studentId), parseNumber(subjectId));
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
              ID предмета
              <input id="subjectId" type="number" name="subjectId" value={sessionForm.subjectId} onChange={handleSessionInputChange} min="1" required />
            </label>
            <label className="form-group" htmlFor="groupIds">
              ID групп через запятую
              <input id="groupIds" type="text" name="groupIds" value={sessionForm.groupIds.join(', ')} onChange={handleSessionInputChange} placeholder="1, 2, 3" required />
            </label>
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
            <h2>Посещаемость группы</h2>
            <p>Посмотрите, сколько занятий посетил каждый студент.</p>
          </div>

          <form onSubmit={handleGetStats} className="teacher-form compact-form">
            <label className="form-group" htmlFor="statGroupId">
              ID группы
              <input id="statGroupId" type="number" name="groupId" value={statsForm.groupId} onChange={handleStatsInputChange} min="1" required />
            </label>
            <label className="form-group" htmlFor="statSubjectId">
              ID предмета
              <input id="statSubjectId" type="number" name="subjectId" value={statsForm.subjectId} onChange={handleStatsInputChange} min="1" placeholder="Можно оставить пустым" />
            </label>
            <div className="form-actions">
              <button type="submit" className="submit-btn" disabled={statsLoading}>
                {statsLoading ? 'Загружаем...' : 'Показать посещаемость'}
              </button>
            </div>
          </form>

          {statsResult && (
            <div className="table-wrap">
              <table className="stats-table">
                <thead>
                  <tr>
                    <th>ID студента</th>
                    <th>ФИО</th>
                    <th>Посещено</th>
                    <th>Всего</th>
                    <th>Процент</th>
                  </tr>
                </thead>
                <tbody>
                  {statsResult.students?.length > 0 ? (
                    statsResult.students.map((student) => {
                      const percent = Number(student.attendance_percent);
                      return (
                        <tr key={student.student_id}>
                          <td>{student.student_id}</td>
                          <td>{student.student_name}</td>
                          <td>{student.attended_sessions}</td>
                          <td>{student.total_sessions}</td>
                          <td>{Number.isFinite(percent) ? `${percent.toFixed(1)}%` : 'нет данных'}</td>
                        </tr>
                      );
                    })
                  ) : (
                    <tr><td colSpan={5}>Данных пока нет</td></tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </section>
      )}

      {activeTab === 'grades' && (
        <section className="teacher-section">
          <div className="section-heading">
            <h2>Оценки и балльно-рейтинговая система</h2>
            <p>Создавайте контрольные точки до 100 баллов по предмету и выставляйте результаты студентам.</p>
          </div>

          <div className="grades-layout">
            <form onSubmit={handleCreateGradeItem} className="tool-panel">
              <h3>Новая контрольная точка</h3>
              <label className="form-group" htmlFor="gradeSubjectId">
                ID предмета
                <input id="gradeSubjectId" type="number" name="subjectId" min="1" value={itemForm.subjectId} onChange={handleItemInputChange} required />
              </label>
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
                <button type="submit" className="submit-btn" disabled={gradesLoading}>Добавить</button>
                <button type="button" className="secondary-btn" onClick={() => handleLoadGradeItems()} disabled={gradesLoading}>Обновить список</button>
              </div>
            </form>

            <form onSubmit={handleSaveGrade} className="tool-panel">
              <h3>Выставить баллы</h3>
              <label className="form-group" htmlFor="gradeStudentId">
                ID студента
                <input id="gradeStudentId" type="number" name="studentId" min="1" value={gradeForm.studentId} onChange={handleGradeInputChange} required />
              </label>
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
                <button type="submit" className="submit-btn" disabled={gradesLoading || !gradeItems.length}>Сохранить оценку</button>
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
                <p className="empty-state">Загрузите или создайте контрольные точки по предмету.</p>
              )}
            </div>

            <div className="list-panel">
              <form onSubmit={handleLoadStudentSheet} className="sheet-toolbar">
                <h3>Ведомость студента</h3>
                <label htmlFor="sheetStudentId">
                  ID студента
                  <input id="sheetStudentId" type="number" name="studentId" min="1" value={sheetForm.studentId} onChange={handleSheetInputChange} />
                </label>
                <label htmlFor="sheetSubjectId">
                  ID предмета
                  <input id="sheetSubjectId" type="number" name="subjectId" min="1" value={sheetForm.subjectId} onChange={handleSheetInputChange} />
                </label>
                <button type="submit" className="secondary-btn" disabled={gradesLoading}>Показать</button>
              </form>

              {studentSheet ? (
                <>
                  <div className="score-summary">
                    <div><span>Набрано</span><strong>{studentSheet.summary?.current_score || 0}</strong></div>
                    <div><span>План</span><strong>{studentSheet.summary?.total_max || 0}</strong></div>
                    <div><span>Прошло</span><strong>{studentSheet.summary?.passed_max || 0}</strong></div>
                  </div>
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
                <p className="empty-state">Введите ID студента и предмета, чтобы показать ведомость.</p>
              )}
            </div>
          </div>
        </section>
      )}
    </div>
  );
};

export default TeacherAccount;
