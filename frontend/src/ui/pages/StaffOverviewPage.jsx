import React, { useEffect, useMemo, useState } from 'react';
import api from '../../services/api';

const TABS = [
  { key: 'overview', label: 'Обзор' },
  { key: 'groups', label: 'Группы' },
  { key: 'teachers', label: 'Преподаватели' },
  { key: 'students', label: 'Студенты' }
];

const ROLE_LABELS = {
  admin: 'Администратор',
  dean: 'Декан',
  head: 'Зав. кафедрой',
  teacher: 'Преподаватель'
};

const clampPct = (value) => Math.max(0, Math.min(100, Math.round(Number(value) || 0)));

const toneByPct = (pct) => {
  if (pct >= 80) return 'good';
  if (pct >= 60) return 'warn';
  return 'bad';
};

const StaffIcon = ({ name }) => {
  const icons = {
    groups: (
      <>
        <path d="M7.5 10a2.7 2.7 0 1 0 0-5.4 2.7 2.7 0 0 0 0 5.4Z" />
        <path d="M16.5 10a2.7 2.7 0 1 0 0-5.4 2.7 2.7 0 0 0 0 5.4Z" />
        <path d="M4.5 19v-1.3c0-2.5 1.3-4.2 3.2-4.2s3.2 1.7 3.2 4.2V19" />
        <path d="M13.1 19v-1.3c0-2.5 1.3-4.2 3.2-4.2s3.2 1.7 3.2 4.2V19" />
      </>
    ),
    teachers: (
      <>
        <path d="M4.5 7.2 12 3.8l7.5 3.4-7.5 3.5-7.5-3.5Z" />
        <path d="M7 9v5.2c1.6 1.5 3.3 2.2 5 2.2s3.4-.7 5-2.2V9" />
      </>
    ),
    students: (
      <>
        <circle cx="12" cy="8" r="3.4" />
        <path d="M5.4 19c.9-3.5 3.1-5.3 6.6-5.3s5.7 1.8 6.6 5.3" />
      </>
    ),
    attendance: (
      <>
        <circle cx="12" cy="12" r="8.5" />
        <path d="m8.6 12.2 2.3 2.3 4.8-5" />
      </>
    )
  };

  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      {icons[name] || icons.attendance}
    </svg>
  );
};

const SummaryCard = ({ icon, label, value, helper }) => (
  <article className="staff-summary-card">
    <span className="staff-summary-icon"><StaffIcon name={icon} /></span>
    <span>
      <small>{label}</small>
      <strong>{value}</strong>
      <em>{helper}</em>
    </span>
  </article>
);

const PercentBar = ({ value }) => {
  const pct = clampPct(value);
  return (
    <div className="staff-percent-cell">
      <div className="staff-percent-track">
        <span className={`is-${toneByPct(pct)}`} style={{ width: `${pct}%` }} />
      </div>
      <strong>{pct}%</strong>
    </div>
  );
};

