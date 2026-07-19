import React, { useEffect, useMemo, useState } from 'react';
import api from '../services/api';
import RadarChart from './RadarChart';
import './StudentGradesPanel.css';

const clampPct = (value) => Math.max(0, Math.min(100, Math.round(Number(value) || 0)));

const pctTone = (pct) => {
  if (pct >= 80) return 'good';
  if (pct >= 60) return 'warn';
  return 'bad';
};

const formatDate = (value) => (value ? new Date(value).toLocaleDateString('ru-RU') : '—');

// Concentric percent ring (pure SVG).
const DonutRing = ({ percent }) => {
  const size = 150;
  const stroke = 14;
  const radius = (size - stroke) / 2;
  const circumference = 2 * Math.PI * radius;
  const value = clampPct(percent);
  const offset = circumference * (1 - value / 100);
  return (
    <div className="sg-ring">
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <circle className="sg-ring-track" cx={size / 2} cy={size / 2} r={radius} strokeWidth={stroke} fill="none" />
        <circle
          className={`sg-ring-value sg-${pctTone(value)}`}
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
      <div className="sg-ring-center">
        <span className="sg-ring-num">{value}%</span>
        <span className="sg-ring-cap">успеваемость</span>
      </div>
    </div>
  );
};

const StatCard = ({ value, label, accent, sub }) => (
  <div className={`sg-stat sg-stat-${accent}`}>
    <span className="sg-stat-value">{value}</span>
    <span className="sg-stat-label">{label}</span>
    {sub && <span className="sg-stat-sub">{sub}</span>}
  </div>
);

