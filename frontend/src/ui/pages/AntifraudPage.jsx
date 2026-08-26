import React, { useCallback, useEffect, useMemo, useState } from 'react';
import api from '../../services/api';

const PAGE_SIZE = 20;

const EMPTY_FILTERS = {
  search: '',
  group_id: '',
  teacher_id: '',
  reason: '',
  date_from: '',
  date_to: ''
};

const reasonLabel = (reason) => {
  if (reason === 'student is too far from lesson location') return 'Вне зоны занятия';
  if (reason === 'device_id already used in this lesson') return 'Повторное устройство';
  return reason || 'Причина не указана';
};

const reasonClass = (reason) => (
  reason === 'student is too far from lesson location' ? 'is-distance' : 'is-device'
);

const formatDateTime = (value) => {
  if (!value) return 'Нет данных';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'Нет данных';
  return new Intl.DateTimeFormat('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date);
};

const formatCoordinates = (lat, lon) => {
  if (lat === null || lat === undefined || lon === null || lon === undefined) return 'Не переданы';
  return `${Number(lat).toFixed(5)}, ${Number(lon).toFixed(5)}`;
};

const AntifraudIcon = ({ name }) => {
  const icons = {
    shield: <><path d="M12 3 5 6v5c0 4.6 2.7 8.1 7 10 4.3-1.9 7-5.4 7-10V6l-7-3Z" /><path d="m9 12 2 2 4-4" /></>,
    search: <><circle cx="10.5" cy="10.5" r="6.5" /><path d="m15.5 15.5 4 4" /></>,
    refresh: (
      <>
        <path d="M19 8a7.5 7.5 0 1 0 .5 7" />
        <path d="M19 3v5h-5" />
      </>
    ),
    list: <><path d="M8 6h12M8 12h12M8 18h12" /><path d="M4 6h.01M4 12h.01M4 18h.01" /></>,
    ranking: <><path d="M5 20v-6h4v6M10 20V8h4v12M15 20V4h4v16" /></>,
    filter: <path d="M4 5h16l-6.5 7.2V19l-3 1v-7.8L4 5Z" />,
    warning: <><path d="m12 3 9 17H3l9-17Z" /><path d="M12 9v4M12 17h.01" /></>
  };
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      {icons[name] || icons.shield}
    </svg>
  );
};

const LoadingRows = ({ columns }) => (
  <>
    {[0, 1, 2, 3].map((item) => (
      <tr key={item} className="antifraud-skeleton-row" aria-hidden="true">
        {Array.from({ length: columns }, (_, index) => (
          <td key={index}><span /></td>
        ))}
      </tr>
    ))}
  </>
);

