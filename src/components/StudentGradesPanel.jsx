import React, { useEffect, useState } from 'react';
import api from '../services/api';
import RadarChart from './RadarChart';
import './StudentGradesPanel.css';

const formatDateTime = (value) => {
  if (!value) {
    return 'нет оценки';
  }
  return new Date(value).toLocaleString('ru-RU');
};

const StudentGradesPanel = ({ token }) => {
  const [subjectId, setSubjectId] = useState(1);
  const [gradesData, setGradesData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [radarSubjects, setRadarSubjects] = useState([]);
  const [radarLoading, setRadarLoading] = useState(false);
  const [radarError, setRadarError] = useState('');

  const loadRadar = async () => {
    setRadarError('');
    setRadarLoading(true);
    try {
      const response = await api.getStudentPerformanceRadar(token);
      setRadarSubjects(Array.isArray(response?.subjects) ? response.subjects : []);
    } catch (err) {
      setRadarError(api.getErrorMessage(err, 'Не удалось загрузить диаграмму успеваемости'));
    } finally {
      setRadarLoading(false);
    }
  };

  const loadGrades = async (event) => {
    if (event) {
      event.preventDefault();
    }
    setError('');
    setLoading(true);

    try {
      const response = await api.getStudentGrades(token, Number(subjectId));
      setGradesData(response);
    } catch (err) {
      setError(api.getErrorMessage(err, 'Не удалось загрузить оценки'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (token) {
      loadGrades();
      loadRadar();
    }
  }, [token]);

  const summary = gradesData?.summary || {};
  const currentScore = Number(summary.current_score || 0);
  const totalMax = Number(summary.total_max || 0);
  const progressPercent = totalMax > 0 ? Math.min(100, Math.round((currentScore / totalMax) * 100)) : 0;

  return (
    <section className="student-grades">
      <div className="student-grades-header">
        <div>
          <h2>Оценки</h2>
          <p>Текущие баллы по выбранному предмету</p>
        </div>
        <form onSubmit={loadGrades} className="student-grades-filter">
          <label htmlFor="studentSubjectId">
            ID предмета
            <input
              id="studentSubjectId"
              type="number"
              min="1"
              value={subjectId}
              onChange={(event) => setSubjectId(Number(event.target.value))}
            />
          </label>
          <button type="submit" disabled={loading}>
            {loading ? 'Загрузка...' : 'Показать'}
          </button>
        </form>
      </div>

      <div className="student-radar-card">
        <div className="student-radar-head">
          <h3>Диаграмма успеваемости за семестр</h3>
          <p>По всем предметам учебного плана — доля набранных баллов среди прошедших работ.</p>
        </div>
        {radarError && <div className="student-grades-error">{radarError}</div>}
        {radarLoading ? (
          <p className="radar-empty">Загрузка диаграммы…</p>
        ) : (
          <RadarChart data={radarSubjects} />
        )}
      </div>

      {error && <div className="student-grades-error">{error}</div>}

      <div className="student-score-row">
        <div className="student-score-main">
          <span>Набрано</span>
          <strong>{currentScore}</strong>
        </div>
        <div>
          <span>Максимум по плану</span>
          <strong>{totalMax || '0'}</strong>
        </div>
        <div>
          <span>Прошедшие работы</span>
          <strong>{summary.passed_max || 0}</strong>
        </div>
      </div>

      <div className="student-progress">
        <div style={{ width: `${progressPercent}%` }} />
      </div>

      <div className="student-grades-table-wrap">
        <table className="student-grades-table">
          <thead>
            <tr>
              <th>Работа</th>
              <th>Тип</th>
              <th>Баллы</th>
              <th>Максимум</th>
              <th>Дата оценки</th>
            </tr>
          </thead>
          <tbody>
            {gradesData?.grades?.length > 0 ? (
              gradesData.grades.map((grade) => (
                <tr key={grade.item_id}>
                  <td>{grade.title}</td>
                  <td>{grade.item_type}</td>
                  <td>{grade.score}</td>
                  <td>{grade.max_score}</td>
                  <td>{formatDateTime(grade.graded_at)}</td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={5}>Оценок по этому предмету пока нет</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
};

export default StudentGradesPanel;
