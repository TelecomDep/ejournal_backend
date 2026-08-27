import React, { useEffect, useMemo, useState } from 'react';
import api from '../../services/api';

const SCOPE_TABS = [
  { key: 'faculty', label: 'Факультет' },
  { key: 'stream', label: 'Поток' },
  { key: 'group', label: 'Группа' },
  { key: 'student', label: 'Студент' }
];

const asNumber = (value) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
};

const clamp = (value) => Math.max(0, Math.min(100, asNumber(value) ?? 0));

const formatPercent = (value, digits = 1) => {
  const parsed = asNumber(value);
  return parsed === null ? '—' : `${parsed.toFixed(digits)}%`;
};

const formatCount = (value) => new Intl.NumberFormat('ru-RU').format(Number(value) || 0);

const formatDate = (value) => {
  if (!value) return '';
  const date = new Date(value);
  return Number.isNaN(date.getTime())
    ? String(value)
    : date.toLocaleDateString('ru-RU', { day: '2-digit', month: 'short' });
};

const toneForValue = (value) => {
  const parsed = asNumber(value);
  if (parsed === null) return 'empty';
  if (parsed >= 80) return 'good';
  if (parsed >= 60) return 'watch';
  return 'risk';
};

const riskLabel = {
  stable: 'стабильно',
  watch: 'зона внимания',
  risk: 'риск',
  critical: 'критично'
};

const RiskBadge = ({ value }) => (
  <span className={`staff-analytics-risk is-${value || 'stable'}`}>
    {riskLabel[value] || value || 'нет данных'}
  </span>
);

const StatCard = ({ label, value, helper, accent = 'blue' }) => (
  <article className={`staff-analytics-stat is-${accent}`}>
    <span>{label}</span>
    <strong>{value}</strong>
    <small>{helper}</small>
  </article>
);

const SectionHeader = ({ eyebrow, title, description, action = null }) => (
  <div className="staff-analytics-section-head">
    <div>
      {eyebrow && <span>{eyebrow}</span>}
      <h2>{title}</h2>
      {description && <p>{description}</p>}
    </div>
    {action}
  </div>
);

const EmptyState = ({ children = 'Недостаточно данных для визуализации.' }) => (
  <div className="staff-analytics-empty">{children}</div>
);

const LineChart = ({ points, metric }) => {
  const width = 760;
  const height = 292;
  const plot = { left: 48, right: 18, top: 18, bottom: 42 };
  const visible = (points || [])
    .map((point, index) => ({
      ...point,
      index,
      value: asNumber(metric === 'attendance' ? point.attendance_percent : point.average_rating)
    }))
    .filter((point) => point.value !== null);

  if (!visible.length) return <EmptyState>Недельных значений пока нет.</EmptyState>;

  const chartWidth = width - plot.left - plot.right;
  const chartHeight = height - plot.top - plot.bottom;
  const maxIndex = Math.max((points || []).length - 1, 1);
  const x = (index) => plot.left + (index / maxIndex) * chartWidth;
  const y = (value) => plot.top + ((100 - clamp(value)) / 100) * chartHeight;
  const path = visible.map((point, index) => `${index === 0 ? 'M' : 'L'} ${x(point.index)} ${y(point.value)}`).join(' ');
  const labels = (points || []).filter((_, index) => (
    index === 0 || index === points.length - 1 || index % Math.max(1, Math.ceil(points.length / 5)) === 0
  ));

  return (
    <div className="staff-analytics-chart-wrap">
      <svg className="staff-analytics-line-chart" viewBox={`0 0 ${width} ${height}`} role="img" aria-label="Динамика показателя по неделям">
        {[0, 20, 40, 60, 80, 100].map((tick) => (
          <g key={tick}>
            <line x1={plot.left} x2={width - plot.right} y1={y(tick)} y2={y(tick)} className="staff-analytics-grid-line" />
            <text x={plot.left - 10} y={y(tick) + 4} textAnchor="end" className="staff-analytics-axis-label">{tick}</text>
          </g>
        ))}
        <path d={path} className="staff-analytics-line" />
        {visible.map((point) => (
          <circle key={`${point.week_start}-${point.index}`} cx={x(point.index)} cy={y(point.value)} r="4.5" className="staff-analytics-point">
            <title>{`${formatDate(point.week_start)}: ${formatPercent(point.value)}`}</title>
          </circle>
        ))}
        {labels.map((point) => (
          <text key={`label-${point.week_start}`} x={x(points.indexOf(point))} y={height - 13} textAnchor="middle" className="staff-analytics-axis-label">
            {formatDate(point.week_start)}
          </text>
        ))}
      </svg>
    </div>
  );
};

