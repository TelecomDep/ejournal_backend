import React, { useEffect, useMemo, useState } from 'react';
import api from '../../services/api';

const DAY_MS = 24 * 60 * 60 * 1000;
const WEEKDAYS = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
const MONTHS = [
  'Январь',
  'Февраль',
  'Март',
  'Апрель',
  'Май',
  'Июнь',
  'Июль',
  'Август',
  'Сентябрь',
  'Октябрь',
  'Ноябрь',
  'Декабрь'
];

const toISODate = (date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const fromISODate = (isoDate) => new Date(`${isoDate}T00:00:00`);

const formatDate = (isoDate) => {
  if (!isoDate) return '—';
  const date = fromISODate(isoDate);
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'long',
    year: 'numeric'
  });
};

const formatShortDate = (isoDate) => {
  if (!isoDate) return '—';
  const date = fromISODate(isoDate);
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric'
  });
};

const normalizeItems = (payload) => {
  if (Array.isArray(payload)) return payload;
  if (Array.isArray(payload?.items)) return payload.items;
  return [];
};

const getIntensity = (count) => {
  if (count <= 0) return 0;
  if (count === 1) return 1;
  if (count <= 3) return 2;
  return 3;
};

const buildMonthDays = (year, monthIndex, itemsByDate) => {
  const first = new Date(year, monthIndex, 1);
  const offset = (first.getDay() || 7) - 1;
  const gridStart = new Date(first.getTime() - offset * DAY_MS);

  return Array.from({ length: 42 }, (_, index) => {
    const date = new Date(gridStart.getTime() + index * DAY_MS);
    const iso = toISODate(date);
    const count = Number(itemsByDate.get(iso)?.count || 0);
    return {
      iso,
      number: date.getDate(),
      count,
      isCurrentMonth: date.getMonth() === monthIndex
    };
  });
};

const AttendanceIcon = ({ name }) => {
  const icons = {
    total: (
      <>
        <circle cx="12" cy="12" r="7.4" />
        <path d="m8.6 12.3 2.2 2.1 4.8-5" />
      </>
    ),
    days: (
      <>
        <rect x="5" y="5.5" width="14" height="14" rx="2.4" />
        <path d="M8.2 4v4M15.8 4v4M5 10h14" />
      </>
    ),
    best: (
      <>
        <path d="M12 5.2 14 9.3l4.5.7-3.2 3.2.8 4.5-4.1-2.2-4.1 2.2.8-4.5L5.5 10l4.5-.7L12 5.2Z" />
      </>
    )
  };

  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      {icons[name] || icons.total}
    </svg>
  );
};

const AttendanceStat = ({ icon, label, value, helper }) => (
  <article className="attendance-stat-card">
    <div className="attendance-stat-icon">
      <AttendanceIcon name={icon} />
    </div>
    <div>
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{helper}</small>
    </div>
  </article>
);

