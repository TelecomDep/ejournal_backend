import React, { useEffect, useMemo, useState } from 'react';
import api from '../../services/api';

const STATUS_LABELS = {
  planned: 'Запланирован',
  open: 'Текущий',
  closed: 'Закрыт',
  archived: 'Архивный'
};

const ReportIcon = ({ name }) => {
  const paths = name === 'pdf' ? (
    <>
      <path d="M6 3.5h8l4 4V20H6z" />
      <path d="M14 3.5V8h4" />
      <path d="M9 12h6M9 15.5h4" />
    </>
  ) : (
    <>
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <path d="M8 4v16M4 9h16M4 14h16M13 9v11" />
    </>
  );

  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      {paths}
    </svg>
  );
};

const DownloadIcon = () => (
  <svg viewBox="0 0 24 24" aria-hidden="true">
    <path d="M12 4v11" />
    <path d="m8 11 4 4 4-4" />
    <path d="M5 20h14" />
  </svg>
);

const semesterTitle = (semester) => (
  semester.name || `${semester.academic_year || 'Учебный год'}, ${semester.term_num || '—'} семестр`
);

const formatDate = (value) => {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  return new Intl.DateTimeFormat('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' }).format(date);
};

const ReportsPage = ({ token, user }) => {
  const [semesters, setSemesters] = useState([]);
  const [selectedSemesterId, setSelectedSemesterId] = useState('');
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [reportOptions, setReportOptions] = useState({ departments: [], subjects: [] });
  const [optionsLoading, setOptionsLoading] = useState(false);
  const [optionsError, setOptionsError] = useState('');
  const [selectedDepartmentId, setSelectedDepartmentId] = useState('');
  const [selectedSubjectId, setSelectedSubjectId] = useState('');
  const [downloading, setDownloading] = useState('');
  const [feedback, setFeedback] = useState(null);

  const isTeacher = user?.role === 'teacher';

  useEffect(() => {
    let cancelled = false;
    setLoading(true);

    api.getSemesters()
      .then((payload) => {
        if (cancelled) return;
        const items = Array.isArray(payload?.items) ? payload.items : [];
        setSemesters(items);
        const current = items.find((semester) => semester.is_current) || items.find((semester) => semester.status === 'open');
        setSelectedSemesterId(current ? String(current.semester_id) : String(items[0]?.semester_id || ''));
        setLoadError('');
      })
      .catch((error) => {
        if (!cancelled) {
          setLoadError(api.getErrorMessage(error, 'Не удалось загрузить список семестров'));
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!selectedSemesterId) {
      setReportOptions({ departments: [], subjects: [] });
      return undefined;
    }

    let cancelled = false;
    setOptionsLoading(true);
    setOptionsError('');

    api.getStaffGeneralRating(token, selectedSemesterId)
      .then((payload) => {
        if (cancelled) return;
        const departments = Array.isArray(payload?.departments) ? payload.departments : [];
        const subjects = Array.isArray(payload?.subjects) ? payload.subjects : [];
        setReportOptions({ departments, subjects });
        setSelectedDepartmentId((current) => (
          departments.some((item) => String(item.department_id) === current)
            ? current
            : String(departments[0]?.department_id || '')
        ));
        setSelectedSubjectId((current) => (
          subjects.some((item) => String(item.subject_id) === current)
            ? current
            : String(subjects[0]?.subject_id || '')
        ));
      })
      .catch((error) => {
        if (!cancelled) {
          setReportOptions({ departments: [], subjects: [] });
          setSelectedDepartmentId('');
          setSelectedSubjectId('');
          setOptionsError(api.getErrorMessage(error, 'Не удалось загрузить параметры Excel-отчёта'));
        }
      })
      .finally(() => {
        if (!cancelled) setOptionsLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [selectedSemesterId, token]);

  const selectedSemester = useMemo(
    () => semesters.find((semester) => String(semester.semester_id) === selectedSemesterId),
    [semesters, selectedSemesterId]
  );

  const availableSubjects = useMemo(() => {
    if (isTeacher || !selectedDepartmentId) return reportOptions.subjects;
    return reportOptions.subjects.filter(
      (subject) => String(subject.department_id) === selectedDepartmentId
    );
  }, [isTeacher, reportOptions.subjects, selectedDepartmentId]);

  useEffect(() => {
    if (isTeacher) return;
    if (!availableSubjects.some((subject) => String(subject.subject_id) === selectedSubjectId)) {
      setSelectedSubjectId('');
    }
  }, [availableSubjects, isTeacher, selectedSubjectId]);

  const excelReady = isTeacher ? Boolean(selectedSubjectId) : Boolean(selectedDepartmentId);

  const downloadReport = async (format) => {
    setDownloading(format);
    setFeedback(null);
    try {
      const { blob, filename } = await api.downloadStaffPerformanceReport(
        token,
        format,
        selectedSemesterId,
        format === 'xlsx'
          ? {
              departmentId: isTeacher ? '' : selectedDepartmentId,
              subjectId: isTeacher ? selectedSubjectId : ''
            }
          : {}
      );
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
      setFeedback({ type: 'success', text: `Отчёт ${format.toUpperCase()} сформирован и загружен.` });
    } catch (error) {
      setFeedback({
        type: 'error',
        text: api.getErrorMessage(error, 'Не удалось сформировать отчёт')
      });
    } finally {
      setDownloading('');
    }
  };

  const reportFormats = [
    {
      key: 'xlsx',
      title: 'Подробный Excel',
      extension: '.xlsx',
      description: isTeacher
        ? 'Ведомость по выбранному предмету с аналитикой и диаграммами.'
        : 'Отчёт по выбранной кафедре со статистикой и диаграммами.',
      meta: 'Формируется подключённым Python-скриптом'
    },
    {
      key: 'pdf',
      title: 'Сводный PDF',
      extension: '.pdf',
      description: 'Готовый документ с цветовой индикацией результатов студентов.',
      meta: 'Подходит для печати и отправки'
    }
  ];

  return (
    <section className="reports-page">
      <header className="reports-heading">
        <div>
          <span>Учебная аналитика</span>
          <h1>Формирование отчётов</h1>
          <p>Сформируйте рейтинговый отчёт по успеваемости и посещаемости студентов.</p>
        </div>
      </header>

      <div className="reports-settings">
        <div className="reports-settings-copy">
          <span>Параметры отчёта</span>
          <h2>Рейтинг успеваемости</h2>
          <p>В отчёт войдут проценты по предметам, посещаемость и итоговый рейтинг.</p>
        </div>

        <div className="reports-filter-grid">
          <label className="reports-semester-field">
            <span>Семестр</span>
            <select
              value={selectedSemesterId}
              onChange={(event) => {
                setSelectedSemesterId(event.target.value);
                setFeedback(null);
              }}
              disabled={loading || !semesters.length || Boolean(downloading)}
            >
              {!semesters.length && <option value="">Нет доступных семестров</option>}
              {semesters.map((semester) => (
                <option key={semester.semester_id} value={semester.semester_id}>
                  {semesterTitle(semester)} — {STATUS_LABELS[semester.status] || semester.status}
                </option>
              ))}
            </select>
          </label>

          {isTeacher ? (
            <label className="reports-semester-field">
              <span>Предмет для Excel</span>
              <select
                value={selectedSubjectId}
                onChange={(event) => setSelectedSubjectId(event.target.value)}
                disabled={optionsLoading || !reportOptions.subjects.length || Boolean(downloading)}
              >
                {!reportOptions.subjects.length && <option value="">Нет доступных предметов</option>}
                {reportOptions.subjects.map((subject) => (
                  <option key={subject.subject_id} value={subject.subject_id}>
                    {subject.name || subject.short_name}
                  </option>
                ))}
              </select>
            </label>
          ) : (
            <label className="reports-semester-field">
              <span>Кафедра для Excel</span>
              <select
                value={selectedDepartmentId}
                onChange={(event) => setSelectedDepartmentId(event.target.value)}
                disabled={optionsLoading || !reportOptions.departments.length || Boolean(downloading)}
              >
                {!reportOptions.departments.length && <option value="">Нет доступных кафедр</option>}
                {reportOptions.departments.map((department) => (
                  <option key={department.department_id} value={department.department_id}>
                    {department.department_name}
                  </option>
                ))}
              </select>
            </label>
          )}
        </div>
      </div>

      {loadError && <div className="reports-feedback is-error" role="alert">{loadError}</div>}
      {optionsError && <div className="reports-feedback is-error" role="alert">{optionsError}</div>}
      {feedback && (
        <div className={`reports-feedback is-${feedback.type}`} role="status" aria-live="polite">
          {feedback.text}
        </div>
      )}

      {selectedSemester && (
        <div className="reports-period">
          <span className={`reports-status is-${selectedSemester.status || 'planned'}`}>
            {STATUS_LABELS[selectedSemester.status] || selectedSemester.status}
          </span>
          <strong>{semesterTitle(selectedSemester)}</strong>
          <span>{formatDate(selectedSemester.starts_at)} – {formatDate(selectedSemester.ends_at)}</span>
        </div>
      )}

      <div className="reports-format-grid">
        {reportFormats.map((format) => (
          <article className="reports-format-card" key={format.key}>
            <div className={`reports-format-icon is-${format.key}`}><ReportIcon name={format.key} /></div>
            <div className="reports-format-copy">
              <div><h2>{format.title}</h2><span>{format.extension}</span></div>
              <p>{format.description}</p>
              <small>{format.meta}</small>
            </div>
            <button
              type="button"
              onClick={() => downloadReport(format.key)}
              disabled={
                loading
                || Boolean(loadError)
                || !selectedSemesterId
                || Boolean(downloading)
                || (format.key === 'xlsx' && (optionsLoading || Boolean(optionsError) || !excelReady))
              }
            >
              <DownloadIcon />
              <span>{downloading === format.key ? 'Формирование...' : `Скачать ${format.title}`}</span>
            </button>
          </article>
        ))}
      </div>
    </section>
  );
};

export default ReportsPage;
