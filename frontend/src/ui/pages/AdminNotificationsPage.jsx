import React, { useCallback, useEffect, useMemo, useState } from "react";
import api from "../../services/api";
import { SearchableMultiSelect } from "../components/SearchableSelect";

const PAGE_SIZE = 12;

const ROLE_OPTIONS = [
  { value: "student", label: "Студенты" },
  { value: "teacher", label: "Преподаватели" },
  { value: "head", label: "Заведующие кафедрами" },
  { value: "dean", label: "Деканы" },
  { value: "admin", label: "Администраторы" }
];

const AUDIENCE_OPTIONS = [
  { value: "all", label: "Все активные пользователи" },
  { value: "role", label: "Пользователи выбранной роли" },
  { value: "groups", label: "Учебные группы" },
  { value: "users", label: "Отдельные пользователи" }
];

const CATEGORY_OPTIONS = [
  {
    value: "system",
    label: "Системное",
    events: [
      { value: "admin_update", label: "Объявление администратора" },
      { value: "fraud", label: "Нарушение" }
    ]
  },
  {
    value: "schedule",
    label: "Расписание",
    events: [
      { value: "lesson_created", label: "Добавлена пара" },
      { value: "lesson_rescheduled", label: "Пара перенесена" },
      { value: "lesson_cancelled", label: "Пара отменена" }
    ]
  },
  {
    value: "grades",
    label: "Оценки",
    events: [
      { value: "grade_created", label: "Добавлена оценка" },
      { value: "grade_updated", label: "Оценка изменена" }
    ]
  },
  {
    value: "attendance",
    label: "Посещаемость",
    events: [
      { value: "attendance_opened", label: "Открыта отметка" },
      { value: "attendance_marked", label: "Посещение отмечено" },
      { value: "attendance_rejected", label: "Отметка отклонена" }
    ]
  }
];

const EMPTY_FORM = {
  title: "",
  message: "",
  category: "system",
  event_type: "admin_update",
  audience: "all",
  role: "student",
  user_ids: [],
  group_ids: [],
  expires_at: ""
};

const categoryLabel = (value) => (
  CATEGORY_OPTIONS.find((item) => item.value === value)?.label || value || "Системное"
);

const formatDateTime = (value) => {
  if (!value) return "Без срока";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit"
  }).format(date);
};

const toDateTimeLocal = (value) => {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const offset = date.getTimezoneOffset() * 60000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
};

const NotificationAdminIcon = ({ name }) => {
  const icons = {
    bell: (
      <>
        <path d="M6.5 10.5a5.5 5.5 0 0 1 11 0c0 5 2 5.5 2 5.5h-15s2-.5 2-5.5Z" />
        <path d="M10 19h4" />
      </>
    ),
    add: <path d="M12 5v14M5 12h14" />,
    refresh: (
      <>
        <path d="M19 8a7.5 7.5 0 1 0 .5 7" />
        <path d="M19 3v5h-5" />
      </>
    ),
    edit: <><path d="m4 20 4.3-1 10-10-3.3-3.3-10 10L4 20Z" /><path d="m13.8 6.8 3.4 3.4" /></>,
    trash: <><path d="M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13" /><path d="M10 11v5M14 11v5" /></>,
    previous: <path d="m15 18-6-6 6-6" />,
    next: <path d="m9 18 6-6-6-6" />
  };

  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      {icons[name] || icons.bell}
    </svg>
  );
};

