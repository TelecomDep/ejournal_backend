import React, { useEffect, useMemo, useState } from 'react';
import api from '../../services/api';
import StudentGroupRating from '../components/StudentGroupRating';

const clampPercent = (value) => Math.max(0, Math.min(100, Math.round(Number(value) || 0)));

const getArray = (payload) => {
  if (Array.isArray(payload)) return payload;
  if (Array.isArray(payload?.subjects)) return payload.subjects;
  if (Array.isArray(payload?.points)) return payload.points;
  if (Array.isArray(payload?.items)) return payload.items;
  return [];
};

const getSubjectKey = (subject, index) => String(
  subject?.subject_id ?? subject?.id ?? subject?.subject_name ?? subject?.name ?? index
);

const getSubjectName = (subject, index) => (
  subject?.subject_name || subject?.name || `Предмет ${index + 1}`
);

const getGradePercent = (subject) => {
  const directValue = subject?.grade_percent ?? subject?.percent ?? subject?.performance_percent;
  if (directValue !== undefined && directValue !== null) return clampPercent(directValue);

  const score = Number(subject?.current_score ?? subject?.score ?? 0);
  const maximum = Number(subject?.total_max ?? subject?.max_score ?? 0);
  return maximum > 0 ? clampPercent((score / maximum) * 100) : 0;
};

const getAttendancePercent = (subject) => {
  const value = subject?.attendance_percent ?? subject?.attendance_pct;
  return value === undefined || value === null ? null : clampPercent(value);
};

const normalizeSubjects = (radarPayload, gradesPayload) => {
  const radarSubjects = getArray(radarPayload);
  const gradeSubjects = getArray(gradesPayload);
  const subjects = new Map();

  gradeSubjects.forEach((subject, index) => {
    const key = getSubjectKey(subject, index);
    subjects.set(key, {
      key,
      subjectId: subject?.subject_id ?? subject?.id ?? key,
      name: getSubjectName(subject, index),
      gradePercent: getGradePercent(subject),
      attendancePercent: getAttendancePercent(subject)
    });
  });

  radarSubjects.forEach((subject, index) => {
    const key = getSubjectKey(subject, index);
    const current = subjects.get(key);
    subjects.set(key, {
      key,
      subjectId: subject?.subject_id ?? subject?.id ?? current?.subjectId ?? key,
      name: subject?.subject_name || subject?.name || current?.name || `Предмет ${index + 1}`,
      gradePercent: getGradePercent(subject),
      attendancePercent: getAttendancePercent(subject) ?? current?.attendancePercent ?? null
    });
  });

  return Array.from(subjects.values());
};

const polarPoint = (center, radius, index, count) => {
  const angle = -Math.PI / 2 + (Math.PI * 2 * index) / count;
  return {
    x: center + Math.cos(angle) * radius,
    y: center + Math.sin(angle) * radius
  };
};

const pointsToString = (points) => points.map(({ x, y }) => `${x},${y}`).join(' ');

const shortenName = (name, maxLength) => {
  const value = String(name || 'Предмет');
  return value.length > maxLength ? `${value.slice(0, maxLength - 1)}…` : value;
};

const RadarChart = ({ subjects, metric }) => {
  const visibleSubjects = subjects.filter((subject) => subject[metric] !== null);
  const count = visibleSubjects.length;

  if (count < 3) {
    return (
      <div className="analytics-radar-fallback">
        <strong>Для многоугольника нужно минимум 3 предмета</strong>
        <span>Сейчас доступны данные только по {count}.</span>
      </div>
    );
  }

  const size = 620;
  const center = size / 2;
  const radius = 185;
  const labelRadius = 246;
  const levels = [20, 40, 60, 80, 100];
  const compactLabels = count > 8;
  const outerPoints = visibleSubjects.map((_, index) => polarPoint(center, radius, index, count));
  const valuePoints = visibleSubjects.map((subject, index) => (
    polarPoint(center, radius * (subject[metric] / 100), index, count)
  ));

  return (
    <div className="analytics-radar-wrap">
      <svg
        className="analytics-radar"
        viewBox={`0 0 ${size} ${size}`}
        role="img"
        aria-label={metric === 'gradePercent' ? 'Успеваемость по предметам' : 'Посещаемость по предметам'}
      >
        {levels.map((level) => (
          <polygon
            key={level}
            className="analytics-radar-level"
            points={pointsToString(
              visibleSubjects.map((_, index) => polarPoint(center, radius * (level / 100), index, count))
            )}
          />
        ))}

        {outerPoints.map((point, index) => (
          <line
            key={visibleSubjects[index].key}
            className="analytics-radar-axis"
            x1={center}
            y1={center}
            x2={point.x}
            y2={point.y}
          />
        ))}

        <polygon className={`analytics-radar-value is-${metric}`} points={pointsToString(valuePoints)} />

        {valuePoints.map((point, index) => (
          <circle
            key={visibleSubjects[index].key}
            className={`analytics-radar-point is-${metric}`}
            cx={point.x}
            cy={point.y}
            r="5"
          >
            <title>{`${visibleSubjects[index].name}: ${visibleSubjects[index][metric]}%`}</title>
          </circle>
        ))}

        {visibleSubjects.map((subject, index) => {
          const point = polarPoint(center, labelRadius, index, count);
          const horizontalOffset = point.x - center;
          const anchor = Math.abs(horizontalOffset) < 12 ? 'middle' : horizontalOffset > 0 ? 'start' : 'end';
          return (
            <text
              key={subject.key}
              className="analytics-radar-label"
              x={point.x}
              y={point.y}
              textAnchor={anchor}
            >
              <title>{subject.name}</title>
              <tspan x={point.x} dy="0">{shortenName(subject.name, compactLabels ? 11 : 17)}</tspan>
              <tspan className="analytics-radar-percent" x={point.x} dy="20">{subject[metric]}%</tspan>
            </text>
          );
        })}
      </svg>
    </div>
  );
};

