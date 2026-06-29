import React, { useEffect, useMemo, useState } from 'react';
import api from '../services/api';
import './StaffDashboard.css';

const ROLE_LABELS = {
  admin: 'Администратор',
  dean: 'Декан',
  head: 'Зав. кафедрой',
  teacher: 'Преподаватель'
};

const clampPct = (value) => Math.max(0, Math.min(100, Math.round(Number(value) || 0)));

const pctTone = (pct) => {
  if (pct >= 80) return 'good';
  if (pct >= 60) return 'warn';
  return 'bad';
};

// Concentric attendance ring (pure SVG, no deps — matches RadarChart pattern).
const DonutRing = ({ percent, label }) => {
  const size = 168;
  const stroke = 16;
  const radius = (size - stroke) / 2;
  const circumference = 2 * Math.PI * radius;
  const value = clampPct(percent);
  const offset = circumference * (1 - value / 100);

  return (
    <div className="ring-wrap">
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} className="ring-svg">
        <circle className="ring-track" cx={size / 2} cy={size / 2} r={radius} strokeWidth={stroke} fill="none" />
        <circle
          className={`ring-value ring-${pctTone(value)}`}
          cx={size / 2}
          cy={size / 2}
          r={radius}
          strokeWidth={stroke}
          fill="none"
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          transform={`rotate(-90 ${size / 2} ${size / 2})`}
        />
      </svg>
      <div className="ring-center">
        <span className="ring-number">{value}%</span>
        <span className="ring-label">{label}</span>
      </div>
    </div>
  );
};

// Horizontal ranked bars (groups by attendance).
const RankedBars = ({ items }) => {
  if (!items.length) {
    return <p className="staff-empty">Нет данных.</p>;
  }
  return (
    <div className="bars">
      {items.map((item) => {
        const pct = clampPct(item.value);
        return (
          <div className="bar-row" key={item.key}>
            <span className="bar-name" title={item.name}>{item.name}</span>
            <div className="bar-track">
              <div className={`bar-fill bar-${pctTone(pct)}`} style={{ width: `${pct}%` }} />
            </div>
            <span className="bar-value">{pct}%</span>
          </div>
        );
      })}
    </div>
  );
};

// Distribution histogram across attendance buckets.
const Histogram = ({ buckets }) => {
  const max = Math.max(1, ...buckets.map((b) => b.count));
  return (
    <div className="histogram">
      {buckets.map((bucket) => (
        <div className="hist-col" key={bucket.label}>
          <div className="hist-bar-wrap">
            <span className="hist-count">{bucket.count}</span>
            <div
              className={`hist-bar bar-${bucket.tone}`}
              style={{ height: `${(bucket.count / max) * 100}%` }}
            />
          </div>
          <span className="hist-label">{bucket.label}</span>
        </div>
      ))}
    </div>
  );
};

const StatCard = ({ icon, value, label, accent }) => (
  <div className={`stat-card stat-${accent}`}>
    <span className="stat-icon">{icon}</span>
    <span className="stat-value">{value}</span>
    <span className="stat-label">{label}</span>
  </div>
);

const TABS = [
  { key: 'overview', label: 'Обзор' },
  { key: 'groups', label: 'Группы' },
  { key: 'teachers', label: 'Преподаватели' },
  { key: 'students', label: 'Студенты' }
];