const WeeklyTrendCard = ({ points }) => {
  const [metric, setMetric] = useState('rating');
  const title = metric === 'rating' ? 'Средний рейтинг по неделям' : 'Средняя посещаемость по неделям';
  const subtitle = metric === 'rating'
    ? 'Исторический snapshot: оценки учитываются по состоянию на конец каждой недели.'
    : 'Доля присутствий и опозданий среди занятий, не отмеченных как уважительные.';

  return (
    <article className="staff-analytics-card staff-analytics-trend-card">
      <SectionHeader
        eyebrow="Динамика"
        title={title}
        description={subtitle}
        action={(
          <div className="staff-analytics-switch" role="group" aria-label="Показатель динамики">
            <button type="button" className={metric === 'rating' ? 'is-active' : ''} onClick={() => setMetric('rating')}>Рейтинг</button>
            <button type="button" className={metric === 'attendance' ? 'is-active' : ''} onClick={() => setMetric('attendance')}>Посещаемость</button>
          </div>
        )}
      />
      <LineChart points={points} metric={metric} />
    </article>
  );
};

const ProgressBar = ({ value, className = '' }) => {
  const parsed = asNumber(value);
  return (
    <div className={`staff-analytics-progress ${className}`}>
      <span className={`is-${toneForValue(parsed)}`} style={{ width: `${clamp(parsed)}%` }} />
    </div>
  );
};

const RankingCard = ({ data, scopeType, onSelectStream, onSelectGroup, onSelectStudent }) => {
  const streamMode = scopeType === 'faculty' && (data.streams || []).length > 1;
  const items = scopeType === 'student'
    ? (data.subjects || []).map((item) => ({
      id: item.id,
      label: item.name,
      rating: item.average_rating,
      attendance: item.average_attendance,
      meta: `${item.students_with_rating || 0} с оценками`
    }))
    : scopeType === 'group'
      ? (data.students || []).map((item) => ({
        id: item.id,
        label: item.label,
        rating: item.overall_rating,
        attendance: item.attendance_percent,
        risk: item.risk,
        meta: item.group_name
      }))
      : (streamMode ? data.streams : data.groups || []).map((item) => ({
        id: item.id,
        label: item.name,
        rating: item.average_rating,
        attendance: item.attendance_percent,
        meta: streamMode
          ? `${item.group_count || 0} групп · ${item.student_count || 0} студ.`
          : `${item.student_count || 0} студ.`
      }));

  const sorted = [...items]
    .sort((left, right) => (asNumber(right.rating) ?? -1) - (asNumber(left.rating) ?? -1))
    .slice(0, 8);
  const title = scopeType === 'student'
    ? 'Рейтинг по предметам'
    : scopeType === 'group'
      ? 'Студенты группы'
      : streamMode ? 'Потоки по рейтингу' : 'Группы по рейтингу';
  const description = scopeType === 'student' ? 'Среднее значение по выбранным предметам.' : 'Сначала лидеры, затем остальные участники с данными.';

  return (
    <article className="staff-analytics-card staff-analytics-ranking-card">
      <SectionHeader eyebrow="Сравнение" title={title} description={description} />
      {sorted.length ? (
        <div className="staff-analytics-ranking-list">
          {sorted.map((item, index) => {
            const row = (
              <div className="staff-analytics-ranking-row" key={item.id}>
                <span className="staff-analytics-rank">{index + 1}</span>
                <div className="staff-analytics-ranking-main">
                  <div className="staff-analytics-ranking-label">
                    <strong>{item.label}</strong>
                    <small>{item.meta}</small>
                  </div>
                  <ProgressBar value={item.rating} />
                </div>
                <strong className="staff-analytics-ranking-value">{formatPercent(item.rating)}</strong>
                {item.risk && <RiskBadge value={item.risk} />}
              </div>
            );
            if (scopeType === 'group') {
              return <button type="button" className="staff-analytics-ranking-button" key={item.id} onClick={() => onSelectStudent(item.id)}>{row}</button>;
            }
            if (streamMode) {
              return <button type="button" className="staff-analytics-ranking-button" key={item.id} onClick={() => onSelectStream(item.id)}>{row}</button>;
            }
            if (scopeType !== 'student') {
              return <button type="button" className="staff-analytics-ranking-button" key={item.id} onClick={() => onSelectGroup(item.id)}>{row}</button>;
            }
            return row;
          })}
        </div>
      ) : <EmptyState>Нет элементов для сравнения.</EmptyState>}
    </article>
  );
};

