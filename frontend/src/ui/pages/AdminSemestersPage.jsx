import React, { useCallback, useEffect, useMemo, useState } from "react";
import api from "../../services/api";

const STATUS_META = {
  open: { label: "Текущий", className: "is-open" },
  planned: { label: "Запланирован", className: "is-planned" },
  closed: { label: "Закрыт", className: "is-closed" },
  archived: { label: "В архиве", className: "is-archived" }
};

const FILTER_OPTIONS = [
  { value: "all", label: "Все" },
  { value: "planned", label: "План" },
  { value: "open", label: "Текущий" },
  { value: "closed", label: "Закрытые" },
  { value: "archived", label: "Архив" }
];

const formatDate = (value) => {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "long",
    year: "numeric"
  }).format(date);
};

const parseAcademicYear = (academicYear) => {
  const match = /^(\d{4})\/(\d{4})$/.exec(academicYear.trim());
  if (!match || Number(match[2]) !== Number(match[1]) + 1) return null;
  return { first: Number(match[1]), second: Number(match[2]) };
};

const getTermDates = (academicYear, termNum) => {
  const years = parseAcademicYear(academicYear);
  if (!years) return null;
  return Number(termNum) === 1
    ? { starts_at: `${years.first}-09-01`, ends_at: `${years.second}-01-31` }
    : { starts_at: `${years.second}-02-01`, ends_at: `${years.second}-06-30` };
};

const getTermName = (academicYear, termNum) => (
  `${academicYear}, ${Number(termNum) === 1 ? "осенний" : "весенний"} семестр`
);

const createDefaultForm = () => {
  const now = new Date();
  const month = now.getMonth();
  const firstYear = month >= 6 ? now.getFullYear() : now.getFullYear() - 1;
  const academicYear = `${firstYear}/${firstYear + 1}`;
  const termNum = month >= 1 && month <= 5 ? 2 : 1;
  const dates = getTermDates(academicYear, termNum);
  return {
    academic_year: academicYear,
    term_num: termNum,
    name: getTermName(academicYear, termNum),
    starts_at: dates?.starts_at || "",
    ends_at: dates?.ends_at || ""
  };
};

const getSemesterProgress = (semester) => {
  if (!semester?.starts_at || !semester?.ends_at) return 0;
  const start = new Date(semester.starts_at).getTime();
  const end = new Date(semester.ends_at).getTime();
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return 0;
  const progress = ((Date.now() - start) / (end - start)) * 100;
  return Math.min(100, Math.max(0, Math.round(progress)));
};

const getSemesterError = (error, fallback) => {
  const message = api.getErrorMessage(error, fallback);
  const translations = {
    "semester already exists": "Семестр с таким учебным годом и номером уже существует.",
    "semester date range overlaps an existing semester": "Период пересекается с уже созданным семестром.",
    "semester has not started": "Семестр нельзя открыть раньше даты начала.",
    "ended semester cannot be opened": "Нельзя открыть семестр, срок которого уже завершился.",
    "invalid semester status transition": "Это действие недоступно для текущего статуса семестра.",
    "semester has active attendance sessions": "Сначала завершите активные сессии посещаемости.",
    "semester not found": "Семестр не найден. Обновите список и повторите действие."
  };
  return translations[message] || message;
};

const getFilterCount = (semesters, filter) => {
  if (filter === "all") return semesters.length;
  return semesters.filter((semester) => (
    filter === "open"
      ? semester.status === "open" || semester.is_current
      : semester.status === filter
  )).length;
};

const SemesterIcon = ({ name }) => {
  const paths = {
    calendar: (
      <>
        <rect x="3.5" y="5.5" width="17" height="15" rx="2.5" />
        <path d="M8 3.5v4M16 3.5v4M3.5 10h17M8 14h3M13 14h3M8 17.5h3" />
      </>
    ),
    plus: <path d="M12 5v14M5 12h14" />,
    refresh: (
      <>
        <path d="M19 8a7.5 7.5 0 1 0 .5 7" />
        <path d="M19 3v5h-5" />
      </>
    ),
    arrow: <path d="m9 6 6 6-6 6" />,
    close: <path d="m6 6 12 12M18 6 6 18" />,
    archive: (
      <>
        <path d="M4 7h16v13H4zM3 4h18v3H3zM9 11h6" />
      </>
    ),
    trash: (
      <>
        <path d="M5 7h14M9 7V4h6v3M7 7l1 13h8l1-13M10 11v5M14 11v5" />
      </>
    ),
    lock: (
      <>
        <rect x="5" y="10" width="14" height="10" rx="2" />
        <path d="M8 10V7a4 4 0 0 1 8 0v3" />
      </>
    )
  };

  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      {paths[name] || paths.calendar}
    </svg>
  );
};

