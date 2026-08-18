import React, { useCallback, useEffect, useMemo, useState } from 'react';
import api from '../../services/api';

const ROLE_OPTIONS = [
  { value: 'student', label: 'Студент' },
  { value: 'teacher', label: 'Преподаватель' },
  { value: 'head', label: 'Зав. кафедрой' },
  { value: 'dean', label: 'Декан' },
  { value: 'admin', label: 'Администратор' }
];

const STATUS_OPTIONS = [
  { value: 'active', label: 'Активен' },
  { value: 'blocked', label: 'Заблокирован' },
  { value: 'archived', label: 'В архиве' }
];

const EMPTY_CREATE_FORM = {
  login: '',
  password: '',
  role: 'student',
  email: '',
  full_name: '',
  group_id: '',
  lectern_id: '',
  faculty_id: '',
  job_title: ''
};

const EMPTY_INVITE_FORM = {
  full_name: '',
  lectern_id: '',
  job_title: '',
  custom_code: ''
};

const roleLabel = (role) => ROLE_OPTIONS.find((item) => item.value === role)?.label || role || '—';
const statusLabel = (status) => STATUS_OPTIONS.find((item) => item.value === status)?.label || status || '—';

const formatDate = (value) => {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  return new Intl.DateTimeFormat('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric'
  }).format(date);
};

const formatDateTime = (value) => {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  return new Intl.DateTimeFormat('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date);
};

const numberOrUndefined = (value) => {
  const number = Number(value);
  return Number.isInteger(number) && number > 0 ? number : undefined;
};

const AdminIcon = ({ name }) => {
  const icons = {
    add: <path d="M12 5v14M5 12h14" />,
    search: <><circle cx="10.8" cy="10.8" r="6.8" /><path d="m16 16 4 4" /></>,
    refresh: <><path d="M19 7v5h-5" /><path d="M5.3 16.5A8 8 0 0 0 19 12M5 12a8 8 0 0 1 13.7-4.5" /></>,
    edit: <><path d="m4 20 4.3-1 10-10-3.3-3.3-10 10L4 20Z" /><path d="m13.8 6.8 3.4 3.4" /></>,
    archive: <><path d="M4 7h16v13H4V7Z" /><path d="M3 4h18v3H3V4ZM9 11h6" /></>,
    close: <path d="m6 6 12 12M18 6 6 18" />,
    previous: <path d="m15 18-6-6 6-6" />,
    next: <path d="m9 18 6-6-6-6" />,
    users: <><circle cx="9" cy="8" r="3" /><path d="M3.5 19c.6-3.4 2.4-5.2 5.5-5.2s4.9 1.8 5.5 5.2" /><path d="M15.5 5.8a3 3 0 0 1 0 5.8M16 14c2.6.4 4 2.1 4.5 5" /></>,
    copy: <><rect x="9" y="9" width="13" height="13" rx="2" ry="2" /><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" /></>,
    key: <><circle cx="7.5" cy="15.5" r="4.5" /><path d="m10.7 12.3 8.3-8.3M15 8l2 2M18 5l2 2" /></>,
    trash: <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
  };
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" style={{ width: 18, height: 18, fill: 'none', stroke: 'currentColor', strokeWidth: 2, strokeLinecap: 'round', strokeLinejoin: 'round' }}>
      {icons[name] || icons.users}
    </svg>
  );
};

const Field = ({ label, children, required = false, wide = false }) => (
  <label className={`admin-form-field ${wide ? 'is-wide' : ''}`}>
    <span>{label}{required ? ' *' : ''}</span>
    {children}
  </label>
);

