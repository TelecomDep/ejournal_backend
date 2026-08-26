import React, { useMemo, useState } from 'react';

const toPercent = (value) => Math.max(0, Math.min(100, Number(value) || 0));

const mean = (values) => {
  const numeric = values.filter((value) => Number.isFinite(value));
  return numeric.length ? numeric.reduce((sum, value) => sum + value, 0) / numeric.length : 0;
};

const formatPercent = (value) => `${Math.round(toPercent(value))}%`;

const getZone = (value) => {
  const percent = toPercent(value);
  if (percent >= 80) return 'green';
  if (percent >= 60) return 'yellow';
  if (percent >= 40) return 'orange';
  return 'red';
};

const zoneLabels = {
  green: 'высокая зона',
  yellow: 'хорошая зона',
  orange: 'требует внимания',
  red: 'зона риска'
};

const buildRows = (payload, selectedSubject) => {
  const subjects = Array.isArray(payload?.subjects) ? payload.subjects : [];
  const group = Array.isArray(payload?.groups) ? payload.groups[0] : null;
  const visibleSubjects = selectedSubject === 'all'
    ? subjects
    : subjects.filter((subject) => String(subject.subject_id) === selectedSubject);

  const rows = (group?.students || []).map((student) => {
    const metrics = new Map(
      (student.subjects || []).map((subject) => [String(subject.subject_id), subject])
    );
    const performanceValues = visibleSubjects.map((subject) => {
      const metric = metrics.get(String(subject.subject_id));
      return metric ? toPercent(metric.assessment_summary?.performance_percent) : 0;
    });
    const attendanceValues = visibleSubjects.map((subject) => {
      const metric = metrics.get(String(subject.subject_id));
      return metric ? toPercent(metric.attendance_summary?.attendance_percent) : 0;
    });

    return {
      ref: student.student_ref,
      label: student.student_label || student.student_ref,
      consent: Boolean(student.personal_data_consent?.accepted),
      isCurrentUser: Boolean(student.is_current_user),
      metrics,
      overall: mean(performanceValues),
      attendance: mean(attendanceValues)
    };
  });

  rows.sort((left, right) => (
    right.overall - left.overall
    || right.attendance - left.attendance
    || left.label.localeCompare(right.label, 'ru')
  ));

  return {
    group,
    subjects,
    visibleSubjects,
    rows: rows.map((row, index) => ({ ...row, rank: index + 1 }))
  };
};

const StudentGroupRating = ({ payload, loading, error }) => {
  const [selectedSubject, setSelectedSubject] = useState('all');
  const rating = useMemo(
    () => buildRows(payload, selectedSubject),
    [payload, selectedSubject]
  );
  const currentStudent = rating.rows.find((student) => student.isCurrentUser);
  const groupAverage = mean(rating.rows.map((student) => student.overall));

  if (loading) return <div className="analytics-empty">Загрузка рейтинга группы...</div>;
  if (error) return <div className="analytics-empty is-error">{error}</div>;
  if (!rating.group || rating.rows.length === 0) {
    return (
      <div className="analytics-empty">
        <strong>Рейтинг группы пока не сформирован</strong>
        <span>Для открытого семестра ещё нет данных по студентам и предметам.</span>
      </div>
    );
  }

  return (
    <div className="group-rating">
      <section className="group-rating-summary">
        <div className="group-rating-title">
          <span>Ваша учебная группа</span>
          <h2>{rating.group.group_name || 'Группа не указана'}</h2>
          <p>{payload?.semester?.title || 'Текущий семестр'}</p>
        </div>
        <article>
          <small>Ваше место</small>
          <strong>{currentStudent ? `${currentStudent.rank} из ${rating.rows.length}` : '—'}</strong>
          <span>
            {currentStudent
              ? `${formatPercent(currentStudent.overall)} · ${zoneLabels[getZone(currentStudent.overall)]}`
              : 'в выбранном рейтинге'}
          </span>
        </article>
        <article>
          <small>Средний результат</small>
          <strong>{formatPercent(groupAverage)}</strong>
          <span>по группе</span>
        </article>
        <article>
          <small>Студентов</small>
          <strong>{rating.rows.length}</strong>
          <span>в рейтинге</span>
        </article>
      </section>

      <section className="group-rating-card">
        <div className="group-rating-toolbar">
          <div>
            <span>Сводная успеваемость</span>
            <h2>Рейтинг студентов</h2>
          </div>
          <label>
            <span>Предмет</span>
            <select value={selectedSubject} onChange={(event) => setSelectedSubject(event.target.value)}>
              <option value="all">Все предметы</option>
              {rating.subjects.map((subject) => (
                <option key={subject.subject_id} value={String(subject.subject_id)}>
                  {subject.short_name || subject.name}
                </option>
              ))}
            </select>
          </label>
        </div>

        <div className="group-rating-legend" aria-label="Зоны рейтинга">
          <span className="is-green">80–100% Высокий</span>
          <span className="is-yellow">60–79% Хороший</span>
          <span className="is-orange">40–59% Требует внимания</span>
          <span className="is-red">0–39% Риск</span>
        </div>

        <div className="group-rating-table-wrap">
          <table className="group-rating-table">
            <thead>
              <tr>
                <th>Место</th>
                <th>Студент</th>
                {rating.visibleSubjects.map((subject) => (
                  <th key={subject.subject_id} title={subject.name}>
                    {subject.short_name || subject.name}
                  </th>
                ))}
                <th>Итог</th>
                <th>Посещаемость</th>
              </tr>
            </thead>
            <tbody>
              {rating.rows.map((student) => {
                const zone = getZone(student.overall);
                return (
                  <tr
                    key={student.ref}
                    className={`is-${zone}${student.isCurrentUser ? ' is-current' : ''}`}
                  >
                    <td>
                      <span className={`group-rating-rank is-${zone}`}>{student.rank}</span>
                    </td>
                    <td>
                      <span className="group-rating-student">
                        <strong>{student.label}</strong>
                        <small>
                          {student.isCurrentUser
                            ? 'Это вы'
                            : student.consent ? 'Согласие получено' : 'Персональные данные скрыты'}
                        </small>
                      </span>
                    </td>
                    {rating.visibleSubjects.map((subject) => {
                      const metric = student.metrics.get(String(subject.subject_id));
                      const value = metric?.assessment_summary?.performance_percent;
                      return (
                        <td key={subject.subject_id}>
                          <span className={`group-rating-score is-${getZone(value)}`}>
                            {formatPercent(value)}
                          </span>
                        </td>
                      );
                    })}
                    <td>
                      <span className={`group-rating-score is-${zone}`}>
                        {formatPercent(student.overall)}
                      </span>
                    </td>
                    <td>{formatPercent(student.attendance)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
};

export default StudentGroupRating;