const AdminSemestersPage = ({ token }) => {
  const [semesters, setSemesters] = useState([]);
  const [currentSemester, setCurrentSemester] = useState(null);
  const [statusFilter, setStatusFilter] = useState("all");
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [creating, setCreating] = useState(false);
  const [actionLoading, setActionLoading] = useState(null);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [actionDialog, setActionDialog] = useState(null);
  const [form, setForm] = useState(createDefaultForm);

  const loadData = useCallback(async (quiet = false) => {
    if (quiet) setRefreshing(true);
    else setLoading(true);
    setError("");

    try {
      const listData = await api.getSemesters(token);
      const list = Array.isArray(listData?.items)
        ? listData.items
        : Array.isArray(listData?.semesters)
          ? listData.semesters
          : Array.isArray(listData)
            ? listData
            : [];
      setSemesters(list);

      try {
        const currentData = await api.getCurrentSemester(token);
        setCurrentSemester(currentData?.semester || currentData || null);
      } catch {
        setCurrentSemester(list.find((semester) => (
          semester.status === "open" || semester.is_current
        )) || null);
      }
    } catch (loadError) {
      setError(getSemesterError(loadError, "Не удалось загрузить семестры."));
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [token]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  useEffect(() => {
    if (!success) return undefined;
    const timeout = window.setTimeout(() => setSuccess(""), 5000);
    return () => window.clearTimeout(timeout);
  }, [success]);

  useEffect(() => {
    if (!isCreateOpen && !actionDialog) return undefined;
    const handleKeyDown = (event) => {
      if (event.key !== "Escape" || creating || actionLoading) return;
      setIsCreateOpen(false);
      setActionDialog(null);
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [actionDialog, actionLoading, creating, isCreateOpen]);

  const openSemester = currentSemester || semesters.find((semester) => (
    semester.status === "open" || semester.is_current
  ));

  const nearestPlannedSemester = useMemo(() => (
    semesters
      .filter((semester) => semester.status === "planned")
      .sort((left, right) => (
        new Date(left.starts_at).getTime() - new Date(right.starts_at).getTime()
      ))[0] || null
  ), [semesters]);

  const visibleSemesters = useMemo(() => (
    semesters
      .filter((semester) => (
        statusFilter === "all"
          || (statusFilter === "open" && (semester.status === "open" || semester.is_current))
          || semester.status === statusFilter
      ))
      .sort((left, right) => (
        new Date(right.starts_at).getTime() - new Date(left.starts_at).getTime()
      ))
  ), [semesters, statusFilter]);

  const resetAndOpenCreate = () => {
    setForm(createDefaultForm());
    setError("");
    setIsCreateOpen(true);
  };

  const updateAcademicYear = (academicYear) => {
    setForm((current) => {
      const dates = getTermDates(academicYear, current.term_num);
      return {
        ...current,
        academic_year: academicYear,
        name: getTermName(academicYear, current.term_num),
        ...(dates || {})
      };
    });
  };

  const updateTerm = (termNum) => {
    setForm((current) => {
      const dates = getTermDates(current.academic_year, termNum);
      return {
        ...current,
        term_num: termNum,
        name: getTermName(current.academic_year, termNum),
        ...(dates || {})
      };
    });
  };

  const handleCreateSubmit = async (event) => {
    event.preventDefault();
    setError("");
    setSuccess("");

    if (!parseAcademicYear(form.academic_year)) {
      setError("Учебный год должен иметь формат 2026/2027 и состоять из последовательных лет.");
      return;
    }
    if (new Date(form.ends_at).getTime() <= new Date(form.starts_at).getTime()) {
      setError("Дата окончания должна быть позже даты начала.");
      return;
    }

    setCreating(true);
    try {
      const payload = {
        academic_year: form.academic_year.trim(),
        term_num: Number(form.term_num),
        name: form.name.trim(),
        starts_at: new Date(`${form.starts_at}T00:00:00Z`).toISOString(),
        ends_at: new Date(`${form.ends_at}T23:59:59Z`).toISOString(),
        status: "planned"
      };
      await api.createAdminSemester(token, payload);
      setIsCreateOpen(false);
      setSuccess(`Семестр «${payload.name}» создан и добавлен в план.`);
      await loadData(true);
    } catch (createError) {
      setError(getSemesterError(createError, "Не удалось создать семестр."));
    } finally {
      setCreating(false);
    }
  };

  const openActionDialog = (type, semester) => {
    setError("");
    setSuccess("");
    setActionDialog({ type, semester });
  };

  const getActionCopy = () => {
    if (!actionDialog) return null;
    const name = actionDialog.semester.name;
    if (actionDialog.type === "activate") {
      return {
        eyebrow: "Смена учебного периода",
        title: "Открыть семестр?",
        description: openSemester
          ? `«${name}» станет текущим, а семестр «${openSemester.name}» будет закрыт.`
          : `«${name}» станет текущим семестром и откроет запись учебных данных.`,
        confirm: "Открыть семестр",
        className: ""
      };
    }
    if (actionDialog.type === "close") {
      return {
        eyebrow: "Завершение периода",
        title: "Закрыть текущий семестр?",
        description: `После закрытия данные семестра «${name}» станут доступны только для чтения.`,
        confirm: "Закрыть семестр",
        className: "is-warning"
      };
    }
    if (actionDialog.type === "archive") {
      return {
        eyebrow: "История обучения",
        title: "Перенести семестр в архив?",
        description: `Семестр «${name}» останется доступен в исторических отчётах.`,
        confirm: "В архив",
        className: ""
      };
    }
    return {
      eyebrow: "Необратимое действие",
      title: "Удалить семестр?",
      description: `Семестр «${name}» будет удалён. Отменить это действие нельзя.`,
      confirm: "Удалить",
      className: "is-danger"
    };
  };

  const confirmAction = async () => {
    if (!actionDialog) return;
    const { type, semester } = actionDialog;
    setActionLoading(semester.semester_id);
    setError("");

    try {
      if (type === "activate") await api.activateAdminSemester(token, semester.semester_id);
      if (type === "close") await api.closeAdminSemester(token, semester.semester_id);
      if (type === "archive") await api.archiveAdminSemester(token, semester.semester_id);
      if (type === "delete") await api.deleteAdminSemester(token, semester.semester_id);

      const messages = {
        activate: `Семестр «${semester.name}» открыт.`,
        close: `Семестр «${semester.name}» закрыт.`,
        archive: `Семестр «${semester.name}» перенесён в архив.`,
        delete: `Семестр «${semester.name}» удалён.`
      };
      setActionDialog(null);
      setSuccess(messages[type] || "Изменения сохранены.");
      await loadData(true);
    } catch (actionError) {
      setActionDialog(null);
      setError(getSemesterError(actionError, "Не удалось изменить статус семестра."));
    } finally {
      setActionLoading(null);
    }
  };

  const progress = getSemesterProgress(openSemester);
  const actionCopy = getActionCopy();

  return (
    <section className="semester-admin-page">
      <header className="semester-admin-heading">
        <div>
          <span>Панель администратора</span>
          <h1>Учебные семестры</h1>
          <p>Планируйте учебные периоды и управляйте состоянием текущего семестра.</p>
        </div>
        <div className="semester-heading-actions">
          <button
            type="button"
            className="semester-icon-button ui-refresh-button"
            onClick={() => loadData(true)}
            disabled={refreshing}
            title="Обновить список"
            aria-label="Обновить список семестров"
          >
            <SemesterIcon name="refresh" />
          </button>
          <button type="button" className="admin-primary-button" onClick={resetAndOpenCreate}>
            <SemesterIcon name="plus" />
            Новый семестр
          </button>
        </div>
      </header>

      {success && <div className="admin-notice is-success" role="status">{success}</div>}
      {error && <div className="admin-notice is-error" role="alert">{error}</div>}

      <section className={`semester-current-panel ${openSemester ? "" : "is-empty"}`}>
        <div className="semester-current-icon">
          <SemesterIcon name={openSemester ? "calendar" : "lock"} />
        </div>
        <div className="semester-current-main">
          <span className="semester-current-label">
            <i />
            {openSemester ? "Текущий учебный период" : "Межсезонье"}
          </span>
          <h2>{openSemester?.name || "Открытого семестра нет"}</h2>
          <p>
            {openSemester
              ? `${formatDate(openSemester.starts_at)} — ${formatDate(openSemester.ends_at)}`
              : nearestPlannedSemester
                ? `Ближайший в плане: ${nearestPlannedSemester.name}, с ${formatDate(nearestPlannedSemester.starts_at)}`
                : "Создайте учебный период, чтобы добавить его в план."}
          </p>
          {openSemester && (
            <div className="semester-progress" aria-label={`Семестр завершён на ${progress}%`}>
              <div><span style={{ width: `${progress}%` }} /></div>
              <strong>{progress}% периода</strong>
            </div>
          )}
        </div>
        <div className="semester-current-aside">
          {openSemester ? (
            <>
              <span>Семестр №{openSemester.term_num}</span>
              <strong>{openSemester.academic_year}</strong>
              <button type="button" onClick={() => openActionDialog("close", openSemester)}>
                Закрыть семестр
              </button>
            </>
          ) : (
            <button type="button" onClick={resetAndOpenCreate}>Создать семестр</button>
          )}
        </div>
      </section>

      <section className="semester-list-panel">
        <div className="semester-list-toolbar">
          <div>
            <span>Учебные периоды</span>
            <h2>Все семестры</h2>
          </div>
          <div className="semester-filter" aria-label="Фильтр по статусу">
            {FILTER_OPTIONS.map((option) => (
              <button
                key={option.value}
                type="button"
                className={statusFilter === option.value ? "is-active" : ""}
                onClick={() => setStatusFilter(option.value)}
              >
                {option.label}
                <span>{getFilterCount(semesters, option.value)}</span>
              </button>
            ))}
          </div>
        </div>

        <div className={`semester-list ${loading || refreshing ? "is-loading" : ""}`}>
          <div className="semester-list-head" aria-hidden="true">
            <span>№</span>
            <span>Семестр</span>
            <span>Период</span>
            <span>Статус</span>
            <span>Действия</span>
          </div>

          {loading ? (
            <div className="semester-empty-state">Загружаем учебные периоды...</div>
          ) : visibleSemesters.length === 0 ? (
            <div className="semester-empty-state">
              <SemesterIcon name="calendar" />
              <strong>Семестры не найдены</strong>
              <span>Измените фильтр или создайте новый учебный период.</span>
            </div>
          ) : visibleSemesters.map((semester) => {
            const isOpen = semester.status === "open" || semester.is_current;
            const status = isOpen ? "open" : semester.status;
            const statusMeta = STATUS_META[status] || { label: status, className: "is-closed" };
            const isLoadingRow = actionLoading === semester.semester_id;
            const startsAt = new Date(semester.starts_at).getTime();
            const endsAt = new Date(semester.ends_at).getTime();
            const canActivate = status === "planned" && Date.now() >= startsAt && Date.now() < endsAt;

            return (
              <article className={`semester-row ${statusMeta.className}`} key={semester.semester_id}>
                <div className="semester-row-number" data-label="Номер">{semester.term_num}</div>
                <div className="semester-row-title" data-label="Семестр">
                  <strong>{semester.name}</strong>
                  <span>{semester.academic_year} · ID {semester.semester_id}</span>
                </div>
                <div className="semester-row-period" data-label="Период">
                  <strong>{formatDate(semester.starts_at)}</strong>
                  <span>{formatDate(semester.ends_at)}</span>
                </div>
                <div data-label="Статус">
                  <span className={`semester-status ${statusMeta.className}`}>
                    <i />
                    {statusMeta.label}
                  </span>
                </div>
                <div className="semester-row-actions" data-label="Действия">
                  {canActivate && (
                    <button
                      type="button"
                      className="is-primary"
                      onClick={() => openActionDialog("activate", semester)}
                      disabled={isLoadingRow}
                    >
                      Открыть
                      <SemesterIcon name="arrow" />
                    </button>
                  )}
                  {status === "planned" && !canActivate && (
                    <span className="semester-action-hint">
                      {Date.now() < startsAt ? `Открытие с ${formatDate(semester.starts_at)}` : "Срок завершён"}
                    </span>
                  )}
                  {isOpen && (
                    <button
                      type="button"
                      onClick={() => openActionDialog("close", semester)}
                      disabled={isLoadingRow}
                    >
                      Закрыть
                    </button>
                  )}
                  {status === "closed" && (
                    <button
                      type="button"
                      onClick={() => openActionDialog("archive", semester)}
                      disabled={isLoadingRow}
                    >
                      <SemesterIcon name="archive" />
                      В архив
                    </button>
                  )}
                  {!isOpen && (
                    <button
                      type="button"
                      className="is-danger"
                      onClick={() => openActionDialog("delete", semester)}
                      disabled={isLoadingRow}
                      title="Удалить семестр"
                      aria-label={`Удалить семестр ${semester.name}`}
                    >
                      <SemesterIcon name="trash" />
                    </button>
                  )}
                </div>
              </article>
            );
          })}
        </div>
      </section>

      {isCreateOpen && (
        <div
          className="admin-modal-backdrop"
          role="presentation"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget && !creating) setIsCreateOpen(false);
          }}
        >
          <section className="admin-modal semester-create-modal" role="dialog" aria-modal="true" aria-labelledby="create-semester-title">
            <header>
              <div>
                <span>Новый учебный период</span>
                <h2 id="create-semester-title">Создать семестр</h2>
              </div>
              <button type="button" onClick={() => setIsCreateOpen(false)} disabled={creating} aria-label="Закрыть окно">
                <SemesterIcon name="close" />
              </button>
            </header>
            <form onSubmit={handleCreateSubmit}>
              <div className="semester-form-intro">
                <SemesterIcon name="calendar" />
                <p>Семестр будет создан со статусом «Запланирован». Открыть его можно отдельным действием в установленный период.</p>
              </div>
              <div className="admin-form-grid">
                <label className="admin-form-field">
                  <span>Учебный год</span>
                  <input
                    type="text"
                    required
                    pattern="\d{4}/\d{4}"
                    placeholder="2026/2027"
                    value={form.academic_year}
                    onChange={(event) => updateAcademicYear(event.target.value)}
                  />
                </label>
                <label className="admin-form-field">
                  <span>Семестр</span>
                  <select value={form.term_num} onChange={(event) => updateTerm(Number(event.target.value))}>
                    <option value={1}>1 · Осенний</option>
                    <option value={2}>2 · Весенний</option>
                  </select>
                </label>
                <label className="admin-form-field is-wide">
                  <span>Название</span>
                  <input
                    type="text"
                    required
                    maxLength={255}
                    value={form.name}
                    onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
                  />
                </label>
                <label className="admin-form-field">
                  <span>Дата начала</span>
                  <input
                    type="date"
                    required
                    value={form.starts_at}
                    onChange={(event) => setForm((current) => ({ ...current, starts_at: event.target.value }))}
                  />
                </label>
                <label className="admin-form-field">
                  <span>Дата окончания</span>
                  <input
                    type="date"
                    required
                    min={form.starts_at}
                    value={form.ends_at}
                    onChange={(event) => setForm((current) => ({ ...current, ends_at: event.target.value }))}
                  />
                </label>
              </div>
              <footer>
                <button type="button" className="admin-secondary-button" onClick={() => setIsCreateOpen(false)} disabled={creating}>Отмена</button>
                <button type="submit" className="admin-primary-button" disabled={creating}>
                  {creating ? "Создание..." : "Создать семестр"}
                </button>
              </footer>
            </form>
          </section>
        </div>
      )}

      {actionDialog && actionCopy && (
        <div
          className="admin-modal-backdrop"
          role="presentation"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget && !actionLoading) setActionDialog(null);
          }}
        >
          <section className="admin-modal is-confirm semester-action-modal" role="dialog" aria-modal="true" aria-labelledby="semester-action-title">
            <header>
              <div>
                <span>{actionCopy.eyebrow}</span>
                <h2 id="semester-action-title">{actionCopy.title}</h2>
              </div>
              <button type="button" onClick={() => setActionDialog(null)} disabled={Boolean(actionLoading)} aria-label="Закрыть окно">
                <SemesterIcon name="close" />
              </button>
            </header>
            <div className="admin-confirm-body">
              <p>{actionCopy.description}</p>
              <footer>
                <button type="button" className="admin-secondary-button" onClick={() => setActionDialog(null)} disabled={Boolean(actionLoading)}>Отмена</button>
                <button
                  type="button"
                  className={`admin-primary-button ${actionCopy.className}`}
                  onClick={confirmAction}
                  disabled={Boolean(actionLoading)}
                >
                  {actionLoading ? "Выполняется..." : actionCopy.confirm}
                </button>
              </footer>
            </div>
          </section>
        </div>
      )}
    </section>
  );
};

export default AdminSemestersPage;