const ScatterCard = ({ data, scopeType, onSelectGroup, onSelectStudent }) => {
  const width = 620;
  const height = 300;
  const plot = { left: 50, right: 22, top: 20, bottom: 42 };
  const items = scopeType === 'group'
    ? (data.students || []).map((item) => ({
      id: item.id,
      label: item.label,
      rating: item.overall_rating,
      attendance: item.attendance_percent,
      size: 6,
      risk: item.risk
    }))
    : (data.groups || []).map((item) => ({
      id: item.id,
      label: item.name,
      rating: item.average_rating,
      attendance: item.attendance_percent,
      size: Math.max(5, Math.min(13, Math.sqrt(item.student_count || 1) * 2.3)),
      risk: toneForValue(item.average_rating)
    }));
  const points = items.filter((item) => asNumber(item.rating) !== null && asNumber(item.attendance) !== null);
  const x = (value) => plot.left + (clamp(value) / 100) * (width - plot.left - plot.right);
  const y = (value) => plot.top + ((100 - clamp(value)) / 100) * (height - plot.top - plot.bottom);

  return (
    <article className="staff-analytics-card staff-analytics-scatter-card">
      <SectionHeader
        eyebrow="Связь показателей"
        title="Рейтинг × посещаемость"
        description={scopeType === 'group' ? 'Каждая точка — студент группы.' : 'Каждая точка — группа; размер показывает численность.'}
      />
      {points.length ? (
        <div className="staff-analytics-chart-wrap">
          <svg className="staff-analytics-scatter" viewBox={`0 0 ${width} ${height}`} role="img" aria-label="Диаграмма рейтинга и посещаемости">
            {[0, 20, 40, 60, 80, 100].map((tick) => (
              <g key={tick}>
                <line x1={plot.left} x2={width - plot.right} y1={y(tick)} y2={y(tick)} className="staff-analytics-grid-line" />
                <line x1={x(tick)} x2={x(tick)} y1={plot.top} y2={height - plot.bottom} className="staff-analytics-grid-line is-vertical" />
                <text x={x(tick)} y={height - 13} textAnchor="middle" className="staff-analytics-axis-label">{tick}</text>
                <text x={plot.left - 10} y={y(tick) + 4} textAnchor="end" className="staff-analytics-axis-label">{tick}</text>
              </g>
            ))}
            <text x={width - plot.right} y={height - 3} textAnchor="end" className="staff-analytics-axis-title">посещаемость →</text>
            <text x="10" y={plot.top - 5} className="staff-analytics-axis-title">рейтинг ↑</text>
            {points.map((point) => (
              <circle
                key={point.id}
                cx={x(point.attendance)}
                cy={y(point.rating)}
                r={point.size}
                className={`staff-analytics-scatter-point is-${point.risk || 'watch'}`}
                onClick={() => (scopeType === 'group' ? onSelectStudent(point.id) : onSelectGroup(point.id))}
              >
                <title>{`${point.label}: рейтинг ${formatPercent(point.rating)}, посещаемость ${formatPercent(point.attendance)}`}</title>
              </circle>
            ))}
          </svg>
        </div>
      ) : <EmptyState>Нужны одновременно рейтинг и посещаемость.</EmptyState>}
    </article>
  );
};

