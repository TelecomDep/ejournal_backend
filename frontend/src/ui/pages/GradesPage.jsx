import React, { useEffect, useMemo, useState } from 'react';
import api from '../../services/api';

const clampPercent = (value) => Math.max(0, Math.min(100, Math.round(Number(value) || 0)));

const formatDate = (value) => {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleDateString('ru-RU');
};

const getGradeTypeLabel = (type) => {
  const normalized = String(type || '').toLowerCase();
  if (normalized.includes('lab')) return 'Лабораторная';
  if (normalized.includes('test')) return 'Контрольная';
  if (normalized.includes('exam')) return 'Экзамен';
  if (normalized.includes('home')) return 'Домашняя';
  if (normalized.includes('practice')) return 'Практика';
  return type || 'Работа';
};

const getPercentTone = (percent) => {
  if (percent >= 85) return 'is-good';
  if (percent >= 65) return 'is-ok';
  if (percent >= 45) return 'is-warn';
  return 'is-bad';
};

const normalizeSubjects = (payload) => {
  if (Array.isArray(payload)) return payload;
  if (Array.isArray(payload?.subjects)) return payload.subjects;
  return [];
};

const getSubjectPercent = (subject) => {
  if (subject.percent !== undefined) return clampPercent(subject.percent);
  if (subject.grade_percent !== undefined) return clampPercent(subject.grade_percent);

  const current = Number(subject.current_score || subject.score || 0);
  const max = Number(subject.total_max || subject.max_score || 0);
  return max > 0 ? clampPercent((current / max) * 100) : 0;
};

const getSubjectScore = (subject) => Number(subject.current_score || subject.score || 0);
const getSubjectMax = (subject) => Number(subject.total_max || subject.max_score || 0);

const getSubjectRows = (subject) => {
  if (Array.isArray(subject?.grades)) return subject.grades;
  if (Array.isArray(subject?.items)) return subject.items;
  if (Array.isArray(subject?.grade_items)) return subject.grade_items;
  return [];
};

const getSummary = (payload, subjects) => {
  const summary = payload?.summary || {};
  const current = Number(summary.current_score || subjects.reduce((sum, item) => sum + getSubjectScore(item), 0));
  const max = Number(summary.total_max || subjects.reduce((sum, item) => sum + getSubjectMax(item), 0));
  const gradedWorks = Number(
    summary.graded_works || subjects.reduce((sum, item) => (
      sum + getSubjectRows(item).filter((grade) => grade.score !== null && grade.score !== undefined).length
    ), 0)
  );
  const totalWorks = Number(
    summary.total_works || subjects.reduce((sum, item) => sum + getSubjectRows(item).length, 0)
  );

  return {
    current,
    max,
    percent: max > 0 ? clampPercent((current / max) * 100) : 0,
    gradedWorks,
    totalWorks
  };
};

const ProgressBar = ({ percent }) => (
  <div className="grades-progress" aria-label={`${percent}%`}>
    <span style={{ width: `${clampPercent(percent)}%` }} />
  </div>
);

const GradeIcon = ({ name }) => {
  const icons = {
    subject: (
      <>
        <path d="M4.5 10.2 12 6.6l7.5 3.6-7.5 3.7-7.5-3.7Z" />
        <path d="M7.3 12.1v3.2c1.4 1.3 3 2 4.7 2s3.3-.7 4.7-2v-3.2" />
      </>
    ),
    score: (
      <>
        <rect x="7" y="5" width="10" height="14" rx="2" />
        <path d="M9.4 9h5.2M9.4 12h5.2M9.4 15h3.1" />
      </>
    ),
    works: (
      <>
        <rect x="6" y="5" width="12" height="14" rx="2.4" />
        <path d="m9.1 12 2 2 4.3-4.5" />
      </>
    )
  };

  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      {icons[name] || icons.subject}
    </svg>
  );
};

const SummaryCard = ({ icon, label, value, helper }) => (
  <article className="grades-summary-card">
    <div className="grades-card-icon">
      <GradeIcon name={icon} />
    </div>
    <div>
      <span>{label}</span>
      <strong>{value}</strong>
      {helper && <small>{helper}</small>}
    </div>
  </article>
);