const StaffDashboard = ({ token }) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState('overview');
  const [query, setQuery] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    api.getStaffOverview(token)
      .then((result) => {
        if (!cancelled) {
          setData(result);
          setError('');
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(api.getErrorMessage(err, 'Не удалось загрузить данные'));
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
  }, [token]);

  const metrics = useMemo(() => {
    const students = data?.students || [];
    const groups = data?.groups || [];
    const overall = students.length
      ? Math.round(students.reduce((sum, s) => sum + clampPct(s.attendance_pct), 0) / students.length)
      : 0;
    const atRisk = students.filter((s) => clampPct(s.attendance_pct) < 60).length;

    const buckets = [
      { label: '0–40', tone: 'bad', count: 0 },
      { label: '40–60', tone: 'bad', count: 0 },
      { label: '60–80', tone: 'warn', count: 0 },
      { label: '80–100', tone: 'good', count: 0 }
    ];
    students.forEach((s) => {
      const pct = clampPct(s.attendance_pct);
      if (pct < 40) buckets[0].count += 1;
      else if (pct < 60) buckets[1].count += 1;
      else if (pct < 80) buckets[2].count += 1;
      else buckets[3].count += 1;
    });

    const topGroups = [...groups]
      .sort((a, b) => clampPct(b.attendance_pct) - clampPct(a.attendance_pct))
      .slice(0, 8)
      .map((g) => ({ key: g.group_id, name: g.group_name || '—', value: g.attendance_pct }));

    return { overall, atRisk, buckets, topGroups };
  }, [data]);

  if (loading) {
    return <div className="staff-loading">Загрузка сводки…</div>;
  }
  if (error) {
    return <div className="staff-error">{error}</div>;
  }
  if (!data) {
    return null;
  }

  const roleLabel = ROLE_LABELS[data.role] || data.role;
  const counts = data.counts || {};
  const filterText = query.trim().toLowerCase();
  const matches = (text) => !filterText || String(text || '').toLowerCase().includes(filterText);

  const filteredStudents = (data.students || []).filter(
    (s) => matches(s.name) || matches(s.group_name) || matches(s.lectern_name)
  );
  const filteredGroups = (data.groups || []).filter(
    (g) => matches(g.group_name) || matches(g.lectern_name)
  );
  const filteredTeachers = (data.teachers || []).filter(
    (t) => matches(t.name) || matches(t.job_title) || matches(t.lectern_name)
  );

  return (
    <div className="staff-dashboard">
      <header className="staff-head">
        <div>
          <span className="staff-role-chip">{roleLabel}</span>
          <h1 className="staff-title">Сводка по охвату</h1>
          <p className="staff-scope">{data.label}</p>
        </div>
        {!data.can_edit && <span className="staff-readonly">Только просмотр</span>}
      </header>

      <div className="staff-tabs">
        {TABS.map((item) => (
          <button
            key={item.key}
            type="button"
            className={`staff-tab ${tab === item.key ? 'active' : ''}`}
            onClick={() => setTab(item.key)}
          >
            {item.label}
          </button>
        ))}
      </div>

      {tab === 'overview' && (
        <div className="staff-bento">
          <div className="bento-card bento-stats">
            <StatCard icon="🏛️" value={counts.lecterns ?? 0} label="Кафедры" accent="indigo" />
            <StatCard icon="👥" value={counts.groups ?? 0} label="Группы" accent="blue" />
            <StatCard icon="🎓" value={counts.teachers ?? 0} label="Преподаватели" accent="violet" />
            <StatCard icon="🧑‍🎓" value={counts.students ?? 0} label="Студенты" accent="cyan" />
          </div>

          <div className="bento-card bento-ring">
            <h3 className="bento-title">Средняя посещаемость</h3>
            <DonutRing percent={metrics.overall} label="по охвату" />
            <p className="bento-foot">
              В зоне риска (&lt;60%): <strong>{metrics.atRisk}</strong>
            </p>
          </div>

          <div className="bento-card bento-hist">
            <h3 className="bento-title">Распределение студентов по посещаемости</h3>
            <Histogram buckets={metrics.buckets} />
          </div>

          <div className="bento-card bento-groups">
            <h3 className="bento-title">Топ групп по посещаемости</h3>
            <RankedBars items={metrics.topGroups} />
          </div>
        </div>
      )}

      {tab !== 'overview' && (
        <div className="staff-search">
          <input
            type="text"
            placeholder="Поиск…"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
          <span className="staff-count">
            {tab === 'groups' && `${filteredGroups.length} групп`}
            {tab === 'teachers' && `${filteredTeachers.length} преподавателей`}
            {tab === 'students' && `${filteredStudents.length} студентов`}
          </span>
        </div>
      )}

      {tab === 'groups' && (
        <div className="staff-table-wrap">
          <table className="staff-table">
            <thead>
              <tr>
                <th>Группа</th>
                <th>Кафедра</th>
                <th>Студентов</th>
                <th>Посещаемость</th>
              </tr>
            </thead>
            <tbody>
              {filteredGroups.map((g) => {
                const pct = clampPct(g.attendance_pct);
                return (
                  <tr key={g.group_id}>
                    <td className="cell-strong">{g.group_name || '—'}</td>
                    <td>{g.lectern_name || '—'}</td>
                    <td>{g.student_count ?? 0}</td>
                    <td>
                      <div className="cell-bar">
                        <div className="cell-bar-track">
                          <div className={`cell-bar-fill bar-${pctTone(pct)}`} style={{ width: `${pct}%` }} />
                        </div>
                        <span>{pct}%</span>
                      </div>
                    </td>
                  </tr>
                );
              })}
              {!filteredGroups.length && (
                <tr><td colSpan={4} className="staff-empty">Ничего не найдено.</td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {tab === 'teachers' && (
        <div className="staff-cards">
          {filteredTeachers.map((t) => (
            <div className="teacher-card" key={t.teacher_id}>
              <div className="teacher-avatar">{(t.name || '?').slice(0, 1)}</div>
              <div className="teacher-meta">
                <strong>{t.name || '—'}</strong>
                <span>{t.job_title || 'Преподаватель'}</span>
                <span className="teacher-lectern">{t.lectern_name || '—'}</span>
              </div>
            </div>
          ))}
          {!filteredTeachers.length && <p className="staff-empty">Ничего не найдено.</p>}
        </div>
      )}

      {tab === 'students' && (
        <div className="staff-table-wrap">
          <table className="staff-table">
            <thead>
              <tr>
                <th>Студент</th>
                <th>Группа</th>
                <th>Кафедра</th>
                <th>Посещаемость</th>
              </tr>
            </thead>
            <tbody>
              {filteredStudents.map((s) => {
                const pct = clampPct(s.attendance_pct);
                return (
                  <tr key={s.student_id}>
                    <td className="cell-strong">{s.name || '—'}</td>
                    <td>{s.group_name || '—'}</td>
                    <td>{s.lectern_name || '—'}</td>
                    <td>
                      <div className="cell-bar">
                        <div className="cell-bar-track">
                          <div className={`cell-bar-fill bar-${pctTone(pct)}`} style={{ width: `${pct}%` }} />
                        </div>
                        <span>{pct}%</span>
                      </div>
                    </td>
                  </tr>
                );
              })}
              {!filteredStudents.length && (
                <tr><td colSpan={4} className="staff-empty">Ничего не найдено.</td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

export default StaffDashboard;