const StaffOverviewPage = ({ token }) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState('overview');
  const [query, setQuery] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    api.getStaffOverview(token)
      .then((payload) => {
        if (!cancelled) {
          setData(payload);
          setError('');
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(api.getErrorMessage(err, 'Не удалось загрузить сводку'));
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [token]);

  const metrics = useMemo(() => {
    const students = data?.students || [];
    const groups = data?.groups || [];
    const average = students.length
      ? Math.round(students.reduce((sum, item) => sum + clampPct(item.attendance_pct), 0) / students.length)
      : 0;
    const risk = students.filter((item) => clampPct(item.attendance_pct) < 60).length;
    const topGroups = [...groups]
      .sort((a, b) => clampPct(b.attendance_pct) - clampPct(a.attendance_pct))
      .slice(0, 6);
    return { average, risk, topGroups };
  }, [data]);

  if (loading) {
    return <section className="staff-overview-page"><div className="staff-state-card">Загрузка сводки...</div></section>;
  }

  if (error) {
    return <section className="staff-overview-page"><div className="staff-state-card is-error">{error}</div></section>;
  }

  const counts = data?.counts || {};
  const normalizedQuery = query.trim().toLowerCase();
  const matches = (...values) => !normalizedQuery || values.some((value) => String(value || '').toLowerCase().includes(normalizedQuery));
  const groups = (data?.groups || []).filter((item) => matches(item.group_name, item.lectern_name));
  const teachers = (data?.teachers || []).filter((item) => matches(item.name, item.job_title, item.lectern_name));
  const students = (data?.students || []).filter((item) => matches(item.name, item.group_name, item.lectern_name));
  const roleLabel = ROLE_LABELS[data?.role] || data?.role || 'Роль';

  return (
    <section className="staff-overview-page">
      <header className="staff-overview-hero">
        <div>
          <span>{roleLabel}</span>
          <h1>Сводка по охвату</h1>
          <p>{data?.label || 'Данные доступны в рамках вашей роли.'}</p>
        </div>
        {!data?.can_edit && <strong>Только просмотр</strong>}
      </header>

      <nav className="staff-tabs" aria-label="Разделы сводки">
        {TABS.map((item) => (
          <button
            key={item.key}
            type="button"
            className={tab === item.key ? 'is-active' : ''}
            onClick={() => setTab(item.key)}
          >
            {item.label}
          </button>
        ))}
      </nav>

      {tab === 'overview' && (
        <>
          <div className="staff-summary-grid">
            <SummaryCard icon="groups" label="Кафедры" value={counts.lecterns ?? 0} helper="в зоне доступа" />
            <SummaryCard icon="groups" label="Группы" value={counts.groups ?? 0} helper="учебные группы" />
            <SummaryCard icon="teachers" label="Преподаватели" value={counts.teachers ?? 0} helper="в расписании" />
            <SummaryCard icon="students" label="Студенты" value={counts.students ?? 0} helper="под наблюдением" />
          </div>

          <div className="staff-overview-grid">
            <article className="staff-attendance-card">
              <span>Средняя посещаемость</span>
              <strong>{metrics.average}%</strong>
              <p>Студентов в зоне риска: {metrics.risk}</p>
              <div className="staff-big-track">
                <span className={`is-${toneByPct(metrics.average)}`} style={{ width: `${metrics.average}%` }} />
              </div>
            </article>

            <article className="staff-list-card">
              <div className="staff-list-head">
                <span>Топ групп</span>
                <strong>по посещаемости</strong>
              </div>
              {metrics.topGroups.length ? metrics.topGroups.map((group) => (
                <div className="staff-ranked-row" key={group.group_id}>
                  <span>{group.group_name || '—'}</span>
                  <PercentBar value={group.attendance_pct} />
                </div>
              )) : <p className="staff-empty-text">Нет данных по группам.</p>}
            </article>
          </div>
        </>
      )}

      {tab !== 'overview' && (
        <div className="staff-search-row">
          <input
            type="text"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Поиск по сводке"
          />
          <span>
            {tab === 'groups' && `${groups.length} групп`}
            {tab === 'teachers' && `${teachers.length} преподавателей`}
            {tab === 'students' && `${students.length} студентов`}
          </span>
        </div>
      )}

      {tab === 'groups' && (
        <div className="staff-table-card">
          <table>
            <thead>
              <tr><th>Группа</th><th>Кафедра</th><th>Студентов</th><th>Посещаемость</th></tr>
            </thead>
            <tbody>
              {groups.map((group) => (
                <tr key={group.group_id}>
                  <td>{group.group_name || '—'}</td>
                  <td>{group.lectern_name || '—'}</td>
                  <td>{group.student_count ?? 0}</td>
                  <td><PercentBar value={group.attendance_pct} /></td>
                </tr>
              ))}
              {!groups.length && <tr><td colSpan="4">Ничего не найдено.</td></tr>}
            </tbody>
          </table>
        </div>
      )}

      {tab === 'teachers' && (
        <div className="staff-card-grid">
          {teachers.map((teacher) => (
            <article className="staff-person-card" key={teacher.teacher_id}>
              <span>{(teacher.name || '?').slice(0, 1)}</span>
              <div>
                <strong>{teacher.name || '—'}</strong>
                <small>{teacher.job_title || 'Преподаватель'}</small>
                <em>{teacher.lectern_name || '—'}</em>
              </div>
            </article>
          ))}
          {!teachers.length && <div className="staff-state-card">Ничего не найдено.</div>}
        </div>
      )}

      {tab === 'students' && (
        <div className="staff-table-card">
          <table>
            <thead>
              <tr><th>Студент</th><th>Группа</th><th>Кафедра</th><th>Посещаемость</th></tr>
            </thead>
            <tbody>
              {students.map((student) => (
                <tr key={student.student_id}>
                  <td>{student.name || '—'}</td>
                  <td>{student.group_name || '—'}</td>
                  <td>{student.lectern_name || '—'}</td>
                  <td><PercentBar value={student.attendance_pct} /></td>
                </tr>
              ))}
              {!students.length && <tr><td colSpan="4">Ничего не найдено.</td></tr>}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
};

export default StaffOverviewPage;
