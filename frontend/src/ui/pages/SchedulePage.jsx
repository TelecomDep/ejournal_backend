import React, { useEffect, useMemo, useState } from 'react';
import api from '../../services/api';

const DAY_MS = 24 * 60 * 60 * 1000;
const WEEKDAYS_SHORT = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
const WEEKDAYS_FULL = ['Воскресенье', 'Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота'];
const MONTHS = [
  'января',
  'февраля',
  'марта',
  'апреля',
  'мая',
  'июня',
  'июля',
  'августа',
  'сентября',
  'октября',
  'ноября',
  'декабря'
];

const toISODate = (date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const fromISODate = (isoDate) => new Date(`${isoDate}T00:00:00`);

const startOfCurrentWeek = () => {
  const today = new Date();
  const day = today.getDay() || 7;
  const monday = new Date(today);
  monday.setHours(0, 0, 0, 0);
  monday.setDate(today.getDate() - day + 1);
  return monday;
};

const buildAllowedDays = () => {
  const monday = startOfCurrentWeek();
  return Array.from({ length: 14 }, (_, index) => {
    const date = new Date(monday.getTime() + index * DAY_MS);
    return toISODate(date);
  });
};

const buildMonthDays = (selectedDate) => {
  const date = fromISODate(selectedDate);
  const first = new Date(date.getFullYear(), date.getMonth(), 1);
  const startOffset = (first.getDay() || 7) - 1;
  const gridStart = new Date(first.getTime() - startOffset * DAY_MS);

  return Array.from({ length: 42 }, (_, index) => {
    const day = new Date(gridStart.getTime() + index * DAY_MS);
    return {
      iso: toISODate(day),
      number: day.getDate(),
      isCurrentMonth: day.getMonth() === date.getMonth()
    };
  });
};

const formatHeaderDate = (isoDate) => {
  const date = fromISODate(isoDate);
  return `${date.getDate()} ${MONTHS[date.getMonth()]} ${date.getFullYear()}, ${WEEKDAYS_FULL[date.getDay()]}`;
};

const formatMonthTitle = (isoDate) => {
  const date = fromISODate(isoDate);
  const month = MONTHS[date.getMonth()].replace(/я$/, 'ь');
  return `${month[0].toUpperCase()}${month.slice(1)} ${date.getFullYear()}`;
};

const normalizeLessons = (payload) => {
  if (Array.isArray(payload)) {
    return payload;
  }
  if (Array.isArray(payload?.lessons)) {
    return payload.lessons;
  }
  return [];
};

const getLessonType = (lesson) => {
  const type = String(lesson.lesson_type || '').toLowerCase();
  if (type.includes('лаб')) return 'Лабораторная';
  if (type.includes('лек')) return 'Лекция';
  if (type.includes('прак')) return 'Практика';
  if (lesson.lesson_type) return lesson.lesson_type;
  return 'Пара';
};

const SchedulePage = ({ user, token }) => {
  const isTeacher = ['teacher', 'head'].includes(user?.role);
  const canViewSchedule = user?.role === 'student' || isTeacher;
  const allowedDays = useMemo(buildAllowedDays, []);
  const todayISO = toISODate(new Date());
  const [selectedDate, setSelectedDate] = useState(
    allowedDays.includes(todayISO) ? todayISO : allowedDays[0]
  );
  const [activeView, setActiveView] = useState('day');
  const [lessons, setLessons] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const monthDays = useMemo(() => buildMonthDays(selectedDate), [selectedDate]);
  const selectedIndex = allowedDays.indexOf(selectedDate);

  useEffect(() => {
    if (!canViewSchedule) {
      return undefined;
    }

    let cancelled = false;
    setLoading(true);
    setError('');

    const scheduleRequest = isTeacher
      ? api.getTeacherScheduleDay(token, selectedDate)
      : api.getStudentScheduleDay(token, selectedDate);

    scheduleRequest
      .then((payload) => {
        if (!cancelled) {
          setLessons(normalizeLessons(payload));
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setLessons([]);
          setError(api.getErrorMessage(err, 'Не удалось загрузить расписание'));
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
  }, [canViewSchedule, isTeacher, selectedDate, token]);

  const selectDate = (isoDate) => {
    if (allowedDays.includes(isoDate)) {
      setSelectedDate(isoDate);
    }
  };

  const shiftDate = (shift) => {
    const nextDate = allowedDays[selectedIndex + shift];
    if (nextDate) {
      setSelectedDate(nextDate);
    }
  };

  if (!canViewSchedule) {
    return (
      <section className="schedule-page">
        <h1>Расписание</h1>
        <div className="schedule-empty-role">
          Расписание доступно студентам, преподавателям и заведующим кафедрой.
        </div>
      </section>
    );
  }

  return (
    <section className="schedule-page">
      <div className="schedule-toolbar">
        <h1>Расписание</h1>

        <div className="schedule-date-nav">
          <button type="button" onClick={() => shiftDate(-1)} disabled={selectedIndex <= 0}>
            ‹
          </button>
          <span>{formatHeaderDate(selectedDate)}</span>
          <button type="button" onClick={() => shiftDate(1)} disabled={selectedIndex >= allowedDays.length - 1}>
            ›
          </button>
        </div>
      </div>

      <div className="schedule-tabs" aria-label="Вид расписания">
        <button
          type="button"
          className={activeView === 'week' ? 'is-active' : ''}
          onClick={() => setActiveView('week')}
        >
          Неделя
        </button>
        <button
          type="button"
          className={activeView === 'day' ? 'is-active' : ''}
          onClick={() => setActiveView('day')}
        >
          День
        </button>
        <button
          type="button"
          className={activeView === 'list' ? 'is-active' : ''}
          onClick={() => setActiveView('list')}
        >
          Список
        </button>
      </div>

      <div className="schedule-grid">
        <section className="schedule-day-card">
          <div className="schedule-day-strip">
            {allowedDays.slice(selectedIndex < 7 ? 0 : 7, selectedIndex < 7 ? 7 : 14).map((isoDate) => {
              const date = fromISODate(isoDate);
              return (
                <button
                  key={isoDate}
                  type="button"
                  className={isoDate === selectedDate ? 'is-active' : ''}
                  onClick={() => setSelectedDate(isoDate)}
                >
                  <span>{WEEKDAYS_SHORT[(date.getDay() + 6) % 7]}</span>
                  <strong>{date.getDate()}</strong>
                </button>
              );
            })}
          </div>

          {loading && <div className="schedule-message">Загрузка расписания...</div>}
          {error && !loading && <div className="schedule-message schedule-message--error">{error}</div>}

          {!loading && !error && lessons.length === 0 && (
            <div className="schedule-message">
              <strong>Пар нет</strong>
              <span>На выбранный день расписание пустое.</span>
            </div>
          )}

          {!loading && !error && lessons.length > 0 && (
            <div className={`schedule-list schedule-list--${activeView}`}>
              {lessons.map((lesson) => (
                <article
                  className="schedule-row"
                  key={lesson.schedule_id || `${lesson.lesson_num}-${lesson.subject_id}-${lesson.group_id || ''}-${lesson.start_time}`}
                >
                  <time>
                    {lesson.start_time} - {lesson.end_time}
                  </time>
                  <div className="schedule-row-line" />
                  <div className="schedule-row-main">
                    <h2>{lesson.subject_name}</h2>
                    <p>
                      {lesson.room_info ? `ауд. ${lesson.room_info}` : 'ауд. не указана'}
                      {lesson.subgroup ? `, ${lesson.subgroup}` : ''}
                    </p>
                    <span>
                      {isTeacher
                        ? (lesson.group_name ? `Группа ${lesson.group_name}` : 'Группа не указана')
                        : (lesson.teacher_name || 'Преподаватель не указан')}
                    </span>
                  </div>
                  <strong>{getLessonType(lesson)}</strong>
                </article>
              ))}
            </div>
          )}
        </section>

        <aside className="schedule-side-card">
          <div className="schedule-calendar-head">
            <button type="button" onClick={() => shiftDate(-7)} disabled={selectedIndex < 7}>
              ‹
            </button>
            <strong>{formatMonthTitle(selectedDate)}</strong>
            <button type="button" onClick={() => shiftDate(7)} disabled={selectedIndex > 6}>
              ›
            </button>
          </div>

          <div className="schedule-calendar">
            {WEEKDAYS_SHORT.map((weekday) => (
              <span className="schedule-calendar-weekday" key={weekday}>{weekday}</span>
            ))}
            {monthDays.map((day) => {
              const isAllowed = allowedDays.includes(day.iso);
              return (
                <button
                  key={day.iso}
                  type="button"
                  className={`${day.iso === selectedDate ? 'is-active' : ''} ${!day.isCurrentMonth ? 'is-muted' : ''}`}
                  disabled={!isAllowed}
                  onClick={() => selectDate(day.iso)}
                >
                  {day.number}
                </button>
              );
            })}
          </div>

          <div className="schedule-filters">
            <label>
              Фильтры
              <select disabled>
                <option>Все предметы</option>
              </select>
            </label>
            <label>
              <span className="sr-only">Преподаватель</span>
              <select disabled>
                <option>Все преподаватели</option>
              </select>
            </label>
          </div>
        </aside>
      </div>
    </section>
  );
};

export default SchedulePage;