const AdminNotificationsPage = ({ token, onNotificationCreated }) => {
  const [items, setItems] = useState([]);
  const [page, setPage] = useState(1);
  const [category, setCategory] = useState("");
  const [pagination, setPagination] = useState({ page: 1, pages: 0, total: 0 });
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [deletingId, setDeletingId] = useState(0);
  const [composerOpen, setComposerOpen] = useState(false);
  const [editingId, setEditingId] = useState(0);
  const [form, setForm] = useState(EMPTY_FORM);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [reloadKey, setReloadKey] = useState(0);

  const [catalogs, setCatalogs] = useState({ groups: [], lecterns: [], faculties: [] });
  const [usersList, setUsersList] = useState([]);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const [cats, usersRes] = await Promise.all([
          api.getAdminCatalogs(token),
          api.getAdminUsers(token, { page_size: 1000 })
        ]);
        if (active) {
          setCatalogs(cats || {});
          setUsersList(Array.isArray(usersRes?.items) ? usersRes.items : []);
        }
      } catch (err) {
        console.error("Ошибка загрузки справочников для уведомлений:", err);
      }
    })();
    return () => { active = false; };
  }, [token]);

  const groupOptions = useMemo(() => {
    return (catalogs.groups || []).map((g) => ({
      id: g.id,
      name: g.name,
      sub: g.lectern_name ? `Кафедра: ${g.lectern_name}` : ""
    }));
  }, [catalogs.groups]);

  const userOptions = useMemo(() => {
    return usersList.map((u) => {
      const roleName = ROLE_OPTIONS.find((r) => r.value === u.role)?.label || u.role || "Пользователь";
      return {
        id: u.user_id,
        name: u.full_name || u.login,
        sub: `${roleName} • ${u.login}${u.group_name ? ` • ${u.group_name}` : ""}`
      };
    });
  }, [usersList]);

  const selectedCategory = useMemo(
    () => CATEGORY_OPTIONS.find((item) => item.value === form.category) || CATEGORY_OPTIONS[0],
    [form.category]
  );

  const loadNotifications = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const payload = await api.getAdminNotifications(token, {
        page,
        page_size: PAGE_SIZE,
        category
      });
      const nextPagination = payload?.pagination || {};
      setItems(Array.isArray(payload?.items) ? payload.items : []);
      setPagination({
        page: Number(nextPagination.page) || page,
        pages: Number(nextPagination.pages) || 0,
        total: Number(nextPagination.total) || 0
      });
    } catch (requestError) {
      setItems([]);
      setError(api.getErrorMessage(requestError, "Не удалось загрузить рассылки"));
    } finally {
      setLoading(false);
    }
  }, [category, page, token]);

  useEffect(() => {
    loadNotifications();
  }, [loadNotifications, reloadKey]);

  useEffect(() => {
    if (!notice) return undefined;
    const timer = window.setTimeout(() => setNotice(""), 4500);
    return () => window.clearTimeout(timer);
  }, [notice]);

  const openCreate = () => {
    setEditingId(0);
    setForm(EMPTY_FORM);
    setError("");
    setComposerOpen(true);
  };

  const openEdit = (item) => {
    setEditingId(Number(item.notification_id) || 0);
    setForm({
      ...EMPTY_FORM,
      title: item.title || "",
      message: item.message || "",
      category: item.category || "system",
      event_type: item.event_type || "admin_update",
      expires_at: toDateTimeLocal(item.expires_at)
    });
    setError("");
    setComposerOpen(true);
  };

  const closeComposer = () => {
    if (submitting) return;
    setComposerOpen(false);
    setEditingId(0);
    setForm(EMPTY_FORM);
    setError("");
  };

  const updateForm = (event) => {
    const { name, value } = event.target;
    setForm((current) => {
      const updated = { ...current, [name]: value };
      if (name === "category") {
        const matchingCategory = CATEGORY_OPTIONS.find((item) => item.value === value) || CATEGORY_OPTIONS[0];
        updated.event_type = matchingCategory.events[0]?.value || "admin_update";
      }
      return updated;
    });
    setError("");
  };

  const submitComposer = async (event) => {
    event.preventDefault();
    setError("");

    if (!form.title.trim()) {
      setError("Укажите заголовок уведомления.");
      return;
    }
    if (!form.message.trim()) {
      setError("Укажите текст уведомления.");
      return;
    }

    setSubmitting(true);
    try {
      if (editingId) {
        await api.updateAdminNotification(token, editingId, {
          title: form.title.trim(),
          message: form.message.trim(),
          expires_at: form.expires_at ? new Date(form.expires_at).toISOString() : null
        });
        setNotice("Рассылка успешно обновлена.");
      } else {
        const payload = {
          title: form.title.trim(),
          message: form.message.trim(),
          category: form.category,
          event_type: form.event_type,
          audience: form.audience,
          role: form.audience === "role" ? form.role : undefined,
          user_ids: form.audience === "users" ? form.user_ids : undefined,
          group_ids: form.audience === "groups" ? form.group_ids : undefined,
          expires_at: form.expires_at ? new Date(form.expires_at).toISOString() : undefined
        };

        if (form.audience === "users" && (!payload.user_ids || payload.user_ids.length === 0)) {
          setError("Выберите хотя бы одного пользователя.");
          setSubmitting(false);
          return;
        }
        if (form.audience === "groups" && (!payload.group_ids || payload.group_ids.length === 0)) {
          setError("Выберите хотя бы одну учебную группу.");
          setSubmitting(false);
          return;
        }

        await api.createAdminNotification(token, payload);
        setNotice("Рассылка успешно создана и отправлена.");
        if (typeof onNotificationCreated === "function") {
          onNotificationCreated();
        }
      }

      closeComposer();
      setPage(1);
      setReloadKey((current) => current + 1);
    } catch (submitError) {
      setError(api.getErrorMessage(submitError, "Не удалось сохранить рассылку."));
    } finally {
      setSubmitting(false);
    }
  };

  const deleteNotification = async (item) => {
    const id = Number(item.notification_id) || 0;
    if (!id) return;
    if (!window.confirm(`Удалить рассылку "${item.title || "Без названия"}"?`)) {
      return;
    }
    setDeletingId(id);
    setError("");
    try {
      await api.deleteAdminNotification(token, id);
      setNotice("Рассылка удалена.");
      setReloadKey((current) => current + 1);
    } catch (deleteError) {
      setError(api.getErrorMessage(deleteError, "Не удалось удалить рассылку."));
    } finally {
      setDeletingId(0);
    }
  };

  return (
    <section className="admin-notifications-page">
      <header className="admin-users-header">
        <div>
          <span>Администрирование</span>
          <h1>Уведомления</h1>
          <p>Системные сообщения и история отправленных рассылок.</p>
        </div>
        <button type="button" className="admin-primary-button" onClick={openCreate}>
          <NotificationAdminIcon name="add" />
          Создать рассылку
        </button>
      </header>

      {(error || notice) && (
        <div className={`admin-notice ${error ? "is-error" : "is-success"}`} role={error ? "alert" : "status"}>
          {error || notice}
        </div>
      )}

      {composerOpen && (
        <section className="admin-notification-composer" aria-labelledby="notification-composer-title">
          <div className="admin-notification-composer-heading">
            <span className="admin-notification-composer-icon">
              <NotificationAdminIcon name="bell" />
            </span>
            <div>
              <span>{editingId ? "Редактирование" : "Новая рассылка"}</span>
              <h2 id="notification-composer-title">
                {editingId ? "Изменить уведомление" : "Отправить уведомление"}
              </h2>
            </div>
          </div>

          <form className="admin-notification-form" onSubmit={submitComposer}>
            <label className="admin-form-field is-wide">
              <span>Заголовок *</span>
              <input name="title" value={form.title} onChange={updateForm} maxLength={255} required autoFocus />
            </label>

            <label className="admin-form-field is-wide">
              <span>Текст уведомления *</span>
              <textarea name="message" value={form.message} onChange={updateForm} maxLength={10000} rows={4} required />
            </label>

            {!editingId && (
              <>
                <label className="admin-form-field">
                  <span>Категория</span>
                  <select name="category" value={form.category} onChange={updateForm}>
                    {CATEGORY_OPTIONS.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </label>

                <label className="admin-form-field">
                  <span>Событие</span>
                  <select name="event_type" value={form.event_type} onChange={updateForm}>
                    {selectedCategory.events.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </label>

                <label className="admin-form-field">
                  <span>Получатели</span>
                  <select name="audience" value={form.audience} onChange={updateForm}>
                    {AUDIENCE_OPTIONS.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </label>

                {form.audience === "role" && (
                  <label className="admin-form-field">
                    <span>Роль</span>
                    <select name="role" value={form.role} onChange={updateForm}>
                      {ROLE_OPTIONS.map((option) => (
                        <option key={option.value} value={option.value}>{option.label}</option>
                      ))}
                    </select>
                  </label>
                )}

                {form.audience === "groups" && (
                  <div className="admin-form-field">
                    <span>Учебные группы *</span>
                    <SearchableMultiSelect
                      options={groupOptions}
                      values={form.group_ids}
                      onChange={(nextIds) => setForm((prev) => ({ ...prev, group_ids: nextIds }))}
                      placeholder="Поиск группы по названию..."
                      allLabel="Выбрать все группы"
                    />
                  </div>
                )}

                {form.audience === "users" && (
                  <div className="admin-form-field">
                    <span>Пользователи *</span>
                    <SearchableMultiSelect
                      options={userOptions}
                      values={form.user_ids}
                      onChange={(nextIds) => setForm((prev) => ({ ...prev, user_ids: nextIds }))}
                      placeholder="Поиск по ФИО, логину или группе..."
                      allLabel="Выбрать всех"
                    />
                  </div>
                )}
              </>
            )}

            <label className="admin-form-field">
              <span>Действует до</span>
              <input name="expires_at" type="datetime-local" value={form.expires_at} onChange={updateForm} />
            </label>

            <footer>
              <button type="button" className="admin-secondary-button" onClick={closeComposer} disabled={submitting}>
                Отмена
              </button>
              <button type="submit" className="admin-primary-button" disabled={submitting}>
                {submitting ? "Сохраняем..." : editingId ? "Сохранить" : "Отправить"}
              </button>
            </footer>
          </form>
        </section>
      )}

      <div className="admin-notifications-toolbar">
        <label className="admin-filter-field">
          <span>Категория</span>
          <select value={category} onChange={(event) => { setCategory(event.target.value); setPage(1); }}>
            <option value="">Все категории</option>
            {CATEGORY_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
        </label>
        <span className="admin-notifications-total">Всего: <strong>{pagination.total}</strong></span>
        <button
          type="button"
          className="semester-icon-button ui-refresh-button"
          aria-label="Обновить список"
          title="Обновить список"
          onClick={() => setReloadKey((current) => current + 1)}
          disabled={loading}
        >
          <NotificationAdminIcon name="refresh" />
        </button>
      </div>

      <div className={`admin-notifications-table-wrap ${loading ? "is-loading" : ""}`}>
        <table className="admin-notifications-table">
          <thead>
            <tr>
              <th>Категория</th>
              <th>Сообщение</th>
              <th>Получатели</th>
              <th>Создано</th>
              <th>Действует до</th>
              <th aria-label="Действия" />
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr className="admin-notifications-empty">
                <td colSpan={6}>Загружаем рассылки...</td>
              </tr>
            )}
            {!loading && items.length === 0 && (
              <tr className="admin-notifications-empty">
                <td colSpan={6}>Рассылки не найдены.</td>
              </tr>
            )}
            {items.map((item) => (
              <tr key={item.notification_id}>
                <td data-label="Категория">
                  <span className={`admin-notification-category is-${item.category || "system"}`}>
                    {categoryLabel(item.category)}
                  </span>
                </td>
                <td data-label="Сообщение">
                  <strong className="admin-notification-title">{item.title || "Без названия"}</strong>
                  <small className="admin-notification-message">{item.message || "—"}</small>
                </td>
                <td data-label="Получатели">{Number(item.recipient_count) || 0}</td>
                <td data-label="Создано">{formatDateTime(item.created_at)}</td>
                <td data-label="Действует до">{formatDateTime(item.expires_at)}</td>
                <td data-label="Действия">
                  <div className="admin-row-actions">
                    <button type="button" aria-label="Редактировать" title="Редактировать" onClick={() => openEdit(item)}>
                      <NotificationAdminIcon name="edit" />
                    </button>
                    <button
                      type="button"
                      className="is-danger"
                      aria-label="Удалить"
                      title="Удалить"
                      onClick={() => deleteNotification(item)}
                      disabled={deletingId === item.notification_id}
                    >
                      <NotificationAdminIcon name="trash" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {pagination.pages > 1 && (
        <div className="admin-pagination">
          <span>Страница {pagination.page} из {pagination.pages}</span>
          <div>
            <button
              type="button"
              aria-label="Предыдущая страница"
              onClick={() => setPage((current) => Math.max(1, current - 1))}
              disabled={page <= 1 || loading}
            >
              <NotificationAdminIcon name="previous" />
            </button>
            <button
              type="button"
              aria-label="Следующая страница"
              onClick={() => setPage((current) => Math.min(pagination.pages, current + 1))}
              disabled={page >= pagination.pages || loading}
            >
              <NotificationAdminIcon name="next" />
            </button>
          </div>
        </div>
      )}
    </section>
  );
};

export default AdminNotificationsPage;