const GradesPage = ({ user, token }) => {
  const [payload, setPayload] = useState(null);
  const [selectedSubjectId, setSelectedSubjectId] = useState('all');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (user?.role !== 'student') {
      return undefined;
    }

    let cancelled = false;
    setLoading(true);
    setError('');

    api.getStudentAllGrades(token)
      .then((result) => {
        if (!cancelled) {
          setPayload(result || {});
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setPayload(null);
          setError(api.getErrorMessage(err, 'Не удалось загрузить оценки'));
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
  }, [token, user?.role]);

  const subjects = useMemo(() => normalizeSubjects(payload), [payload]);
  const summary = useMemo(() => getSummary(payload, subjects), [payload, subjects]);
  const selectedSubject = useMemo(() => {
    if (selectedSubjectId === 'all') return null;
    return subjects.find((subject) => String(subject.subject_id) === String(selectedSubjectId)) || null;
  }, [selectedSubjectId, subjects]);

  const rows = useMemo(() => {
    if (selectedSubject) {
      return getSubjectRows(selectedSubject).map((grade) => ({
        ...grade,
        subject_name: selectedSubject.subject_name,
        subject_id: selectedSubject.subject_id
      }));
    }

    return subjects.flatMap((subject) => (
      getSubjectRows(subject).map((grade) => ({
        ...grade,
        subject_name: subject.subject_name,
        subject_id: subject.subject_id
      }))
    ));
  }, [selectedSubject, subjects]);

  if (user?.role !== 'student') {
    return (
      <section className="grades-page">
        <h1>Оценки</h1>
        <div className="grades-empty">
          Этот экран пока подключен для студента. Для преподавателя используется раздел ведомости.
        </div>
      </section>
    );
  }

  return (
    <section className="grades-page">
      <div className="grades-heading">
        <div>
          <h1>Оценки</h1>
          <p>Обзор по предметам, контрольные точки и текущий процент успеваемости.</p>
        </div>

        <label className="grades-filter">
          <span>Предмет</span>
          <select value={selectedSubjectId} onChange={(event) => setSelectedSubjectId(event.target.value)}>
            <option value="all">Все предметы</option>
            {subjects.map((subject) => (
              <option key={subject.subject_id} value={subject.subject_id}>
                {subject.subject_name || `Предмет ${subject.subject_id}`}
              </option>
            ))}
          </select>
        </label>
      </div>

      {loading && <div className="grades-empty">Загрузка оценок...</div>}
      {error && !loading && <div className="grades-empty grades-empty--error">{error}</div>}

      {!loading && !error && (
        <>
          <section className="grades-hero-card">
            <div className="grades-hero-main">
              <span>Успеваемость</span>
              <strong>{summary.percent}%</strong>
              <p>{summary.current} из {summary.max} баллов по доступным работам</p>
              <ProgressBar percent={summary.percent} />
            </div>

            <div className="grades-summary-grid">
              <SummaryCard
                icon="subject"
                label="Предметов"
                value={subjects.length}
                helper="в учебном плане"
              />
              <SummaryCard
                icon="score"
                label="Баллы"
                value={`${summary.current}/${summary.max}`}
                helper="текущий итог"
              />
              <SummaryCard
                icon="works"
                label="Работы"
                value={`${summary.gradedWorks}/${summary.totalWorks}`}
                helper="уже оценены"
              />
            </div>
          </section>

          {subjects.length === 0 ? (
            <div className="grades-empty">
              <strong>Оценок пока нет</strong>
              <span>Когда преподаватель добавит контрольные точки, они появятся здесь.</span>
            </div>
          ) : (
            <div className="grades-layout">
              <section className="grades-subjects">
                {subjects.map((subject) => {
                  const percent = getSubjectPercent(subject);
                  const isActive = String(selectedSubjectId) === String(subject.subject_id);
                  return (
                    <button
                      key={subject.subject_id}
                      type="button"
                      className={`grades-subject-card ${isActive ? 'is-active' : ''}`}
                      onClick={() => setSelectedSubjectId(String(subject.subject_id))}
                    >
                      <span className="grades-subject-icon">
                        <GradeIcon name="subject" />
                      </span>
                      <span className="grades-subject-body">
                        <strong>{subject.subject_name || `Предмет ${subject.subject_id}`}</strong>
                        <small>{getSubjectScore(subject)} из {getSubjectMax(subject)} баллов</small>
                        <ProgressBar percent={percent} />
                      </span>
                      <span className={`grades-percent ${getPercentTone(percent)}`}>{percent}%</span>
                    </button>
                  );
                })}
              </section>

              <section className="grades-table-card">
                <div className="grades-table-head">
                  <div>
                    <span>{selectedSubject ? 'Детально по предмету' : 'Все оценки'}</span>
                    <h2>{selectedSubject?.subject_name || 'Контрольные точки'}</h2>
                  </div>
                  {selectedSubject && (
                    <button type="button" onClick={() => setSelectedSubjectId('all')}>
                      Все предметы
                    </button>
                  )}
                </div>

                <div className="grades-table-wrap">
                  <table className="grades-table">
                    <thead>
                      <tr>
                        <th>Дата</th>
                        {!selectedSubject && <th>Предмет</th>}
                        <th>Работа</th>
                        <th>Тип</th>
                        <th>Баллы</th>
                        <th>Оценка</th>
                      </tr>
                    </thead>
                    <tbody>
                      {rows.length > 0 ? rows.map((grade, index) => {
                        const score = Number(grade.score || 0);
                        const maxScore = Number(grade.max_score || 0);
                        const percent = maxScore > 0 ? clampPercent((score / maxScore) * 100) : 0;
                        return (
                          <tr key={`${grade.subject_id}-${grade.item_id || grade.title}-${index}`}>
                            <td>{formatDate(grade.graded_at || grade.deadline)}</td>
                            {!selectedSubject && <td>{grade.subject_name || '—'}</td>}
                            <td>{grade.title || 'Контрольная точка'}</td>
                            <td>{getGradeTypeLabel(grade.item_type)}</td>
                            <td>{score}/{maxScore}</td>
                            <td>
                              <span className={`grades-mark ${getPercentTone(percent)}`}>{percent}%</span>
                            </td>
                          </tr>
                        );
                      }) : (
                        <tr>
                          <td colSpan={selectedSubject ? 5 : 6}>По выбранному предмету оценок пока нет</td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              </section>
            </div>
          )}
        </>
      )}
    </section>
  );
};

export default GradesPage;