const AttendancePage = ({ user, token }) => {
  const today = new Date();
  const [year, setYear] = useState(today.getFullYear());
  const [monthIndex, setMonthIndex] = useState(today.getMonth());
  const [selectedDate, setSelectedDate] = useState(toISODate(today));
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (user?.role !== 'student') {
      return undefined;
    }

    let cancelled = false;
    setLoading(true);
    setError('');

    api.getStudentAttendanceHeatmap(token, year)
      .then((payload) => {
        if (!cancelled) {
          setItems(normalizeItems(payload));
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setItems([]);
          setError(api.getErrorMessage(err, 'Не удалось загрузить посещаемость'));
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
  }, [token, user?.role, year]);

  const itemsByDate = useMemo(() => {
    const map = new Map();
    items.forEach((item) => {
      if (item.date) map.set(item.date, item);
    });
    return map;
  }, [items]);

  const monthDays = useMemo(
    () => buildMonthDays(year, monthIndex, itemsByDate),
    [year, monthIndex, itemsByDate]
  );

  const monthItems = useMemo(() => (
    items
      .filter((item) => {
        const date = fromISODate(item.date);
        return date.getFullYear() === year && date.getMonth() === monthIndex;
      })
      .sort((a, b) => b.date.localeCompare(a.date))
  ), [items, monthIndex, year]);

  const totalMarks = useMemo(
    () => items.reduce((sum, item) => sum + Number(item.count || 0), 0),
    [items]
  );
  const activeDays = items.filter((item) => Number(item.count || 0) > 0).length;
  const bestDay = items.reduce(
    (best, item) => (Number(item.count || 0) > Number(best.count || 0) ? item : best),
    { date: '', count: 0 }
  );
  const selectedCount = Number(itemsByDate.get(selectedDate)?.count || 0);

  const changeMonth = (shift) => {
    const next = new Date(year, monthIndex + shift, 1);
    setYear(next.getFullYear());
    setMonthIndex(next.getMonth());
    setSelectedDate(toISODate(next));
  };

  if (user?.role !== 'student') {
    return (
      <section className="attendance-page">
        <h1>Посещаемость</h1>
        <div className="attendance-empty">
          Для преподавателя посещаемость открывается через раздел работы с группой.
        </div>
      </section>
    );
  }

  return (
    <section className="attendance-page">
      <div className="attendance-heading">
        <div>
          <h1>Посещаемость</h1>
          <p>Календарь отметок по QR и история посещений за выбранный год.</p>
        </div>

        <label className="attendance-year-select">
          <span>Год</span>
          <select value={year} onChange={(event) => setYear(Number(event.target.value))}>
            {[today.getFullYear(), today.getFullYear() - 1, today.getFullYear() - 2].map((value) => (
              <option key={value} value={value}>{value}</option>
            ))}
          </select>
        </label>
      </div>

      {loading && <div className="attendance-empty">Загрузка посещаемости...</div>}
      {error && !loading && <div className="attendance-empty attendance-empty--error">{error}</div>}

      {!loading && !error && (
        <>
          <section className="attendance-hero-card">
            <div className="attendance-hero-main">
              <span>Отметки за год</span>
              <strong>{totalMarks}</strong>
              <p>{activeDays} дней с подтвержденной посещаемостью</p>
            </div>

            <div className="attendance-stat-grid">
              <AttendanceStat
                icon="total"
                label="Всего"
                value={totalMarks}
                helper="отметок по QR"
              />
              <AttendanceStat
                icon="days"
                label="Дней"
                value={activeDays}
                helper="с отметками"
              />
              <AttendanceStat
                icon="best"
                label="Лучший"
                value={bestDay.count || 0}
                helper={bestDay.date ? formatShortDate(bestDay.date) : 'пока нет'}
              />
            </div>
          </section>

          <div className="attendance-layout">
            <section className="attendance-calendar-card">
              <div className="attendance-calendar-head">
                <button type="button" onClick={() => changeMonth(-1)}>‹</button>
                <strong>{MONTHS[monthIndex]} {year}</strong>
                <button type="button" onClick={() => changeMonth(1)}>›</button>
              </div>

              <div className="attendance-calendar">
                {WEEKDAYS.map((weekday) => (
                  <span className="attendance-weekday" key={weekday}>{weekday}</span>
                ))}
                {monthDays.map((day) => {
                  const intensity = getIntensity(day.count);
                  return (
                    <button
                      key={day.iso}
                      type="button"
                      className={`is-level-${intensity} ${day.iso === selectedDate ? 'is-selected' : ''} ${!day.isCurrentMonth ? 'is-muted' : ''}`}
                      onClick={() => setSelectedDate(day.iso)}
                    >
                      <span>{day.number}</span>
                      {day.count > 0 && <small>{day.count}</small>}
                    </button>
                  );
                })}
              </div>

              <div className="attendance-selected-day">
                <span>{formatDate(selectedDate)}</span>
                <strong>{selectedCount ? `${selectedCount} отмет.` : 'Отметок нет'}</strong>
              </div>
            </section>

            <section className="attendance-history-card">
              <div className="attendance-history-head">
                <div>
                  <span>История</span>
                  <h2>{MONTHS[monthIndex].toLowerCase()} {year}</h2>
                </div>
                <strong>{monthItems.length} дней</strong>
              </div>

              {monthItems.length === 0 ? (
                <div className="attendance-history-empty">
                  <strong>В этом месяце отметок нет</strong>
                  <span>Когда посещение будет подтверждено по QR, день появится в истории.</span>
                </div>
              ) : (
                <div className="attendance-history-list">
                  {monthItems.map((item) => (
                    <button
                      key={item.date}
                      type="button"
                      className={item.date === selectedDate ? 'is-active' : ''}
                      onClick={() => setSelectedDate(item.date)}
                    >
                      <span className={`attendance-history-dot is-level-${getIntensity(Number(item.count || 0))}`} />
                      <span>{formatDate(item.date)}</span>
                      <strong>{item.count} отмет.</strong>
                    </button>
                  ))}
                </div>
              )}
            </section>
          </div>
        </>
      )}
    </section>
  );
};

export default AttendancePage;