const RoleSpecificCreateFields = ({ form, onChange }) => {
  if (form.role === 'student') {
    return (
      <>
        <Field label="ФИО" required wide>
          <input name="full_name" value={form.full_name} onChange={onChange} autoComplete="name" />
        </Field>
        <Field label="ID группы">
          <input name="group_id" value={form.group_id} onChange={onChange} inputMode="numeric" />
        </Field>
      </>
    );
  }
  if (form.role === 'teacher') {
    return (
      <>
        <Field label="ФИО" required wide>
          <input name="full_name" value={form.full_name} onChange={onChange} autoComplete="name" />
        </Field>
        <Field label="ID кафедры">
          <input name="lectern_id" value={form.lectern_id} onChange={onChange} inputMode="numeric" />
        </Field>
        <Field label="Должность">
          <input name="job_title" value={form.job_title} onChange={onChange} />
        </Field>
      </>
    );
  }
  if (form.role === 'head') {
    return (
      <Field label="ID кафедры" required>
        <input name="lectern_id" value={form.lectern_id} onChange={onChange} inputMode="numeric" />
      </Field>
    );
  }
  if (form.role === 'dean') {
    return (
      <Field label="ID факультета" required>
        <input name="faculty_id" value={form.faculty_id} onChange={onChange} inputMode="numeric" />
      </Field>
    );
  }
  return null;
};

const RoleTargetField = ({ role, value, onChange }) => {
  const config = {
    student: ['ID профиля студента', 'student_id'],
    teacher: ['ID профиля преподавателя', 'teacher_id'],
    head: ['ID кафедры', 'lectern_id'],
    dean: ['ID факультета', 'faculty_id']
  }[role];
  if (!config) return null;
  return (
    <Field label={config[0]} required wide>
      <input name="target_id" value={value} onChange={onChange} inputMode="numeric" />
    </Field>
  );
};