const AntifraudPage = ({ token, user, client = api }) => {
  const isAdmin = user?.role === 'admin';
  const [activeTab, setActiveTab] = useState('logs');
  const [filters, setFilters] = useState(EMPTY_FILTERS);
  const [appliedFilters, setAppliedFilters] = useState(EMPTY_FILTERS);
  const [logs, setLogs] = useState([]);
  const [ranking, setRanking] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [catalogs, setCatalogs] = useState({ groups: [], teachers: [] });
  const [loading, setLoading] = useState(true);
  const [catalogsLoading, setCatalogsLoading] = useState(isAdmin);
  const [error, setError] = useState('');
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    if (!isAdmin) return undefined;
    let active = true;
    setCatalogsLoading(true);
    client.getAdminCatalogs(token)
      .then((payload) => {
        if (active) setCatalogs(payload || { groups: [], teachers: [] });
      })
      .catch(() => {
        if (active) setError('Журнал доступен, но справочники фильтров загрузить не удалось.');
      })
      .finally(() => {
        if (active) setCatalogsLoading(false);
      });
    return () => { active = false; };
  }, [client, isAdmin, token]);

  const requestFilters = useMemo(() => ({
    ...appliedFilters,
    page,
    page_size: PAGE_SIZE
  }), [appliedFilters, page]);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [logsResult, rankingResult] = await Promise.all([
        client.getAntifraudLogs(token, requestFilters),
        client.getAntifraudTopCheaters(token)
      ]);
      setLogs(Array.isArray(logsResult?.logs) ? logsResult.logs : []);
      setTotal(Number(logsResult?.total) || 0);
      setRanking(Array.isArray(rankingResult) ? rankingResult : []);
    } catch (requestError) {
      setLogs([]);
      setRanking([]);
      setTotal(0);
      setError(client.getErrorMessage(requestError, 'Не удалось загрузить данные антифрода'));
    } finally {
      setLoading(false);
    }
  }, [appliedFilters, client, requestFilters, token]);

  useEffect(() => {
    loadData();
  }, [loadData, reloadKey]);

  const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const filtersActive = Object.values(appliedFilters).some(Boolean);

  const updateFilter = (event) => {
    const { name, value } = event.target;
    setFilters((current) => ({ ...current, [name]: value }));
  };

  const applyFilters = (event) => {
    event.preventDefault();
    setPage(1);
    setAppliedFilters(filters);
  };

  const resetFilters = () => {
    setFilters(EMPTY_FILTERS);
    setAppliedFilters(EMPTY_FILTERS);
    setPage(1);
  };

  const topStudent = ranking[0];

  return (
    <section className="antifraud-page">
      <header className="antifraud-header">
        <div>
          <span>Контроль посещаемости</span>
          <h1>Антифрод</h1>
          <p>
            {isAdmin
              ? 'Все зафиксированные нарушения при самостоятельной отметке посещаемости.'
              : 'Нарушения студентов из ваших учебных групп.'}
          </p>
        </div>
        <button
          type="button"
          className="semester-icon-button ui-refresh-button antifraud-refresh"
          onClick={() => setReloadKey((value) => value + 1)}
          disabled={loading}
          title="Обновить журнал"
          aria-label="Обновить журнал антифрода"
        >
          <AntifraudIcon name="refresh" />
        </button>
      </header>

      <div className="antifraud-summary" aria-label="Сводка антифрода">
        <article>
          <span>Нарушений по выборке</span>
          <strong>{loading ? '...' : total}</strong>
          <small>{filtersActive ? 'С учетом фильтров' : 'За все время'}</small>
        </article>
        <article>
          <span>Студентов с нарушениями</span>
          <strong>{loading ? '...' : ranking.length}</strong>
          <small>До 50 человек в рейтинге</small>
        </article>
        <article className="is-alert">
          <span>Больше всего попыток</span>
          <strong>{loading ? '...' : topStudent?.total_cheat_attempts ?? 0}</strong>
          <small>{topStudent?.student_name || 'Нарушений не найдено'}</small>
        </article>
      </div>

      <div className="antifraud-tabs" role="tablist" aria-label="Разделы антифрода">
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'logs'}
          className={activeTab === 'logs' ? 'is-active' : ''}
          onClick={() => setActiveTab('logs')}
        >
          <AntifraudIcon name="list" />
          Журнал нарушений
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'ranking'}
          className={activeTab === 'ranking' ? 'is-active' : ''}
          onClick={() => setActiveTab('ranking')}
        >
          <AntifraudIcon name="ranking" />
          Топ нарушителей
        </button>
      </div>

      {activeTab === 'logs' && (isAdmin ? (
        <form className="antifraud-filters" onSubmit={applyFilters}>
          <div className="antifraud-filter-title">
            <AntifraudIcon name="filter" />
            <div>
              <strong>Фильтры журнала</strong>
              <span>Отбор записей журнала</span>
            </div>
          </div>

          <label className="admin-filter-field antifraud-search">
            <span>Поиск</span>
            <div className="admin-search-field">
              <AntifraudIcon name="search" />
              <input
                name="search"
                value={filters.search}
                onChange={updateFilter}
                placeholder="Студент, предмет, устройство"
              />
            </div>
          </label>

          <label className="admin-filter-field">
            <span>Группа</span>
            <select name="group_id" value={filters.group_id} onChange={updateFilter} disabled={catalogsLoading}>
              <option value="">Все группы</option>
              {(catalogs.groups || []).map((group) => (
                <option key={group.id} value={group.id}>{group.name}</option>
              ))}
            </select>
          </label>

          <label className="admin-filter-field">
            <span>Преподаватель</span>
            <select name="teacher_id" value={filters.teacher_id} onChange={updateFilter} disabled={catalogsLoading}>
              <option value="">Все преподаватели</option>
              {(catalogs.teachers || []).map((teacher) => (
                <option key={teacher.id} value={teacher.id}>{teacher.name}</option>
              ))}
            </select>
          </label>

          <label className="admin-filter-field">
            <span>Тип нарушения</span>
            <select name="reason" value={filters.reason} onChange={updateFilter}>
              <option value="">Все типы</option>
              <option value="distance">Вне зоны занятия</option>
              <option value="device">Повторное устройство</option>
            </select>
          </label>

          <label className="admin-filter-field">
            <span>Дата с</span>
            <input type="date" name="date_from" value={filters.date_from} onChange={updateFilter} />
          </label>

          <label className="admin-filter-field">
            <span>Дата по</span>
            <input type="date" name="date_to" value={filters.date_to} onChange={updateFilter} />
          </label>

          <div className="antifraud-filter-actions">
            <button type="button" className="admin-secondary-button" onClick={resetFilters} disabled={!filtersActive && !Object.values(filters).some(Boolean)}>
              Сбросить
            </button>
            <button type="submit" className="admin-primary-button">Применить</button>
          </div>
        </form>
      ) : (
        <div className="antifraud-scope-note">
          <AntifraudIcon name="shield" />
          <div>
            <strong>Доступ ограничен вашими группами</strong>
            <span>Данные других преподавателей и групп не отображаются.</span>
          </div>
        </div>
      ))}

      {error && (
        <div className="admin-notice is-error" role="alert">{error}</div>
      )}

      {activeTab === 'logs' ? (
        <>
          <div className={`antifraud-table-wrap ${loading ? 'is-loading' : ''}`}>
            <table className="antifraud-table">
              <thead>
                <tr>
                  <th>Студент</th>
                  <th>Занятие</th>
                  <th>Нарушение</th>
                  <th>Устройство и координаты</th>
                  <th>Время</th>
                </tr>
              </thead>
              <tbody>
                {loading && <LoadingRows columns={5} />}
                {!loading && logs.length === 0 && (
                  <tr className="antifraud-empty">
                    <td colSpan={5}>
                      <AntifraudIcon name="shield" />
                      <strong>Нарушений не найдено</strong>
                      <span>Измените фильтры или обновите журнал.</span>
                    </td>
                  </tr>
                )}
                {!loading && logs.map((item) => (
                  <tr key={`${item.session_id}-${item.student_id}-${item.marked_at}`}>
                    <td data-label="Студент">
                      <strong>{item.student_name}</strong>
                      <small>{item.group_name || 'Группа не указана'}</small>
                    </td>
                    <td data-label="Занятие">
                      <strong>{item.subject_name || 'Предмет не указан'}</strong>
                      <small>{item.teacher_name || 'Преподаватель не указан'}</small>
                    </td>
                    <td data-label="Нарушение">
                      <span className={`antifraud-reason ${reasonClass(item.fraud_reason)}`}>
                        {reasonLabel(item.fraud_reason)}
                      </span>
                    </td>
                    <td data-label="Устройство и координаты" className="antifraud-device-cell">
                      <strong>{item.device_id || 'Устройство не указано'}</strong>
                      <small>{formatCoordinates(item.check_in_lat, item.check_in_lon)}</small>
                    </td>
                    <td data-label="Время" className="antifraud-date-cell">{formatDateTime(item.marked_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {!loading && total > 0 && (
            <div className="admin-pagination antifraud-pagination">
              <span>Показано {logs.length} из {total}</span>
              <div>
                <button type="button" onClick={() => setPage((value) => Math.max(1, value - 1))} disabled={page <= 1} aria-label="Предыдущая страница">‹</button>
                <strong>{page} из {pages}</strong>
                <button type="button" onClick={() => setPage((value) => Math.min(pages, value + 1))} disabled={page >= pages} aria-label="Следующая страница">›</button>
              </div>
            </div>
          )}
        </>
      ) : (
        <div className={`antifraud-ranking ${loading ? 'is-loading' : ''}`}>
          {loading && [0, 1, 2, 3].map((item) => <div key={item} className="antifraud-ranking-skeleton" />)}
          {!loading && ranking.length === 0 && (
            <div className="antifraud-ranking-empty">
              <AntifraudIcon name="ranking" />
              <strong>Рейтинг пока пуст</strong>
              <span>В выбранной области нет зафиксированных нарушений.</span>
            </div>
          )}
          {!loading && ranking.map((item, index) => (
            <article key={item.student_id} className={index < 3 ? `is-top is-top-${index + 1}` : ''}>
              <span className="antifraud-rank-number">{index + 1}</span>
              <div className="antifraud-rank-person">
                <strong>{item.student_name}</strong>
                <span>{item.group_name || 'Группа не указана'}</span>
              </div>
              <div className="antifraud-rank-last">
                <span>Последняя попытка</span>
                <strong>{formatDateTime(item.last_attempt_at)}</strong>
              </div>
              <div className="antifraud-rank-count">
                <strong>{item.total_cheat_attempts}</strong>
                <span>нарушений</span>
              </div>
            </article>
          ))}
        </div>
      )}
    </section>
  );
};

export default AntifraudPage;
