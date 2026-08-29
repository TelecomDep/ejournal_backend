import React, { useEffect, useMemo, useState } from 'react';
import api from '../../services/api';

const LESSON_TYPES = [
  { value: 'lecture', label: 'Лекция' },
  { value: 'practice', label: 'Практика' },
  { value: 'laboratory', label: 'Лабораторная' },
  { value: 'test', label: 'Контрольная' }
];

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

const formatLessonDate = (value) => {
  if (!value) return 'Без даты';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'Без даты';
  return date.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' });
};

const formatDateForAPI = (value) => (
  value ? new Date(`${value}T12:00:00`).toISOString() : undefined
);

const getLessonTypeLabel = (value) => (
  LESSON_TYPES.find((type) => type.value === value)?.label || 'Занятие'
);

const ATTENDANCE_STATUS_OPTIONS = [
  { value: 'present', code: 'Пусто', label: 'Присутствовал' },
  { value: 'absent', code: 'Н', label: 'Отсутствовал' },
  { value: 'excused', code: 'Б', label: 'Болел' }
];

const isAttendanceItem = (item) => item?.item_type === 'attendance_auto';

const normalizeAttendanceStatus = (status) => {
  if (status === 'late') return 'present';
  if (status === 'absent' || status === 'excused' || status === 'present') return status;
  return '';
};

const getAttendanceCode = (status) => {
  if (status === 'absent') return 'Н';
  if (status === 'excused') return 'Б';
  return '\u00A0';
};

const getAttendanceLabel = (status) => (
  ATTENDANCE_STATUS_OPTIONS.find((option) => option.value === status)?.label || 'Статус не выставлен'
);

const getGradeTone = (score, maxScore, hasGrade) => {
  if (!hasGrade) return 'is-empty';
  const percent = maxScore > 0 ? (Number(score) / Number(maxScore)) * 100 : 0;
  if (percent >= 80) return 'is-high';
  if (percent >= 60) return 'is-mid';
  return 'is-low';
};

const getQuickScores = (maxScore) => {
  const max = Math.max(1, asNumber(maxScore));
  if (max === 5) return [2, 3, 4, 5];
  return Array.from(new Set([0, Math.round(max * 0.6), Math.round(max * 0.8), max]));
};

const getGradePoint = (sheets, studentId, itemId) => (
  (sheets[studentId]?.grades || []).find((grade) => Number(grade.item_id) === Number(itemId))
);

const hasRecordedGrade = (grade) => Boolean(grade?.grade_id || grade?.graded_at);

const getAttendanceStatus = (grade, item) => {
  const status = normalizeAttendanceStatus(grade?.attendance_status);
  if (status) return status;
  if (!hasRecordedGrade(grade)) return '';
  return Number(grade.score) >= Number(item.max_score) ? 'present' : 'absent';
};

const hasRecordedCell = (grade, item) => (
  isAttendanceItem(item) ? Boolean(getAttendanceStatus(grade, item)) : hasRecordedGrade(grade)
);

const getCellTone = (grade, item) => {
  if (!isAttendanceItem(item)) {
    return getGradeTone(grade?.score, item.max_score, hasRecordedGrade(grade));
  }
  const status = getAttendanceStatus(grade, item);
  if (status === 'present') return 'is-high';
  if (status === 'excused') return 'is-mid';
  if (status === 'absent') return 'is-low';
  return 'is-empty';
};

const mapSettledWithConcurrency = async (values, limit, worker) => {
  const results = new Array(values.length);
  let nextIndex = 0;
  const runners = Array.from({ length: Math.min(limit, values.length) }, async () => {
    while (nextIndex < values.length) {
      const index = nextIndex;
      nextIndex += 1;
      try {
        results[index] = { status: 'fulfilled', value: await worker(values[index], index) };
      } catch (reason) {
        results[index] = { status: 'rejected', reason };
      }
    }
  });
  await Promise.all(runners);
  return results;
};

const replaceGradePoint = (sheet, item, nextGrade) => {
  const currentGrades = Array.isArray(sheet?.grades) ? sheet.grades : [];
  const exists = currentGrades.some((grade) => Number(grade.item_id) === Number(item.item_id));
  const merged = {
    item_id: item.item_id,
    title: item.title,
    max_score: item.max_score,
    item_type: item.item_type,
    deadline: item.deadline,
    ...nextGrade
  };

  return {
    ...(sheet || {}),
    grades: exists
      ? currentGrades.map((grade) => (
        Number(grade.item_id) === Number(item.item_id) ? { ...grade, ...merged } : grade
      ))
      : [...currentGrades, merged]
  };
};