const AdminUsersPage = ({ token, currentUser }) => {
  const [activeTab, setActiveTab] = useState('users'); // 'users' | 'invites'
  
  // Users state
  const [items, setItems] = useState([]);
  const [pagination, setPagination] = useState({ page: 1, page_size: 20, total: 0, pages: 0 });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [searchDraft, setSearchDraft] = useState('');
  const [filters, setFilters] = useState({ search: '', role: '', status: '' });
  
  // Invites state
  const [invites, setInvites] = useState([]);
  const [inviteStatusFilter, setInviteStatusFilter] = useState(''); // '' (all) | 'pending' | 'used'
  const [invitesLoading, setInvitesLoading] = useState(false);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [reloadKey, setReloadKey] = useState(0);
  const [modal, setModal] = useState(null);

  const loadUsers = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const payload = await api.getAdminUsers(token, {
        page,
        page_size: pageSize,
        search: filters.search,
        role: filters.role,
        status: filters.status
      });
      const nextPagination = payload?.pagination || {};
      setItems(Array.isArray(payload?.items) ? payload.items : []);
      setPagination({
        page: Number(nextPagination.page) || page,
        page_size: Number(nextPagination.page_size) || pageSize,
        total: Number(nextPagination.total) || 0,
        pages: Number(nextPagination.pages) || 0
      });
    } catch (requestError) {
      setItems([]);
      setError(api.getErrorMessage(requestError, 'Не удалось загрузить пользователей'));
    } finally {
      setLoading(false);
    }
  }, [filters.role, filters.search, filters.status, page, pageSize, token]);

  const loadInvites = useCallback(async () => {
    setInvitesLoading(true);
    setError('');
    try {
      const list = await api.getAdminInvites(token, {
        role: 'teacher',
        status: inviteStatusFilter
      });
      setInvites(Array.isArray(list) ? list : []);
    } catch (requestError) {
      setInvites([]);
      setError(api.getErrorMessage(requestError, 'Не удалось загрузить список инвайтов'));
    } finally {
      setInvitesLoading(false);
    }
  }, [inviteStatusFilter, token]);

  useEffect(() => {
    if (activeTab === 'users') {
      loadUsers();
    } else {
      loadInvites();
    }
  }, [activeTab, loadUsers, loadInvites, reloadKey]);

  useEffect(() => {
    if (!notice) return undefined;
    const timer = window.setTimeout(() => setNotice(''), 4000);
    return () => window.clearTimeout(timer);
  }, [notice]);

  const pageStart = pagination.total ? (pagination.page - 1) * pagination.page_size + 1 : 0;
  const pageEnd = Math.min(pagination.total, pagination.page * pagination.page_size);
  const currentUserId = Number(currentUser?.user_id || currentUser?.id || 0);

  const pageStats = useMemo(() => ({
    active: items.filter((item) => item.status === 'active').length,
    blocked: items.filter((item) => item.status === 'blocked').length,
    withTwoFa: items.filter((item) => item.two_fa_enabled).length
  }), [items]);

  const inviteStats = useMemo(() => ({
    total: invites.length,
    pending: invites.filter((inv) => !inv.used_at).length,
    registered: invites.filter((inv) => !!inv.used_at).length
  }), [invites]);

  const copyInviteLink = (code) => {
    const origin = window.location.origin + window.location.pathname;
    const link = `${origin}#/register?code=${encodeURIComponent(code)}`;
    navigator.clipboard.writeText(link).then(() => {
      setNotice(`Ссылка для инвайта ${code} скопирована!`);
    }).catch(() => {
      setNotice(`Код инвайта: ${code}`);
    });
  };

  const submitSearch = (event) => {
    event.preventDefault();
    setPage(1);
    setFilters((current) => ({ ...current, search: searchDraft.trim() }));
  };

  const changeFilter = (name, value) => {
    setPage(1);
    setFilters((current) => ({ ...current, [name]: value }));
  };

  const openCreate = () => {
    setModal({ type: 'create', form: { ...EMPTY_CREATE_FORM }, error: '', saving: false });
  };

  const openCreateInvite = () => {
    setModal({ type: 'create_invite', form: { ...EMPTY_INVITE_FORM }, error: '', saving: false });
  };

  const openEdit = async (item) => {
    setModal({ type: 'edit', userId: item.user_id, loading: true, error: '', saving: false });
    try {
      const user = await api.getAdminUser(token, item.user_id);
      setModal({
        type: 'edit',
        userId: item.user_id,
        original: user,
        loading: false,
        error: '',
        saving: false,
        form: {
          login: user.login || '',
          email: user.email || '',
          role: user.role || 'student',
          status: user.status || 'active',
          password: '',
          target_id: ''
        }
      });
    } catch (requestError) {
      setModal({
        type: 'edit',
        userId: item.user_id,
        loading: false,
        error: api.getErrorMessage(requestError, 'Не удалось загрузить пользователя'),
        saving: false
      });
    }
  };

  const updateModalForm = (event) => {
    const { name, value } = event.target;
    setModal((current) => ({
      ...current,
      error: '',
      form: { ...current.form, [name]: value }
    }));
  };

  const validateCreate = (form) => {
    if (!form.login.trim()) return 'Введите логин';
    if (form.password.trim().length < 8) return 'Пароль должен содержать не менее 8 символов';
    if (form.email && !form.email.includes('@')) return 'Введите корректный email';
    if (['student', 'teacher'].includes(form.role) && !form.full_name.trim()) return 'Введите ФИО';
    if (form.role === 'head' && !numberOrUndefined(form.lectern_id)) return 'Укажите ID кафедры';
    if (form.role === 'dean' && !numberOrUndefined(form.faculty_id)) return 'Укажите ID факультета';
    return '';
  };

  const createUser = async (event) => {
    event.preventDefault();
    const validationError = validateCreate(modal.form);
    if (validationError) {
      setModal((current) => ({ ...current, error: validationError }));
      return;
    }
    const form = modal.form;
    const payload = {
      login: form.login.trim(),
      password: form.password.trim(),
      role: form.role,
      email: form.email.trim(),
      full_name: form.full_name.trim(),
      group_id: numberOrUndefined(form.group_id),
      lectern_id: numberOrUndefined(form.lectern_id),
      faculty_id: numberOrUndefined(form.faculty_id),
      job_title: form.job_title.trim()
    };
    Object.keys(payload).forEach((key) => payload[key] === undefined && delete payload[key]);
    setModal((current) => ({ ...current, saving: true, error: '' }));
    try {
      await api.createAdminUser(token, payload);
      setModal(null);
      setNotice('Пользователь создан');
      setPage(1);
      setReloadKey((value) => value + 1);
    } catch (requestError) {
      setModal((current) => ({
        ...current,
        saving: false,
        error: api.getErrorMessage(requestError, 'Не удалось создать пользователя')
      }));
    }
  };

  const createTeacherInvite = async (event) => {
    event.preventDefault();
    const form = modal.form;
    if (!form.full_name.trim()) {
      setModal((current) => ({ ...current, error: 'Укажите ФИО преподавателя' }));
      return;
    }
    const payload = {
      full_name: form.full_name.trim(),
      lectern_id: numberOrUndefined(form.lectern_id) || 0,
      job_title: form.job_title.trim(),
      custom_code: form.custom_code.trim()
    };
    setModal((current) => ({ ...current, saving: true, error: '' }));
    try {
      const res = await api.createTeacherInvite(token, payload);
      setModal(null);
      const code = res.invite_code;
      copyInviteLink(code);
      setNotice(`Инвайт-код ${code} успешно создан! Ссылка скопирована в буфер обмена.`);
      setReloadKey((value) => value + 1);
    } catch (requestError) {
      setModal((current) => ({
        ...current,
        saving: false,
        error: api.getErrorMessage(requestError, 'Не удалось создать инвайт')
      }));
    }
  };

  const revokeInvite = async (inviteId, code) => {
    try {
      await api.revokeAdminInvite(token, inviteId);
      setNotice(`Инвайт-код ${code} отозван`);
      setReloadKey((value) => value + 1);
    } catch (requestError) {
      setError(api.getErrorMessage(requestError, 'Не удалось отозвать инвайт'));
    }
  };

  const updateUser = async (event) => {
    event.preventDefault();
    const { form, original, userId } = modal;
    const payload = {};
    if (!form.login.trim()) {
      setModal((current) => ({ ...current, error: 'Введите логин' }));
      return;
    }
    if (form.email && !form.email.includes('@')) {
      setModal((current) => ({ ...current, error: 'Введите корректный email' }));
      return;
    }
    if (form.password && form.password.trim().length < 8) {
      setModal((current) => ({ ...current, error: 'Новый пароль должен содержать не менее 8 символов' }));
      return;
    }
    if (form.login.trim() !== original.login) payload.login = form.login.trim();
    if (form.email.trim() !== (original.email || '')) payload.email = form.email.trim();
    if (form.status !== original.status) payload.status = form.status;
    if (form.password.trim()) payload.password = form.password.trim();
    if (form.role !== original.role) {
      payload.role = form.role;
      const targetId = numberOrUndefined(form.target_id);
      if (['student', 'teacher', 'head', 'dean'].includes(form.role) && !targetId) {
        setModal((current) => ({ ...current, error: 'Укажите идентификатор профиля или области доступа' }));
        return;
      }
      const targetKey = {
        student: 'student_id',
        teacher: 'teacher_id',
        head: 'lectern_id',
        dean: 'faculty_id'
      }[form.role];
      if (targetKey) payload[targetKey] = targetId;
    }
    if (!Object.keys(payload).length) {
      setModal((current) => ({ ...current, error: 'Нет изменений для сохранения' }));
      return;
    }
    setModal((current) => ({ ...current, saving: true, error: '' }));
    try {
      await api.updateAdminUser(token, userId, payload);
      setModal(null);
      setNotice('Изменения сохранены');
      setReloadKey((value) => value + 1);
    } catch (requestError) {
      setModal((current) => ({
        ...current,
        saving: false,
        error: api.getErrorMessage(requestError, 'Не удалось обновить пользователя')
      }));
    }
  };

  const archiveUser = async () => {
    setModal((current) => ({ ...current, saving: true, error: '' }));
    try {
      await api.archiveAdminUser(token, modal.user.user_id);
      setModal(null);
      setNotice('Пользователь перемещён в архив');
      if (items.length === 1 && page > 1) setPage((value) => value - 1);
      else setReloadKey((value) => value + 1);
    } catch (requestError) {
      setModal((current) => ({
        ...current,
        saving: false,
        error: api.getErrorMessage(requestError, 'Не удалось архивировать пользователя')
      }));
    }
  };

  const isEditingSelf = modal?.type === 'edit' && Number(modal.userId) === currentUserId;

  return (
    <section className="admin-users-page">
      <header className="admin-users-header">
        <div>
          <span>Администрирование</span>
          <h1>{activeTab === 'users' ? 'Пользователи' : 'Инвайт-коды преподавателей'}</h1>
          <p>Управление доступом и приглашениями</p>
        </div>
        {activeTab === 'users' ? (
          <button type="button" className="admin-primary-button" onClick={openCreate}>
            <AdminIcon name="add" />
            <span>Создать пользователя</span>
          </button>
        ) : (
          <button type="button" className="admin-primary-button" onClick={openCreateInvite}>
            <AdminIcon name="key" />
            <span>Создать инвайт</span>
          </button>
        )}
      </header>

      {/* Top Tab Switcher */}
      <div style={{ display: 'flex', gap: '12px', marginBottom: '20px', borderBottom: '1px solid var(--border-color, #e2e8f0)', paddingBottom: '10px' }}>
        <button
          type="button"
          className={`admin-secondary-button ${activeTab === 'users' ? 'is-active' : ''}`}
          style={{ background: activeTab === 'users' ? 'var(--primary-color, #2563eb)' : 'transparent', color: activeTab === 'users' ? '#fff' : 'inherit' }}
          onClick={() => setActiveTab('users')}
        >
          <AdminIcon name="users" />
          <span> Учетные записи</span>
        </button>
        <button
          type="button"
          className={`admin-secondary-button ${activeTab === 'invites' ? 'is-active' : ''}`}
          style={{ background: activeTab === 'invites' ? 'var(--primary-color, #2563eb)' : 'transparent', color: activeTab === 'invites' ? '#fff' : 'inherit' }}
          onClick={() => setActiveTab('invites')}
        >
          <AdminIcon name="key" />
          <span> Инвайт-коды преподавателей</span>
        </button>
      </div>

      {notice && <div className="admin-notice is-success" role="status">{notice}</div>}
      {error && <div className="admin-notice is-error" role="alert">{error}</div>}

      {/* TAB 1: USERS */}
      {activeTab === 'users' && (
        <>
          <div className="admin-users-strip" aria-label="Сводка текущей страницы">
            <span><strong>{pagination.total}</strong> всего</span>
            <span><strong>{pageStats.active}</strong> активных на странице</span>
            <span><strong>{pageStats.blocked}</strong> заблокировано</span>
            <span><strong>{pageStats.withTwoFa}</strong> используют 2FA</span>
          </div>

          <form className="admin-users-toolbar" onSubmit={submitSearch}>
            <label className="admin-search-field">
              <AdminIcon name="search" />
              <input
                type="search"
                value={searchDraft}
                onChange={(event) => setSearchDraft(event.target.value)}
                placeholder="Логин или email"
              />
            </label>
            <button type="submit" className="admin-secondary-button">Найти</button>
            <label className="admin-filter-field">
              <span>Роль</span>
              <select value={filters.role} onChange={(event) => changeFilter('role', event.target.value)}>
                <option value="">Все роли</option>
                {ROLE_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </label>
            <label className="admin-filter-field">
              <span>Статус</span>
              <select value={filters.status} onChange={(event) => changeFilter('status', event.target.value)}>
                <option value="">Все статусы</option>
                {STATUS_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </label>
            <button
              type="button"
              className="admin-icon-button"
              onClick={() => setReloadKey((value) => value + 1)}
              title="Обновить список"
              aria-label="Обновить список"
            >
              <AdminIcon name="refresh" />
            </button>
          </form>

          <div className={`admin-users-table-wrap ${loading ? 'is-loading' : ''}`}>
            <table className="admin-users-table">
              <thead>
                <tr>
                  <th>Пользователь</th>
                  <th>Роль</th>
                  <th>Email</th>
                  <th>Статус</th>
                  <th>2FA</th>
                  <th>Создан</th>
                  <th><span className="sr-only">Действия</span></th>
                </tr>
              </thead>
              <tbody>
                {!loading && items.map((item) => {
                  const isSelf = Number(item.user_id) === currentUserId;
                  return (
                    <tr key={item.user_id}>
                      <td data-label="Пользователь">
                        <strong>{item.login || '—'}</strong>
                        <small>#{item.user_id}{isSelf ? ' · вы' : ''}</small>
                      </td>
                      <td data-label="Роль"><span className={`admin-role-badge is-${item.role}`}>{roleLabel(item.role)}</span></td>
                      <td data-label="Email" className="admin-email-cell">{item.email || 'Не привязан'}</td>
                      <td data-label="Статус"><span className={`admin-status-badge is-${item.status}`}>{statusLabel(item.status)}</span></td>
                      <td data-label="2FA"><span className={`admin-twofa ${item.two_fa_enabled ? 'is-enabled' : ''}`}>{item.two_fa_enabled ? 'Включена' : 'Нет'}</span></td>
                      <td data-label="Создан">{formatDate(item.created_at)}</td>
                      <td data-label="Действия">
                        <div className="admin-row-actions">
                          <button type="button" onClick={() => openEdit(item)} title="Редактировать" aria-label={`Редактировать ${item.login}`}>
                            <AdminIcon name="edit" />
                          </button>
                          <button
                            type="button"
                            className="is-danger"
                            disabled={isSelf || item.status === 'archived'}
                            onClick={() => setModal({ type: 'archive', user: item, saving: false, error: '' })}
                            title={isSelf ? 'Нельзя архивировать собственную учетную запись' : 'Архивировать'}
                            aria-label={`Архивировать ${item.login}`}
                          >
                            <AdminIcon name="archive" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
                {!loading && !items.length && (
                  <tr className="admin-empty-row"><td colSpan={7}>Пользователи не найдены</td></tr>
                )}
                {loading && (
                  <tr className="admin-empty-row"><td colSpan={7}>Загрузка пользователей...</td></tr>
                )}
              </tbody>
            </table>
          </div>

          <footer className="admin-pagination">
            <span>{pageStart}–{pageEnd} из {pagination.total}</span>
            <label>
              <span>На странице</span>
              <select value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setPage(1); }}>
                <option value={10}>10</option>
                <option value={20}>20</option>
                <option value={50}>50</option>
              </select>
            </label>
            <div>
              <button
                type="button"
                disabled={page <= 1 || loading}
                onClick={() => setPage((value) => Math.max(1, value - 1))}
                aria-label="Предыдущая страница"
                title="Предыдущая страница"
              ><AdminIcon name="previous" /></button>
              <strong>{pagination.page || 1} / {Math.max(1, pagination.pages || 1)}</strong>
              <button
                type="button"
                disabled={!pagination.pages || page >= pagination.pages || loading}
                onClick={() => setPage((value) => value + 1)}
                aria-label="Следующая страница"
                title="Следующая страница"
              ><AdminIcon name="next" /></button>
            </div>
          </footer>
        </>
      )}

      {/* TAB 2: TEACHER INVITES */}
      {activeTab === 'invites' && (
        <>
          <div className="admin-users-strip" aria-label="Сводка инвайтов">
            <span><strong>{inviteStats.total}</strong> всего инвайтов</span>
            <span><strong style={{ color: '#d97706' }}>{inviteStats.pending}</strong> ожидают регистрации</span>
            <span><strong style={{ color: '#16a34a' }}>{inviteStats.registered}</strong> зарегистрировано</span>
          </div>

          <div className="admin-users-toolbar">
            <label className="admin-filter-field">
              <span>Статус регистрации</span>
              <select value={inviteStatusFilter} onChange={(event) => setInviteStatusFilter(event.target.value)}>
                <option value="">Все статусы (Все)</option>
                <option value="pending">Ожидают регистрации (Не зареган)</option>
                <option value="used">Зарегистрированы (Зареган)</option>
              </select>
            </label>
            <button
              type="button"
              className="admin-icon-button"
              onClick={() => setReloadKey((value) => value + 1)}
              title="Обновить список"
              aria-label="Обновить список"
            >
              <AdminIcon name="refresh" />
            </button>
          </div>

          <div className={`admin-users-table-wrap ${invitesLoading ? 'is-loading' : ''}`}>
            <table className="admin-users-table">
              <thead>
                <tr>
                  <th>Инвайт-код</th>
                  <th>Преподаватель</th>
                  <th>Статус</th>
                  <th>Дата создания</th>
                  <th>Дата регистрации</th>
                  <th>Учетная запись</th>
                  <th><span className="sr-only">Действия</span></th>
                </tr>
              </thead>
              <tbody>
                {!invitesLoading && invites.map((item) => {
                  const isUsed = !!item.used_at;
                  return (
                    <tr key={item.invite_id}>
                      <td data-label="Инвайт-код">
                        <strong style={{ fontFamily: 'monospace', fontSize: '1.05em', color: 'var(--primary-color, #2563eb)' }}>
                          {item.invite_code}
                        </strong>
                      </td>
                      <td data-label="Преподаватель">
                        <strong>{item.teacher_name || '—'}</strong>
                        {item.lectern_name && <small>Кафедра: {item.lectern_name}</small>}
                      </td>
                      <td data-label="Статус">
                        <span className={`admin-status-badge ${isUsed ? 'is-active' : 'is-blocked'}`}>
                          {isUsed ? 'Зарегистрирован' : 'Ожидает входа'}
                        </span>
                      </td>
                      <td data-label="Создан">{formatDateTime(item.created_at)}</td>
                      <td data-label="Зарегистрирован">{formatDateTime(item.used_at)}</td>
                      <td data-label="Логин">
                        {item.registered_as ? (
                          <strong>{item.registered_as}</strong>
                        ) : (
                          <span style={{ color: '#94a3b8' }}>—</span>
                        )}
                      </td>
                      <td data-label="Действия">
                        <div className="admin-row-actions">
                          <button
                            type="button"
                            onClick={() => copyInviteLink(item.invite_code)}
                            title="Скопировать ссылку регистрации"
                            aria-label={`Скопировать ссылку для ${item.invite_code}`}
                          >
                            <AdminIcon name="copy" />
                          </button>
                          {!isUsed && (
                            <button
                              type="button"
                              className="is-danger"
                              onClick={() => revokeInvite(item.invite_id, item.invite_code)}
                              title="Отозвать инвайт"
                              aria-label={`Отозвать инвайт ${item.invite_code}`}
                            >
                              <AdminIcon name="trash" />
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
                {!invitesLoading && !invites.length && (
                  <tr className="admin-empty-row"><td colSpan={7}>Инвайт-коды не найдены</td></tr>
                )}
                {invitesLoading && (
                  <tr className="admin-empty-row"><td colSpan={7}>Загрузка инвайт-кодов...</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </>
      )}

      {/* MODALS */}
      {modal && (
        <div className="admin-modal-backdrop" onMouseDown={(event) => event.target === event.currentTarget && !modal.saving && setModal(null)}>
          <section className={`admin-modal ${modal.type === 'archive' ? 'is-confirm' : ''}`} role="dialog" aria-modal="true">
            <header>
              <div>
                <span>
                  {modal.type === 'create' ? 'Новая учетная запись' :
                   modal.type === 'create_invite' ? 'Генерация инвайта' :
                   modal.type === 'edit' ? 'Настройки пользователя' : 'Подтверждение'}
                </span>
                <h2>
                  {modal.type === 'create' ? 'Создать пользователя' :
                   modal.type === 'create_invite' ? 'Новый инвайт преподавателя' :
                   modal.type === 'edit' ? 'Редактировать пользователя' : 'Архивировать пользователя?'}
                </h2>
              </div>
              <button type="button" onClick={() => setModal(null)} disabled={modal.saving} aria-label="Закрыть" title="Закрыть">
                <AdminIcon name="close" />
              </button>
            </header>

            {modal.type === 'create_invite' && (
              <form onSubmit={createTeacherInvite}>
                <div className="admin-form-grid">
                  <Field label="ФИО Преподавателя" required wide>
                    <input name="full_name" value={modal.form.full_name} onChange={updateModalForm} placeholder="Иванов Иван Иванович" autoComplete="name" />
                  </Field>
                  <Field label="ID кафедры">
                    <input name="lectern_id" value={modal.form.lectern_id} onChange={updateModalForm} inputMode="numeric" placeholder="Например: 1" />
                  </Field>
                  <Field label="Должность">
                    <input name="job_title" value={modal.form.job_title} onChange={updateModalForm} placeholder="Старший преподаватель" />
                  </Field>
                  <Field label="Кастомный код (опционально)" wide>
                    <input name="custom_code" value={modal.form.custom_code} onChange={updateModalForm} placeholder="Оставьте пустым для автокода (TCHR-XXXX)" />
                  </Field>
                </div>
                {modal.error && <div className="admin-form-error" role="alert">{modal.error}</div>}
                <footer>
                  <button type="button" className="admin-secondary-button" onClick={() => setModal(null)} disabled={modal.saving}>Отмена</button>
                  <button type="submit" className="admin-primary-button" disabled={modal.saving}>{modal.saving ? 'Создание...' : 'Сгенерировать инвайт'}</button>
                </footer>
              </form>
            )}

            {modal.type === 'create' && (
              <form onSubmit={createUser}>
                <div className="admin-form-grid">
                  <Field label="Логин" required>
                    <input name="login" value={modal.form.login} onChange={updateModalForm} autoComplete="off" />
                  </Field>
                  <Field label="Email">
                    <input name="email" type="email" value={modal.form.email} onChange={updateModalForm} autoComplete="off" />
                  </Field>
                  <Field label="Пароль" required>
                    <input name="password" type="password" value={modal.form.password} onChange={updateModalForm} autoComplete="new-password" />
                  </Field>
                  <Field label="Роль" required>
                    <select name="role" value={modal.form.role} onChange={updateModalForm}>
                      {ROLE_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                    </select>
                  </Field>
                  <RoleSpecificCreateFields form={modal.form} onChange={updateModalForm} />
                </div>
                {modal.error && <div className="admin-form-error" role="alert">{modal.error}</div>}
                <footer>
                  <button type="button" className="admin-secondary-button" onClick={() => setModal(null)} disabled={modal.saving}>Отмена</button>
                  <button type="submit" className="admin-primary-button" disabled={modal.saving}>{modal.saving ? 'Создание...' : 'Создать'}</button>
                </footer>
              </form>
            )}

            {modal.type === 'edit' && modal.loading && <div className="admin-modal-state">Загрузка...</div>}
            {modal.type === 'edit' && !modal.loading && !modal.form && <div className="admin-modal-state is-error">{modal.error}</div>}
            {modal.type === 'edit' && modal.form && (
              <form onSubmit={updateUser}>
                <div className="admin-form-grid">
                  <Field label="Логин" required>
                    <input name="login" value={modal.form.login} onChange={updateModalForm} />
                  </Field>
                  <Field label="Email">
                    <input name="email" type="email" value={modal.form.email} onChange={updateModalForm} />
                  </Field>
                  <Field label="Новый пароль">
                    <input name="password" type="password" value={modal.form.password} onChange={updateModalForm} autoComplete="new-password" />
                  </Field>
                  <Field label="Статус">
                    <select name="status" value={modal.form.status} onChange={updateModalForm} disabled={isEditingSelf}>
                      {STATUS_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                    </select>
                  </Field>
                  <Field label="Роль" wide>
                    <select name="role" value={modal.form.role} onChange={updateModalForm} disabled={isEditingSelf}>
                      {ROLE_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                    </select>
                  </Field>
                  {modal.form.role !== modal.original.role && (
                    <RoleTargetField role={modal.form.role} value={modal.form.target_id} onChange={updateModalForm} />
                  )}
                </div>
                {modal.error && <div className="admin-form-error" role="alert">{modal.error}</div>}
                <footer>
                  <button type="button" className="admin-secondary-button" onClick={() => setModal(null)} disabled={modal.saving}>Отмена</button>
                  <button type="submit" className="admin-primary-button" disabled={modal.saving}>{modal.saving ? 'Сохранение...' : 'Сохранить'}</button>
                </footer>
              </form>
            )}

            {modal.type === 'archive' && (
              <div className="admin-confirm-body">
                <p>Учетная запись <strong>{modal.user.login}</strong> получит статус «В архиве».</p>
                {modal.error && <div className="admin-form-error" role="alert">{modal.error}</div>}
                <footer>
                  <button type="button" className="admin-secondary-button" onClick={() => setModal(null)} disabled={modal.saving}>Отмена</button>
                  <button type="button" className="admin-danger-button" onClick={archiveUser} disabled={modal.saving}>{modal.saving ? 'Архивирование...' : 'Архивировать'}</button>
                </footer>
              </div>
            )}
          </section>
        </div>
      )}
    </section>
  );
};

export default AdminUsersPage;