const AnalyticsIcon = ({ name }) => {
  const paths = {
    subjects: (
      <>
        <path d="M4.5 9.5 12 6l7.5 3.5L12 13 4.5 9.5Z" />
        <path d="M7.5 11v4c1.4 1.2 2.9 1.8 4.5 1.8s3.1-.6 4.5-1.8v-4" />
      </>
    ),
    average: (
      <>
        <path d="M5 18V11h3v7H5ZM10.5 18V6h3v12h-3ZM16 18v-9h3v9h-3Z" />
      </>
    ),
    best: <path d="M12 4.8 14.1 9l4.7.7-3.4 3.3.8 4.7-4.2-2.2-4.2 2.2.8-4.7-3.4-3.3 4.7-.7L12 4.8Z" />
  };

  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      {paths[name] || paths.subjects}
    </svg>
  );
};

const MetricCard = ({ icon, label, value, helper }) => (
  <article className="analytics-metric-card">
    <span className="analytics-metric-icon"><AnalyticsIcon name={icon} /></span>
    <span className="analytics-metric-copy">
      <small>{label}</small>
      <strong>{value}</strong>
      <span>{helper}</span>
    </span>
  </article>
);

const AnalyticsPage = ({ user, token }) => {
  const [radarPayload, setRadarPayload] = useState(null);
  const [gradesPayload, setGradesPayload] = useState(null);
  const [groupRatingPayload, setGroupRatingPayload] = useState(null);
  const [metric, setMetric] = useState('gradePercent');
  const [loading, setLoading] = useState(false);
  const [groupRatingLoading, setGroupRatingLoading] = useState(false);
  const [error, setError] = useState('');
  const [groupRatingError, setGroupRatingError] = useState('');

  useEffect(() => {
    if (user?.role !== 'student') return undefined;

    let cancelled = false;
    setLoading(true);
    setError('');

    Promise.allSettled([
      api.getStudentPerformanceRadar(token),
      api.getStudentAllGrades(token)
    ]).then(([radarResult, gradesResult]) => {
      if (cancelled) return;

      if (radarResult.status === 'fulfilled') setRadarPayload(radarResult.value || {});
      if (gradesResult.status === 'fulfilled') setGradesPayload(gradesResult.value || {});

      if (radarResult.status === 'rejected' && gradesResult.status === 'rejected') {
        setError(api.getErrorMessage(radarResult.reason, 'Не удалось загрузить аналитику'));
      }
    }).finally(() => {
      if (!cancelled) setLoading(false);
    });

    return () => {
      cancelled = true;
    };
  }, [token, user?.role]);

  useEffect(() => {
    if (user?.role !== 'student' || metric !== 'groupRating' || groupRatingPayload) return undefined;

    let cancelled = false;
    setGroupRatingLoading(true);
    setGroupRatingError('');

    api.getStudentGroupRating(token)
      .then((payload) => {
        if (!cancelled) setGroupRatingPayload(payload || {});
      })
      .catch((requestError) => {
        if (!cancelled) {
          setGroupRatingError(api.getErrorMessage(requestError, 'Не удалось загрузить рейтинг группы'));
        }
      })
      .finally(() => {
        if (!cancelled) setGroupRatingLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [groupRatingPayload, metric, token, user?.role]);

  const subjects = useMemo(
    () => normalizeSubjects(radarPayload, gradesPayload),
    [gradesPayload, radarPayload]
  );
  const isGroupRating = metric === 'groupRating';
  const chartMetric = metric === 'attendancePercent' ? 'attendancePercent' : 'gradePercent';
  const attendanceSubjects = subjects.filter((subject) => subject.attendancePercent !== null);
  const activeSubjects = chartMetric === 'attendancePercent' ? attendanceSubjects : subjects;
  const average = activeSubjects.length
    ? Math.round(activeSubjects.reduce((sum, subject) => sum + subject[chartMetric], 0) / activeSubjects.length)
    : 0;
  const bestSubject = activeSubjects.reduce(
    (best, subject) => (!best || subject[chartMetric] > best[chartMetric] ? subject : best),
    null
  );
  const hasAttendance = attendanceSubjects.length > 0;

  if (user?.role !== 'student') {
    return (
      <section className="analytics-page">
        <h1>Аналитика</h1>
        <div className="analytics-empty">Персональная аналитика доступна студенту.</div>
      </section>
    );
  }

  return (
    <section className="analytics-page">
      <div className="analytics-heading">
        <div>
          <h1>Аналитика</h1>
          <p>Успеваемость, посещаемость и позиция студента внутри своей группы.</p>
        </div>
        <div className="analytics-switch" aria-label="Показатель диаграммы">
          <button
            type="button"
            className={metric === 'gradePercent' ? 'is-active' : ''}
            onClick={() => setMetric('gradePercent')}
          >
            Успеваемость
          </button>
          <button
            type="button"
            className={metric === 'attendancePercent' ? 'is-active' : ''}
            onClick={() => setMetric('attendancePercent')}
          >
            Посещаемость
          </button>
          <button
            type="button"
            className={metric === 'groupRating' ? 'is-active' : ''}
            onClick={() => setMetric('groupRating')}
          >
            Рейтинг группы
          </button>
        </div>
      </div>

      {isGroupRating ? (
        <StudentGroupRating
          payload={groupRatingPayload}
          loading={groupRatingLoading}
          error={groupRatingError}
        />
      ) : (
        <>
          {loading && <div className="analytics-empty">Загрузка аналитики...</div>}
          {error && !loading && <div className="analytics-empty is-error">{error}</div>}

          {!loading && !error && subjects.length === 0 && (
            <div className="analytics-empty">
              <strong>Данных для аналитики пока нет</strong>
              <span>Предметы появятся после формирования учебного плана.</span>
            </div>
          )}

          {!loading && !error && subjects.length > 0 && (
            <>
              <section className="analytics-metrics">
                <MetricCard
                  icon="subjects"
                  label="Предметов"
                  value={activeSubjects.length}
                  helper="на диаграмме"
                />
                <MetricCard
                  icon="average"
                  label="Средний результат"
                  value={activeSubjects.length ? `${average}%` : '—'}
                  helper={chartMetric === 'gradePercent' ? 'по оцененным работам' : 'по посещенным занятиям'}
                />
                <MetricCard
                  icon="best"
                  label="Лучший предмет"
                  value={bestSubject ? `${bestSubject[chartMetric]}%` : '—'}
                  helper={bestSubject?.name || 'данных пока нет'}
                />
              </section>

              {chartMetric === 'attendancePercent' && !hasAttendance ? (
                <section className="analytics-unavailable">
                  <span className="analytics-unavailable-icon"><AnalyticsIcon name="average" /></span>
                  <div>
                    <strong>Нет посещаемости в разрезе предметов</strong>
                    <p>
                      Пока посещаемость доступна в календаре по датам. Проценты отдельно по каждому предмету
                      появятся здесь, когда для них будут доступны данные.
                    </p>
                  </div>
                </section>
              ) : (
                <section className="analytics-content-card">
                  <div className="analytics-chart-panel">
                    <div className="analytics-card-heading">
                      <span>{chartMetric === 'gradePercent' ? 'Результат по оценкам' : 'Посещение занятий'}</span>
                      <h2>{chartMetric === 'gradePercent' ? 'Успеваемость по предметам' : 'Посещаемость по предметам'}</h2>
                    </div>
                    <RadarChart subjects={subjects} metric={chartMetric} />
                  </div>

                  <div className="analytics-subject-panel">
                    <div className="analytics-card-heading">
                      <span>Расшифровка</span>
                      <h2>Все предметы</h2>
                    </div>
                    <div className="analytics-subject-list">
                      {activeSubjects.map((subject) => (
                        <article key={subject.key} className="analytics-subject-row">
                          <span className="analytics-subject-number" aria-hidden="true" />
                          <span className="analytics-subject-name" title={subject.name}>{subject.name}</span>
                          <strong>{subject[chartMetric]}%</strong>
                          <span className="analytics-subject-progress">
                            <span style={{ width: `${subject[chartMetric]}%` }} />
                          </span>
                        </article>
                      ))}
                    </div>
                  </div>
                </section>
              )}
            </>
          )}
        </>
      )}
    </section>
  );
};

export default AnalyticsPage;