const TeacherGradebook = ({ token, subjects, subjectsLoading }) => {
  const [selection, setSelection] = useState({ subjectId: 0, groupId: 0 });
  const [roster, setRoster] = useState([]);
  const [items, setItems] = useState([]);
  const [sheets, setSheets] = useState({});
  const [loading, setLoading] = useState(false);
  const [refreshVersion, setRefreshVersion] = useState(0);
  const [search, setSearch] = useState('');
  const [feedback, setFeedback] = useState({ type: '', text: '' });
  const [showLessonForm, setShowLessonForm] = useState(false);
  const [creatingLesson, setCreatingLesson] = useState(false);
  const [lessonForm, setLessonForm] = useState({
    title: 'Лекция 1',
    date: '',
    maxScore: 5,
    itemType: 'lecture'
  });
  const [editingCell, setEditingCell] = useState(null);
  const [draftScore, setDraftScore] = useState('');
  const [savingCell, setSavingCell] = useState('');

  const selectedSubject = useMemo(
    () => subjects.find((subject) => Number(subject.subject_id) === Number(selection.subjectId)),
    [selection.subjectId, subjects]
  );
  const groups = useMemo(() => groupsOf(selectedSubject), [selectedSubject]);

  useEffect(() => {
    if (!subjects.length) {
      setSelection({ subjectId: 0, groupId: 0 });
      return;
    }

    setSelection((current) => {
      const subject = subjects.find((item) => Number(item.subject_id) === Number(current.subjectId)) || subjects[0];
      const subjectGroups = groupsOf(subject);
      const groupIsValid = subjectGroups.some((group) => Number(group.id) === Number(current.groupId));
      return {
        subjectId: Number(subject.subject_id),
        groupId: groupIsValid ? current.groupId : (subjectGroups[0]?.id || 0)
      };
    });
  }, [subjects]);

  useEffect(() => {
    if (!selection.subjectId || !selection.groupId) {
      setRoster([]);
      setItems([]);
      setSheets({});
      return undefined;
    }

    let active = true;
    setLoading(true);
    setEditingCell(null);
    setFeedback({ type: '', text: '' });

    Promise.all([
      api.getGroupStats(token, selection.groupId, selection.subjectId),
      api.getTeacherGradeItems(token, selection.subjectId)
    ])
      .then(async ([rosterResult, itemResult]) => {
        if (!active) return;
        const nextRoster = Array.isArray(rosterResult?.students) ? rosterResult.students : [];
        const nextItems = (Array.isArray(itemResult?.items) ? itemResult.items : [])
          .slice()
          .sort((left, right) => {
            const leftDate = new Date(left.deadline || left.created_at || 0).getTime();
            const rightDate = new Date(right.deadline || right.created_at || 0).getTime();
            return leftDate - rightDate || Number(left.item_id) - Number(right.item_id);
          });
        setRoster(nextRoster);
        setItems(nextItems);

        const sheetResults = await mapSettledWithConcurrency(
          nextRoster,
          6,
          (student) => api.getTeacherStudentGrades(token, student.student_id, selection.subjectId)
        );
        if (!active) return;

        const nextSheets = {};
        let failedCount = 0;
        sheetResults.forEach((result, index) => {
          const studentId = nextRoster[index]?.student_id;
          if (!studentId) return;
          if (result.status === 'fulfilled') {
            nextSheets[studentId] = result.value;
          } else {
            failedCount += 1;
            nextSheets[studentId] = { student_id: studentId, grades: [] };
          }
        });
        setSheets(nextSheets);
        if (failedCount > 0) {
          setFeedback({
            type: 'error',
            text: `Не удалось загрузить ${failedCount} ${failedCount === 1 ? 'строку' : 'строки'} ведомости. Обновите журнал.`
          });
        }
      })
      .catch((err) => {
        if (!active) return;
        setRoster([]);
        setItems([]);
        setSheets({});
        setFeedback({ type: 'error', text: api.getErrorMessage(err, 'Не удалось загрузить журнал') });
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [refreshVersion, selection.groupId, selection.subjectId, token]);

  const visibleRoster = useMemo(() => {
    const query = search.trim().toLocaleLowerCase('ru-RU');
    if (!query) return roster;
    return roster.filter((student) => (
      `${student.student_name || ''} ${student.group_name || ''}`
        .toLocaleLowerCase('ru-RU')
        .includes(query)
    ));
  }, [roster, search]);

  const gradebookStats = useMemo(() => {
    const totalCells = roster.length * items.length;
    let filledCells = 0;
    roster.forEach((student) => {
      items.forEach((item) => {
        if (hasRecordedCell(getGradePoint(sheets, student.student_id, item.item_id), item)) {
          filledCells += 1;
        }
      });
    });
    return {
      filledCells,
      totalCells,
      percent: totalCells ? Math.round((filledCells / totalCells) * 100) : 0
    };
  }, [items, roster, sheets]);

  const changeSubject = (subjectId) => {
    const subject = subjects.find((item) => Number(item.subject_id) === Number(subjectId));
    setSelection({ subjectId, groupId: groupsOf(subject)[0]?.id || 0 });
  };

  const openLessonForm = () => {
    const lectureCount = items.filter((item) => item.item_type === 'lecture').length;
    setLessonForm({
      title: `Лекция ${lectureCount + 1}`,
      date: new Date().toISOString().slice(0, 10),
      maxScore: 5,
      itemType: 'lecture'
    });
    setShowLessonForm(true);
    setFeedback({ type: '', text: '' });
  };

  const changeLessonType = (itemType) => {
    const typeLabel = getLessonTypeLabel(itemType);
    const typeCount = items.filter((item) => item.item_type === itemType).length;
    setLessonForm((current) => ({
      ...current,
      itemType,
      title: `${typeLabel} ${typeCount + 1}`
    }));
  };

  const createLesson = async (event) => {
    event.preventDefault();
    const title = lessonForm.title.trim();
    if (!selection.subjectId || !title) return;

    setCreatingLesson(true);
    setFeedback({ type: '', text: '' });
    try {
      await api.createGradeItem(token, {
        subject_id: selection.subjectId,
        title,
        max_score: Math.max(1, asNumber(lessonForm.maxScore)),
        item_type: lessonForm.itemType,
        deadline: formatDateForAPI(lessonForm.date)
      });
      setShowLessonForm(false);
      setFeedback({ type: 'success', text: `Занятие «${title}» добавлено в журнал.` });
      setRefreshVersion((current) => current + 1);
    } catch (err) {
      setFeedback({ type: 'error', text: api.getErrorMessage(err, 'Не удалось добавить занятие') });
    } finally {
      setCreatingLesson(false);
    }
  };

  const openCellEditor = (student, item) => {
    const grade = getGradePoint(sheets, student.student_id, item.item_id);
    setEditingCell({ studentId: Number(student.student_id), itemId: Number(item.item_id) });
    setDraftScore(isAttendanceItem(item)
      ? (getAttendanceStatus(grade, item) || 'present')
      : (hasRecordedGrade(grade) ? String(grade.score) : ''));
    setFeedback({ type: '', text: '' });
  };

  const saveCell = async (event, student, item) => {
    event.preventDefault();

    if (isAttendanceItem(item)) {
      const status = normalizeAttendanceStatus(draftScore) || 'present';
      const previousGrade = getGradePoint(sheets, student.student_id, item.item_id);
      const sessionId = Number(previousGrade?.attendance_session_id || 0);
      if (!sessionId) {
        setFeedback({ type: 'error', text: 'Для этой отметки не найдено занятие посещаемости.' });
        return;
      }

      const cellKey = `${student.student_id}:${item.item_id}`;
      setSavingCell(cellKey);
      setSheets((current) => ({
        ...current,
        [student.student_id]: replaceGradePoint(current[student.student_id], item, {
          ...previousGrade,
          attendance_status: status,
          attendance_session_id: sessionId,
          graded_at: new Date().toISOString()
        })
      }));

      try {
        await api.updateAttendanceStatus(token, sessionId, Number(student.student_id), status);
        setEditingCell(null);
        setFeedback({ type: 'success', text: `${student.student_name}: ${getAttendanceLabel(status)}.` });
      } catch (err) {
        setSheets((current) => ({
          ...current,
          [student.student_id]: replaceGradePoint(current[student.student_id], item, previousGrade)
        }));
        setFeedback({ type: 'error', text: api.getErrorMessage(err, 'Не удалось сохранить посещаемость') });
      } finally {
        setSavingCell('');
      }
      return;
    }

    const score = Number(draftScore);
    const maxScore = Number(item.max_score || 0);
    if (!Number.isInteger(score) || score < 0 || score > maxScore) {
      setFeedback({ type: 'error', text: `Введите целое число от 0 до ${maxScore}.` });
      return;
    }

    const cellKey = `${student.student_id}:${item.item_id}`;
    const previousGrade = getGradePoint(sheets, student.student_id, item.item_id);
    setSavingCell(cellKey);
    setSheets((current) => ({
      ...current,
      [student.student_id]: replaceGradePoint(current[student.student_id], item, {
        ...previousGrade,
        score,
        graded_at: new Date().toISOString()
      })
    }));

    try {
      const savedGrade = await api.saveStudentGrade(token, {
        student_id: Number(student.student_id),
        item_id: Number(item.item_id),
        score
      });
      setSheets((current) => ({
        ...current,
        [student.student_id]: replaceGradePoint(current[student.student_id], item, {
          score,
          grade_id: savedGrade?.grade_id,
          graded_at: savedGrade?.updated_at || savedGrade?.created_at || new Date().toISOString()
        })
      }));
      setEditingCell(null);
      setFeedback({ type: 'success', text: `${student.student_name}: ${score} из ${maxScore}.` });
    } catch (err) {
      setSheets((current) => ({
        ...current,
        [student.student_id]: replaceGradePoint(current[student.student_id], item, previousGrade || {
          score: 0,
          grade_id: undefined,
          graded_at: undefined
        })
      }));
      setFeedback({ type: 'error', text: api.getErrorMessage(err, 'Не удалось сохранить оценку') });
    } finally {
      setSavingCell('');
    }
  };

  return (
    <div className="teacher-section-card gradebook-page">
      <header className="teacher-section-head gradebook-hero">
        <div>
          <span>Оценки</span>
          <h1>Журнал по занятиям</h1>
          <p>Студенты — по строкам, занятия — по столбцам. Нажмите на ячейку, чтобы сразу поставить или изменить оценку.</p>
        </div>
        <div className="gradebook-hero-summary" aria-label="Заполнение журнала">
          <strong>{gradebookStats.percent}%</strong>
          <span>ячеек заполнено</span>
        </div>
      </header>

      <section className="gradebook-controls" aria-label="Настройки журнала">
        <label>
          <span>Предмет</span>
          <select
            value={selection.subjectId}
            onChange={(event) => changeSubject(asNumber(event.target.value))}
            disabled={subjectsLoading || !subjects.length}
          >
            <option value="">{subjectsLoading ? 'Загрузка...' : 'Выберите предмет'}</option>
            {subjects.map((subject) => (
              <option key={subject.subject_id} value={subject.subject_id}>{subject.subject_name}</option>
            ))}
          </select>
        </label>
        <label>
          <span>Учебная группа</span>
          <select
            value={selection.groupId}
            onChange={(event) => setSelection((current) => ({ ...current, groupId: asNumber(event.target.value) }))}
            disabled={!groups.length}
          >
            <option value="">{groups.length ? 'Выберите группу' : 'Нет групп'}</option>
            {groups.map((group) => <option key={group.id} value={group.id}>{group.name}</option>)}
          </select>
        </label>
        <label className="gradebook-search">
          <span>Найти студента</span>
          <input
            type="search"
            value={search}
            placeholder="Введите фамилию"
            onChange={(event) => setSearch(event.target.value)}
          />
        </label>
        <button
          type="button"
          className="teacher-primary gradebook-add-button"
          onClick={openLessonForm}
          disabled={!selection.subjectId}
        >
          <span aria-hidden="true">+</span> Добавить занятие
        </button>
      </section>

      {showLessonForm && (
        <form className="gradebook-lesson-form" onSubmit={createLesson}>
          <div className="gradebook-lesson-form-head">
            <div>
              <span>Новый столбец</span>
              <strong>Добавить занятие в журнал</strong>
            </div>
            <button type="button" onClick={() => setShowLessonForm(false)} aria-label="Закрыть форму">×</button>
          </div>
          <label>
            <span>Тип</span>
            <select value={lessonForm.itemType} onChange={(event) => changeLessonType(event.target.value)}>
              {LESSON_TYPES.map((type) => <option key={type.value} value={type.value}>{type.label}</option>)}
            </select>
          </label>
          <label className="is-wide">
            <span>Название занятия</span>
            <input
              value={lessonForm.title}
              onChange={(event) => setLessonForm((current) => ({ ...current, title: event.target.value }))}
              required
            />
          </label>
          <label>
            <span>Дата</span>
            <input
              type="date"
              value={lessonForm.date}
              onChange={(event) => setLessonForm((current) => ({ ...current, date: event.target.value }))}
            />
          </label>
          <label>
            <span>Максимальная оценка</span>
            <input
              type="number"
              min="1"
              max="100"
              value={lessonForm.maxScore}
              onChange={(event) => setLessonForm((current) => ({ ...current, maxScore: asNumber(event.target.value) }))}
              required
            />
          </label>
          <div className="gradebook-lesson-actions">
            <button type="button" className="teacher-secondary" onClick={() => setShowLessonForm(false)}>Отмена</button>
            <button type="submit" className="teacher-primary" disabled={creatingLesson}>
              {creatingLesson ? 'Добавляем...' : 'Добавить в журнал'}
            </button>
          </div>
        </form>
      )}

      {feedback.text && (
        <div className={`gradebook-feedback is-${feedback.type}`} role={feedback.type === 'error' ? 'alert' : 'status'}>
          {feedback.text}
        </div>
      )}

      <div className="gradebook-meta">
        <div>
          <strong>{roster.length}</strong> студентов
          <span aria-hidden="true">•</span>
          <strong>{items.length}</strong> занятий
          <span aria-hidden="true">•</span>
          <strong>{gradebookStats.filledCells}</strong> из {gradebookStats.totalCells} оценок
        </div>
        <div className="gradebook-legend" aria-label="Обозначения оценок">
          <span><i className="is-high" />Высокая</span>
          <span><i className="is-mid" />Средняя</span>
          <span><i className="is-low" />Низкая</span>
          <span><i className="is-empty" />Нет оценки</span>
        </div>
      </div>

      <section className="gradebook-shell" aria-label="Таблица оценок" aria-busy={loading}>
        {loading ? (
          <div className="gradebook-loading">
            <span />
            <strong>Собираем журнал группы</strong>
            <small>Загружаем занятия и оценки студентов</small>
          </div>
        ) : !selection.subjectId || !selection.groupId ? (
          <div className="gradebook-empty">
            <strong>Выберите предмет и группу</strong>
            <span>После выбора здесь появится журнал по занятиям.</span>
          </div>
        ) : !items.length ? (
          <div className="gradebook-empty">
            <strong>В журнале пока нет занятий</strong>
            <span>Добавьте первое занятие — оно станет новым столбцом для всей группы.</span>
            <button type="button" className="teacher-primary" onClick={openLessonForm}>Добавить занятие</button>
          </div>
        ) : (
          <div className="gradebook-scroll">
            <table className="gradebook-table">
              <caption className="sr-only">
                Оценки студентов группы по занятиям. Нажмите на ячейку, чтобы изменить оценку.
              </caption>
              <thead>
                <tr>
                  <th className="gradebook-student-head" scope="col">
                    <span>Студент</span>
                    <small>{visibleRoster.length} в списке</small>
                  </th>
                  {items.map((item, index) => (
                    <th key={item.item_id} scope="col">
                      <span className="gradebook-lesson-number">{String(index + 1).padStart(2, '0')}</span>
                      <strong title={item.title}>{item.title}</strong>
                      <small>
                        {formatLessonDate(item.deadline || item.created_at)}
                        {isAttendanceItem(item) ? ' · посещаемость' : ` · до ${item.max_score}`}
                      </small>
                    </th>
                  ))}
                  <th className="gradebook-total-head" scope="col">
                    <span>Итого</span>
                    <small>по журналу</small>
                  </th>
                </tr>
              </thead>
              <tbody>
                {visibleRoster.length ? visibleRoster.map((student, studentIndex) => {
                  let scoreTotal = 0;
                  let maxTotal = 0;
                  items.forEach((item) => {
                    if (isAttendanceItem(item)) return;
                    const grade = getGradePoint(sheets, student.student_id, item.item_id);
                    maxTotal += Number(item.max_score || 0);
                    if (hasRecordedGrade(grade)) scoreTotal += Number(grade.score || 0);
                  });
                  const totalPercent = maxTotal ? Math.round((scoreTotal / maxTotal) * 100) : 0;

                  return (
                    <tr key={student.student_id}>
                      <th scope="row" className="gradebook-student-cell">
                        <div className="gradebook-student-inner">
                          <span>{studentIndex + 1}</span>
                          <div>
                            <strong>{student.student_name}</strong>
                            <small>{student.group_name || groups.find((group) => group.id === selection.groupId)?.name || ''}</small>
                          </div>
                        </div>
                      </th>
                      {items.map((item) => {
                        const grade = getGradePoint(sheets, student.student_id, item.item_id);
                        const hasGrade = hasRecordedGrade(grade);
                        const attendanceStatus = isAttendanceItem(item) ? getAttendanceStatus(grade, item) : '';
                        const isEditing = editingCell?.studentId === Number(student.student_id)
                          && editingCell?.itemId === Number(item.item_id);
                        const cellKey = `${student.student_id}:${item.item_id}`;

                        return (
                          <td
                            key={item.item_id}
                            className={`gradebook-grade-cell ${getCellTone(grade, item)} ${isEditing ? 'is-editing' : ''}`}
                          >
                            {isEditing ? (
                              <form
                                className="gradebook-cell-editor"
                                onSubmit={(event) => saveCell(event, student, item)}
                                onKeyDown={(event) => {
                                  if (event.key === 'Escape') setEditingCell(null);
                                }}
                              >
                                <div className="gradebook-editor-title">
                                  <strong>{student.student_name}</strong>
                                  <span>{isAttendanceItem(item) ? `${item.title} · посещаемость` : `${item.title} · максимум ${item.max_score}`}</span>
                                </div>
                                {isAttendanceItem(item) ? (
                                  <div className="gradebook-attendance-options" role="radiogroup" aria-label={`Посещаемость для ${student.student_name}`}>
                                    {ATTENDANCE_STATUS_OPTIONS.map((option) => (
                                      <button
                                        key={option.value}
                                        type="button"
                                        className={draftScore === option.value ? 'is-selected' : ''}
                                        onClick={() => setDraftScore(option.value)}
                                      >
                                        <strong>{option.code}</strong>
                                        <span>{option.label}</span>
                                      </button>
                                    ))}
                                  </div>
                                ) : (
                                  <>
                                    <input
                                      type="number"
                                      min="0"
                                      max={item.max_score}
                                      step="1"
                                      value={draftScore}
                                      onChange={(event) => setDraftScore(event.target.value)}
                                      aria-label={`Оценка для ${student.student_name}`}
                                      autoFocus
                                      required
                                    />
                                    <div className="gradebook-quick-scores">
                                      {getQuickScores(item.max_score).map((score) => (
                                        <button key={score} type="button" onClick={() => setDraftScore(String(score))}>{score}</button>
                                      ))}
                                    </div>
                                  </>
                                )}
                                <div className="gradebook-editor-actions">
                                  <button type="button" onClick={() => setEditingCell(null)}>Отмена</button>
                                  <button type="submit" disabled={savingCell === cellKey}>Сохранить</button>
                                </div>
                              </form>
                            ) : (
                              <button
                                type="button"
                                className="gradebook-cell-button"
                                onClick={() => openCellEditor(student, item)}
                                aria-label={isAttendanceItem(item)
                                  ? `${student.student_name}, ${item.title}: ${getAttendanceLabel(attendanceStatus)}`
                                  : `${student.student_name}, ${item.title}: ${hasGrade ? `${grade.score} из ${item.max_score}` : 'оценки нет'}`}
                              >
                                <strong>{isAttendanceItem(item) ? getAttendanceCode(attendanceStatus) : (hasGrade ? grade.score : '—')}</strong>
                                <small>{isAttendanceItem(item) ? '\u00A0' : (hasGrade ? `из ${item.max_score}` : 'поставить')}</small>
                              </button>
                            )}
                          </td>
                        );
                      })}
                      <td className="gradebook-total-cell">
                        <strong>{scoreTotal}<span>/{maxTotal}</span></strong>
                        <small>{totalPercent}%</small>
                      </td>
                    </tr>
                  );
                }) : (
                  <tr>
                    <td className="gradebook-no-results" colSpan={items.length + 2}>
                      По запросу студентов не найдено.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <p className="gradebook-hint">
        Посещаемость: пусто — присутствовал, Н — отсутствовал, Б — болел. Остальные занятия оцениваются баллами.
      </p>
    </div>
  );
};

export default TeacherGradebook;