const DistributionCard = ({ data }) => {
  const max = Math.max(...(data || []).map((item) => Number(item.count) || 0), 1);
  return (
    <article className="staff-analytics-card staff-analytics-distribution-card">
      <SectionHeader eyebrow="Распределение" title="Как распределён рейтинг" description="Доля студентов в каждом диапазоне результата." />
      <div className="staff-analytics-distribution">
        {(data || []).map((item) => (
          <div className="staff-analytics-distribution-row" key={item.key}>
            <span>{item.label}</span>
            <div className="staff-analytics-distribution-track"><i style={{ width: `${((Number(item.count) || 0) / max) * 100}%` }} /></div>
            <strong>{formatCount(item.count)}</strong>
          </div>
        ))}
      </div>
    </article>
  );
};

const AttendanceCard = ({ data }) => {
  const total = ['present', 'late', 'absent', 'excused'].reduce((sum, key) => sum + (Number(data?.[key]) || 0), 0);
  const segments = [
    { key: 'present', label: 'Присутствовали', color: 'present' },
    { key: 'late', label: 'Опоздали', color: 'late' },
    { key: 'absent', label: 'Отсутствовали', color: 'absent' },
    { key: 'excused', label: 'Уважительная причина', color: 'excused' }
  ];
  return (
    <article className="staff-analytics-card staff-analytics-attendance-card">
      <SectionHeader eyebrow="Посещаемость" title="Структура посещений" description="Опоздания входят в присутствия для итогового процента." />
      {total ? (
        <>
          <div className="staff-analytics-attendance-stack">
            {segments.map((item) => (
              <span key={item.key} className={`is-${item.color}`} style={{ width: `${((Number(data?.[item.key]) || 0) / total) * 100}%` }} />
            ))}
          </div>
          <div className="staff-analytics-attendance-legend">
            {segments.map((item) => (
              <span key={item.key}><i className={`is-${item.color}`} /><b>{formatCount(data?.[item.key])}</b>{item.label}</span>
            ))}
          </div>
        </>
      ) : <EmptyState>Отметок посещаемости пока нет.</EmptyState>}
    </article>
  );
};