const StudentGradesPanel = ({ token }) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [subjectFilter, setSubjectFilter] = useState('all');
  const [whatIfMode, setWhatIfMode] = useState(false);
  const [whatIfGrades, setWhatIfGrades] = useState({});

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    api.getStudentAllGrades(token)
      .then((result) => {
        if (!cancelled) {
          setData(result);
          setError('');
        }
      })
      .catch((err) => {
        if (!cancelled) {
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
  }, [token]);

  const subjects = data?.subjects || [];
  const summary = data?.summary || {};

  const computedSubjects = useMemo(() => {
    return subjects.map(s => {
      let currentScore = Number(s.current_score || 0);
      let maxScore = Number(s.total_max || 0);

      if (whatIfMode) {
        let extra = 0;
        (s.grades || []).forEach(g => {
          const key = `${s.subject_id}-${g.item_id}`;
          if (whatIfGrades[key] !== undefined && whatIfGrades[key] !== '') {
             extra += Number(whatIfGrades[key]);
          }
        });

        currentScore += extra;
      }

      return {
        ...s,
        computed_score: currentScore,
        computed_percent: maxScore > 0 ? clampPct((currentScore / maxScore) * 100) : 0
      };
    });
  }, [subjects, whatIfMode, whatIfGrades]);

  const radarData = useMemo(
    () => computedSubjects.map((s) => ({
      subject_id: s.subject_id,
      subject_name: s.subject_name,
      score: s.computed_score,
      max_score: s.total_max,
      percent: s.computed_percent
    })),
    [computedSubjects]
  );

  const overallPct = useMemo(() => {
    let currentScore = Number(summary.current_score || 0);
    if (whatIfMode) {
      let extraScore = 0;
      Object.keys(whatIfGrades).forEach(key => {
        extraScore += Number(whatIfGrades[key] || 0);
      });
      currentScore += extraScore;
    }
    const max = Number(summary.total_max || 0);
    return max > 0 ? clampPct((currentScore / max) * 100) : 0;
  }, [summary, whatIfMode, whatIfGrades]);

  const subjectBars = useMemo(
    () => [...computedSubjects]
      .sort((a, b) => clampPct(b.computed_percent) - clampPct(a.computed_percent))
      .map((s) => ({ key: s.subject_id, name: s.subject_name || `Предмет ${s.subject_id}`, value: clampPct(s.computed_percent) })),
    [computedSubjects]
  );

  // Flatten all grade items into a single table (with subject column).
  const allRows = useMemo(() => {
    const rows = [];
    computedSubjects.forEach((s) => {
      (s.grades || []).forEach((g) => {
        const key = `${s.subject_id}-${g.item_id}`;
        let currentScore = g.score;
        let isHypothetical = false;

        if (whatIfMode && whatIfGrades[key] !== undefined) {
          currentScore = whatIfGrades[key] !== '' ? Number(whatIfGrades[key]) : 0;
          isHypothetical = true;
        }

        rows.push({
          key,
          subjectId: s.subject_id,
          subjectName: s.subject_name || `Предмет ${s.subject_id}`,
          title: g.title,
          itemType: g.item_type,
          score: currentScore,
          originalScore: g.score,
          maxScore: g.max_score,
          gradedAt: g.graded_at,
          percent: g.max_score > 0 ? clampPct((currentScore / g.max_score) * 100) : 0,
          isHypothetical
        });
      });
    });
    return rows;
  }, [computedSubjects, whatIfMode, whatIfGrades]);

  const visibleRows = subjectFilter === 'all'
    ? allRows
    : allRows.filter((r) => String(r.subjectId) === String(subjectFilter));

  const [showTooltip, setShowTooltip] = useState(false);

  if (loading) {
    return <div className="sg-loading">Загрузка оценок…</div>;
  }
  if (error) {
    return <div className="sg-error">{error}</div>;
  }

  const gradedWorks = Number(summary.graded_works || 0);
  const totalWorks = Number(summary.total_works || 0);

  return (
    <section className="student-grades-v2">
      <header className="sg-head">
        <div>
          <span className="sg-chip">Успеваемость</span>
          <h2 className="sg-title">Мои оценки</h2>
          <p className="sg-sub">Все предметы учебного плана, баллы и прогресс за семестр.</p>
        </div>
        <div className="sg-actions">
          <button 
            className={`sg-btn ${whatIfMode ? 'sg-btn-active' : ''}`}
            onClick={() => {
              setWhatIfMode(!whatIfMode);
              if (whatIfMode) setWhatIfGrades({});
            }}
          >
            {whatIfMode ? 'Сбросить прогноз' : 'Прогноз оценок'}
          </button>
          
          <div 
            className="tooltip-container" 
            onMouseEnter={() => setShowTooltip(true)}
            onMouseLeave={() => setShowTooltip(false)}
          >
            <span className="sg-tooltip-icon">
              ?
            </span>
            {showTooltip && (
              <div className="tooltip-text">
                <strong>Прогноз оценок ("Что-Если")</strong>
                Позволяет вводить гипотетические баллы в поля оценок, чтобы рассчитать ваш будущий средневзвешенный балл успеваемости.
              </div>
            )}
          </div>
        </div>
      </header>

      <div className="sg-bento">
        <div className="sg-card sg-ring-card">
          <DonutRing percent={overallPct} />
        </div>
        <div className="sg-stats">
          <StatCard
            accent="indigo"
            value={`${(Number(summary.current_score || 0) + (whatIfMode ? Object.values(whatIfGrades).reduce((a, b) => a + (Number(b) || 0), 0) : 0))}`}
            label="Набрано баллов"
            sub={`из ${summary.total_max || 0} по плану${whatIfMode ? ' (с учетом прогноза)' : ''}`}
          />
          <StatCard
            accent="violet"
            value={`${summary.passed_max || 0}`}
            label="Прошедшие работы"
            sub="максимум на текущий момент"
          />
          <StatCard
            accent="blue"
            value={`${gradedWorks}/${totalWorks}`}
            label="Оценённые работы"
            sub={totalWorks > 0 ? `${clampPct((gradedWorks / totalWorks) * 100)}% выполнено` : 'нет работ'}
          />
          <StatCard
            accent="cyan"
            value={`${subjects.length}`}
            label="Предметов"
            sub="в учебном плане"
          />
        </div>
      </div>

      <div className="sg-grid-2">
        <div className="sg-card">
          <h3 className="sg-card-title">Диаграмма успеваемости</h3>
          {radarData.length ? (
            <RadarChart data={radarData} />
          ) : (
            <p className="sg-empty">Нет предметов для построения диаграммы.</p>
          )}
        </div>
        <div className="sg-card">
          <h3 className="sg-card-title">Баллы по предметам</h3>
          {subjectBars.length ? (
            <div className="sg-bars">
              {subjectBars.map((item) => (
                <div className="sg-bar-row" key={item.key}>
                  <span className="sg-bar-name" title={item.name}>{item.name}</span>
                  <div className="sg-bar-track">
                    <div className={`sg-bar-fill sg-${pctTone(item.value)}`} style={{ width: `${item.value}%` }} />
                  </div>
                  <span className="sg-bar-val">{item.value}%</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="sg-empty">Нет данных.</p>
          )}
        </div>
      </div>

      <div className="sg-subject-cards">
        {computedSubjects.map((s) => {
          const pct = clampPct(s.computed_percent);
          return (
            <div className="sg-subject-card" key={s.subject_id}>
              <div className="sg-subject-top">
                <strong title={s.subject_name}>{s.subject_name || `Предмет ${s.subject_id}`}</strong>
                <span className={`sg-badge sg-badge-${pctTone(pct)}`}>{pct}%</span>
              </div>
              <div className="sg-subject-track">
                <div className={`sg-subject-fill sg-${pctTone(pct)}`} style={{ width: `${pct}%` }} />
              </div>
              <span className="sg-subject-foot">{s.computed_score} / {s.total_max} баллов</span>
            </div>
          );
        })}
      </div>

      <div className="sg-card sg-table-card">
        <div className="sg-table-head">
          <h3 className="sg-card-title">Все оценки</h3>
          <select
            className="sg-select"
            value={subjectFilter}
            onChange={(event) => setSubjectFilter(event.target.value)}
          >
            <option value="all">Все предметы ({allRows.length})</option>
            {subjects.map((s) => (
              <option key={s.subject_id} value={s.subject_id}>{s.subject_name || `Предмет ${s.subject_id}`}</option>
            ))}
          </select>
        </div>
        <div className="sg-table-wrap">
          <table className="sg-table">
            <thead>
              <tr>
                <th>Работа</th>
                <th>Предмет</th>
                <th>Тип</th>
                <th>Результат</th>
                <th>Статус</th>
                <th>Дата</th>
              </tr>
            </thead>
            <tbody>
              {visibleRows.length ? (
                visibleRows.map((row) => (
                  <tr key={row.key}>
                    <td className="sg-cell-strong">{row.title}</td>
                    <td>{row.subjectName}</td>
                    <td><span className="sg-type">{row.itemType}</span></td>
                    <td>
                      <div className="sg-cell-bar">
                        <div className="sg-cell-track">
                          <div className={`sg-cell-fill sg-${pctTone(row.percent)}`} style={{ width: `${row.percent}%` }} />
                        </div>
                        {whatIfMode && (!row.gradedAt || row.isHypothetical) ? (
                          <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                            <input 
                              type="number" 
                              max={row.maxScore} 
                              min="0"
                              value={whatIfGrades[row.key] !== undefined ? whatIfGrades[row.key] : ''}
                              placeholder={row.originalScore}
                              onChange={(e) => {
                                let val = e.target.value;
                                if (val !== '') {
                                  val = String(Math.max(0, Math.min(Number(val), row.maxScore)));
                                }
                                setWhatIfGrades({...whatIfGrades, [row.key]: val});
                              }}
                              style={{ width: '50px', padding: '2px', border: '1px solid var(--line)', borderRadius: '4px', background: 'var(--surface)', color: 'var(--ink)' }}
                            />
                            <span>/ {row.maxScore}</span>
                          </div>
                        ) : (
                          <span>{row.score} / {row.maxScore}</span>
                        )}
                      </div>
                    </td>
                    <td>
                      {row.gradedAt
                        ? <span className="sg-status sg-status-done">Оценено</span>
                        : row.isHypothetical 
                          ? <span className="sg-status sg-status-pending" style={{background: '#ffeb3b', color: '#000'}}>Прогноз</span>
                          : <span className="sg-status sg-status-pending">Ожидает</span>}
                    </td>
                    <td>{formatDate(row.gradedAt)}</td>
                  </tr>
                ))
              ) : (
                <tr><td colSpan={6} className="sg-empty">Оценок пока нет.</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
};

export default StudentGradesPanel;