const HeatmapCard = ({ data }) => {
  const [metric, setMetric] = useState('rating');
  const groups = [...new Map((data || []).map((item) => [item.group_id, { id: item.group_id, name: item.group_name }])).values()].slice(0, 14);
  const subjects = [...new Map((data || []).map((item) => [item.subject_id, { id: item.subject_id, name: item.subject_name }])).values()].slice(0, 12);
  const byKey = new Map((data || []).map((item) => [`${item.group_id}-${item.subject_id}`, item]));

  return (
    <article className="staff-analytics-card staff-analytics-heatmap-card">
      <SectionHeader
        eyebrow="Предметная карта"
        title="Где возникает просадка"
        description={metric === 'rating' ? 'Средний рейтинг по группе и предмету.' : 'Средняя посещаемость по группе и предмету.'}
        action={(
          <div className="staff-analytics-switch" role="group" aria-label="Показатель предметной карты">
            <button type="button" className={metric === 'rating' ? 'is-active' : ''} onClick={() => setMetric('rating')}>Рейтинг</button>
            <button type="button" className={metric === 'attendance' ? 'is-active' : ''} onClick={() => setMetric('attendance')}>Посещаемость</button>
          </div>
        )}
      />
      {groups.length && subjects.length ? (
        <div className="staff-analytics-heatmap-scroll">
          <table className="staff-analytics-heatmap">
            <thead>
              <tr>
                <th>Группа</th>
                {subjects.map((subject) => <th key={subject.id} title={subject.name}>{subject.name}</th>)}
              </tr>
            </thead>
            <tbody>
              {groups.map((group) => (
                <tr key={group.id}>
                  <th>{group.name}</th>
                  {subjects.map((subject) => {
                    const item = byKey.get(`${group.id}-${subject.id}`);
                    const value = metric === 'rating' ? item?.rating : item?.attendance;
                    return <td key={subject.id} className={`is-${toneForValue(value)}`}>{formatPercent(value, 0)}</td>;
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : <EmptyState>Нужны данные по группам и предметам.</EmptyState>}
    </article>
  );
};

const StudentDetailCard = ({ student }) => {
  if (!student) return null;
  return (
    <article className="staff-analytics-card staff-analytics-student-card">
      <SectionHeader eyebrow="Карточка студента" title={student.label} description={`${student.group_name || 'Группа не указана'} · ${student.rank ? `место ${student.rank}` : 'место не рассчитано'}`} action={<RiskBadge value={student.risk} />} />
      <div className="staff-analytics-student-metrics">
        <div><span>Общий рейтинг</span><strong>{formatPercent(student.overall_rating)}</strong><ProgressBar value={student.overall_rating} /></div>
        <div><span>Посещаемость</span><strong>{formatPercent(student.attendance_percent)}</strong><ProgressBar value={student.attendance_percent} /></div>
        <div><span>Перцентиль</span><strong>{formatPercent(student.percentile, 0)}</strong><small>среди доступных студентов</small></div>
      </div>
      <div className="staff-analytics-subject-list">
        {(student.subjects || []).map((subject) => (
          <div className="staff-analytics-subject-row" key={subject.id}>
            <div><strong>{subject.name}</strong><small>{subject.graded_items || 0}/{subject.due_items || 0} работ · {subject.counted_sessions || 0} занятий</small></div>
            <span>{formatPercent(subject.rating)}</span>
            <ProgressBar value={subject.rating} />
            <span>{formatPercent(subject.attendance)}</span>
          </div>
        ))}
      </div>
    </article>
  );
};

const StudentsTable = ({ students, onSelect }) => (
  <article className="staff-analytics-card staff-analytics-students-card">
    <SectionHeader eyebrow="Рейтинг студентов" title="Кому нужно внимание" description="Нажмите на строку, чтобы открыть срез конкретного человека." />
    {students?.length ? (
      <div className="staff-analytics-table-scroll">
        <table className="staff-analytics-students-table">
          <thead><tr><th>Место</th><th>Студент</th><th>Рейтинг</th><th>Посещаемость</th><th>Статус</th></tr></thead>
          <tbody>
            {students.slice(0, 20).map((student) => (
              <tr key={student.id}>
                <td>{student.rank || '—'}</td>
                <td><button type="button" onClick={() => onSelect(student.id)}>{student.label}</button><small>{student.group_name}</small></td>
                <td><strong>{formatPercent(student.overall_rating)}</strong></td>
                <td>{formatPercent(student.attendance_percent)}</td>
                <td><RiskBadge value={student.risk} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    ) : <EmptyState>Студенты с данными не найдены.</EmptyState>}
  </article>
);

const getSemesterItems = (payload) => {
  if (Array.isArray(payload)) return payload;
  return payload?.items || payload?.semesters || [];
};

const semesterIdOf = (item) => item?.semester_id || item?.id;

const StaffAnalyticsPage = ({ token }) => {
  const [semesters, setSemesters] = useState([]);
  const [semesterId, setSemesterId] = useState('');
  const [scopeType, setScopeType] = useState('faculty');
  const [scopeId, setScopeId] = useState('');
  const [subjectId, setSubjectId] = useState('');
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    api.getSemesters(token)
      .then((payload) => {
        if (cancelled) return;
        const items = getSemesterItems(payload);
        setSemesters(items);
        const current = items.find((item) => item.status === 'open') || items[0];
        if (!semesterId && current) setSemesterId(String(semesterIdOf(current)));
      })
      .catch((err) => {
        if (!cancelled) setError(api.getErrorMessage(err, 'Не удалось загрузить список семестров'));
      });
    return () => { cancelled = true; };
  }, [token]);

  const scopeOptions = useMemo(() => {
    if (!data?.options) return [];
    if (scopeType === 'stream') return data.options.streams || [];
    if (scopeType === 'group') return data.options.groups || [];
    if (scopeType === 'student') return data.options.students || [];
    return [];
  }, [data, scopeType]);

  useEffect(() => {
    if (scopeType === 'faculty') {
      if (scopeId) setScopeId('');
      return;
    }
    if (!scopeOptions.some((option) => String(option.id) === String(scopeId))) {
      setScopeId(scopeOptions[0] ? String(scopeOptions[0].id) : '');
    }
  }, [scopeType, scopeId, scopeOptions]);

  useEffect(() => {
    const subjectOptions = data?.options?.subjects || [];
    if (subjectId && !subjectOptions.some((option) => String(option.id) === String(subjectId))) {
      setSubjectId('');
    }
  }, [data, subjectId]);

  useEffect(() => {
    if (!token || (scopeType !== 'faculty' && !scopeId)) return undefined;
    let cancelled = false;
    setLoading(true);
    api.getStaffAnalytics(token, { semesterId, scopeType, scopeId, subjectId })
      .then((payload) => {
        if (!cancelled) {
          setData(payload);
          setError('');
        }
      })
      .catch((err) => {
        if (!cancelled) setError(api.getErrorMessage(err, 'Не удалось загрузить аналитику'));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => { cancelled = true; };
  }, [token, semesterId, scopeType, scopeId, subjectId]);

  const summary = data?.summary || {};
  const riskStudents = Number(summary.at_risk_students) || 0;
  const activeSemester = data?.semester || semesters.find((item) => String(semesterIdOf(item)) === String(semesterId));
  const roleTitle = data?.scope?.label || 'Вся доступная зона';
  const selectionLabel = scopeType === 'faculty'
    ? 'Весь факультет'
    : scopeOptions.find((option) => String(option.id) === String(scopeId))?.label
      || scopeOptions.find((option) => String(option.id) === String(scopeId))?.name
      || roleTitle;

  const selectGroup = (id) => {
    setScopeType('group');
    setScopeId(String(id));
  };

  const selectStudent = (id) => {
    setScopeType('student');
    setScopeId(String(id));
  };

  if (loading && !data) {
    return <section className="staff-analytics-page"><div className="staff-state-card">Загрузка аналитики...</div></section>;
  }

  if (error && !data) {
    return <section className="staff-analytics-page"><div className="staff-state-card is-error">{error}</div></section>;
  }

  return (
    <section className="staff-analytics-page">
      <header className="staff-analytics-hero">
        <div>
          <span>Аналитика успеваемости</span>
          <h1>{selectionLabel}</h1>
          <p>{roleTitle}. Сравнивайте общий рейтинг и посещаемость, находите группы с отклонениями и переходите к предметам конкретного студента.</p>
        </div>
        <div className="staff-analytics-hero-meta">
          <strong>{activeSemester?.name || activeSemester?.title || data?.semester?.title || 'Текущий семестр'}</strong>
          <small>{data?.semester?.starts_at ? `${formatDate(data.semester.starts_at)} — ${formatDate(data.semester.ends_at)}` : 'Данные доступны в рамках текущего семестра'}</small>
        </div>
      </header>

      <section className="staff-analytics-filters">
        <div className="staff-analytics-filter-head">
          <div>
            <span>Срез данных</span>
            <strong>Выберите уровень детализации</strong>
          </div>
          {loading && <small className="staff-analytics-loading-label">Обновляем…</small>}
        </div>
        <div className="staff-analytics-scope-tabs" role="tablist" aria-label="Уровень аналитики">
          {SCOPE_TABS.map((tab) => (
            <button key={tab.key} type="button" className={scopeType === tab.key ? 'is-active' : ''} onClick={() => { setScopeType(tab.key); setScopeId(''); }}>
              {tab.label}
            </button>
          ))}
        </div>
        <div className="staff-analytics-filter-grid">
          <label>
            <span>Семестр</span>
            <select value={semesterId} onChange={(event) => { setSemesterId(event.target.value); setSubjectId(''); }}>
              {!semesters.length && <option value="">Текущий семестр</option>}
              {semesters.map((item) => <option key={semesterIdOf(item)} value={semesterIdOf(item)}>{item.name || item.title}</option>)}
            </select>
          </label>
          {scopeType !== 'faculty' && (
            <label>
              <span>{SCOPE_TABS.find((tab) => tab.key === scopeType)?.label}</span>
              <select value={scopeId} onChange={(event) => setScopeId(event.target.value)}>
                {scopeOptions.map((option) => <option key={option.id} value={option.id}>{option.name || option.label}</option>)}
              </select>
            </label>
          )}
          <label>
            <span>Предмет</span>
            <select value={subjectId} onChange={(event) => setSubjectId(event.target.value)}>
              <option value="">Все предметы</option>
              {(data?.options?.subjects || []).map((subject) => <option key={subject.id} value={subject.id}>{subject.name}</option>)}
            </select>
          </label>
        </div>
      </section>

      {data && (
        <>
          <div className="staff-analytics-stat-grid">
            <StatCard label="Средний рейтинг" value={formatPercent(summary.average_rating)} helper={`${summary.students_with_rating || 0} студентов с оценками`} accent="blue" />
            <StatCard label="Посещаемость" value={formatPercent(summary.attendance_percent)} helper={`${summary.attended_sessions || 0} присутствий из ${summary.counted_sessions || 0}`} accent="green" />
            <StatCard label="Медиана рейтинга" value={formatPercent(summary.median_rating)} helper={`разброс ${formatPercent(summary.rating_spread)}`} accent="violet" />
            <StatCard label="Зона риска" value={formatCount(riskStudents)} helper={`${summary.critical_risk || 0} критических случаев`} accent="red" />
            <StatCard label="Покрытие оценками" value={formatPercent(summary.grade_coverage)} helper={`${summary.graded_items || 0} из ${summary.due_items || 0} работ`} accent="amber" />
            <StatCard label="Студенты" value={formatCount(summary.student_count)} helper={`${summary.students_with_attendance || 0} с отметками посещаемости`} accent="slate" />
          </div>

          <WeeklyTrendCard points={data.weekly} />

          <div className="staff-analytics-two-column">
            <ScatterCard data={data} scopeType={scopeType} onSelectGroup={selectGroup} onSelectStudent={selectStudent} />
            <RankingCard data={data} scopeType={scopeType} onSelectStream={(id) => { setScopeType('stream'); setScopeId(String(id)); }} onSelectGroup={selectGroup} onSelectStudent={selectStudent} />
          </div>

          <div className="staff-analytics-two-column">
            <DistributionCard data={data.distribution} />
            <AttendanceCard data={data.attendance_breakdown} />
          </div>

          <HeatmapCard data={data.heatmap} />

          {data.student && <StudentDetailCard student={data.student} />}
          {scopeType !== 'student' && <StudentsTable students={data.students} onSelect={selectStudent} />}

          {error && <div className="staff-analytics-inline-error">{error}</div>}
        </>
      )}
    </section>
  );
};

export default StaffAnalyticsPage;
