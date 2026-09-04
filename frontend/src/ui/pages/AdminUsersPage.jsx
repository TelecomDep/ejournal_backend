import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import api from '../../services/api';
import { SearchableSelect } from '../components/SearchableSelect';

const ROLE_OPTIONS = [
  { value: 'student', label: 'Студент' },
  { value: 'teacher', label: 'Преподаватель' },
  { value: 'secretary', label: 'Секретарь' },
  { value: 'head', label: 'Зав. кафедрой' },
  { value: 'program_creator', label: 'Руководитель программы' },
  { value: 'director', label: 'Директор института' },
  { value: 'dean', label: 'Декан' },
  { value: 'minister', label: 'Министр образования' },
  { value: 'admin', label: 'Администратор' }
];

const LECTERN_ROLES = ['head', 'secretary', 'program_creator'];
const FACULTY_ROLES = ['dean', 'director'];

const STATUS_OPTIONS = [
  { value: 'active', label: 'Активен' },
  { value: 'blocked', label: 'Заблокирован' },
  { value: 'archived', label: 'В архиве' }
];

const EMPTY_CREATE_FORM = {
  login: '',
  password: '',
  roles: ['student'],
  primary_role: 'student',
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
const rolesOf = (user) => (
  Array.isArray(user?.roles) && user.roles.length ? user.roles : [user?.role].filter(Boolean)
);
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
    refresh: (
      <>
        <path d="M19 8a7.5 7.5 0 1 0 .5 7" />
        <path d="M19 3v5h-5" />
      </>
    ),
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
  <div className={`admin-form-field ${wide ? 'is-wide' : ''}`}>
    <span>{label}{required ? ' *' : ''}</span>
    {children}
  </div>
);

/**
 * @param {{
 *   form: any,
 *   onChange: (event: any) => void,
 *   onSelectChange: (fieldName: string, fieldValue: any) => void,
 *   catalogs?: Record<string, any[]>
 * }} props
 */
const RoleSpecificCreateFields = ({ form, onChange, onSelectChange, catalogs = {} }) => {
  const roles = Array.isArray(form.roles) ? form.roles : [form.role].filter(Boolean);
  const needsFullName = roles.some((role) => ['student', 'teacher', 'head'].includes(role));
  const hasTeachingProfile = roles.some((role) => ['teacher', 'head'].includes(role));
  const hasLecternScope = roles.some((role) => LECTERN_ROLES.includes(role));
  const hasFacultyScope = roles.some((role) => FACULTY_ROLES.includes(role));

  return (
    <>
      {needsFullName && (
        <Field label="ФИО" required wide>
          <input name="full_name" value={form.full_name} onChange={onChange} autoComplete="name" placeholder="Иванов Иван Иванович" />
        </Field>
      )}
      {roles.includes('student') && (
        <Field label="Группа" wide>
          <SearchableSelect
            options={(catalogs.groups || []).map((g) => ({ id: g.id, name: g.name, sub: g.lectern_name ? `Кафедра: ${g.lectern_name}` : '' }))}
            value={form.group_id}
            onChange={(id) => onSelectChange('group_id', id)}
            placeholder="Выберите академическую группу..."
          />
        </Field>
      )}
      {(hasTeachingProfile || hasLecternScope) && (
        <Field label="Кафедра" required={hasLecternScope} wide>
          <SearchableSelect
            options={(catalogs.lecterns || []).map((l) => ({ id: l.id, name: l.name, sub: l.faculty_name ? `Факультет: ${l.faculty_name}` : '' }))}
            value={form.lectern_id}
            onChange={(id) => onSelectChange('lectern_id', id)}
            placeholder="Выберите кафедру..."
          />
        </Field>
      )}
      {hasTeachingProfile && (
        <Field label="Преподавательская должность">
          <input name="job_title" value={form.job_title} onChange={onChange} placeholder={roles.includes('head') ? 'Заведующий кафедрой' : 'Старший преподаватель'} />
        </Field>
      )}
      {hasFacultyScope && (
        <Field label="Факультет" required wide>
          <SearchableSelect
            options={(catalogs.faculties || []).map((f) => ({ id: f.id, name: f.name }))}
            value={form.faculty_id}
            onChange={(id) => onSelectChange('faculty_id', id)}
            placeholder="Выберите факультет..."
          />
        </Field>
      )}
    </>
  );
};

/**
 * @param {{
 *   role: string,
 *   fieldName?: string,
 *   value: any,
 *   onSelectChange: (fieldName: string, fieldValue: any) => void,
 *   catalogs?: Record<string, any[]>
 * }} props
 */
const RoleTargetField = ({ role, value, fieldName = 'target_id', onSelectChange, catalogs = {} }) => {
  if (role === 'student') {
    return (
      <Field label="Привязка студента" required wide>
        <SearchableSelect
          options={(catalogs.students || []).map((s) => ({ id: s.id, name: s.name, sub: s.group_name ? `Группа: ${s.group_name}` : '' }))}
          value={value}
          onChange={(id) => onSelectChange(fieldName, id)}
          placeholder="Поиск студента по ФИО или группе..."
        />
      </Field>
    );
  }
  if (role === 'teacher') {
    return (
      <Field label="Привязка преподавателя" required wide>
        <SearchableSelect
          options={(catalogs.teachers || []).map((t) => ({ id: t.id, name: t.name, sub: t.job_title || t.lectern_name }))}
          value={value}
          onChange={(id) => onSelectChange(fieldName, id)}
          placeholder="Поиск преподавателя по ФИО или кафедре..."
        />
      </Field>
    );
  }
  if (LECTERN_ROLES.includes(role)) {
    return (
      <Field label="Кафедра" required wide>
        <SearchableSelect
          options={(catalogs.lecterns || []).map((l) => ({ id: l.id, name: l.name, sub: l.faculty_name ? `Факультет: ${l.faculty_name}` : '' }))}
          value={value}
          onChange={(id) => onSelectChange(fieldName, id)}
          placeholder="Выберите кафедру..."
        />
      </Field>
    );
  }
  if (FACULTY_ROLES.includes(role)) {
    return (
      <Field label="Факультет" required wide>
        <SearchableSelect
          options={(catalogs.faculties || []).map((f) => ({ id: f.id, name: f.name }))}
          value={value}
          onChange={(id) => onSelectChange(fieldName, id)}
          placeholder="Выберите факультет..."
        />
      </Field>
    );
  }
  return null;
};

const AdminUsersPage = ({ token, currentUser }) => {
  const [activeTab, setActiveTab] = useState('users'); // 'users' | 'teacher_invites' | 'student_invites'
  
  // Users state
  const [items, setItems] = useState([]);
  const [pagination, setPagination] = useState({ page: 1, page_size: 20, total: 0, pages: 0 });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [searchDraft, setSearchDraft] = useState('');
  const [filters, setFilters] = useState({ search: '', role: '', status: '' });
  
  // Invites state
  const [invites, setInvites] = useState([]);
  const [inviteRoleFilter, setInviteRoleFilter] = useState(''); // '' (all) | 'teacher' | 'student'
  const [inviteStatusFilter, setInviteStatusFilter] = useState(''); // '' (all) | 'pending' | 'used'
  const [inviteLecternFilter, setInviteLecternFilter] = useState('');
  const [inviteGroupFilter, setInviteGroupFilter] = useState('');
  const [inviteSearch, setInviteSearch] = useState('');
  const [selectedInviteIds, setSelectedInviteIds] = useState(new Set());
  const [invitesLoading, setInvitesLoading] = useState(false);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [reloadKey, setReloadKey] = useState(0);
  const [modal, setModal] = useState(null);

  const [catalogs, setCatalogs] = useState({ groups: [], lecterns: [], faculties: [], teachers: [], students: [] });

  const loadCatalogs = useCallback(async () => {
    try {
      const data = await api.getAdminCatalogs(token);
      if (data) {
        setCatalogs({
          groups: Array.isArray(data.groups) ? data.groups : [],
          lecterns: Array.isArray(data.lecterns) ? data.lecterns : [],
          faculties: Array.isArray(data.faculties) ? data.faculties : [],
          teachers: Array.isArray(data.teachers) ? data.teachers : [],
          students: Array.isArray(data.students) ? data.students : []
        });
      }
    } catch (err) {
      console.error('Failed to load admin catalogs:', err);
    }
  }, [token]);

  useEffect(() => {
    loadCatalogs();
  }, [loadCatalogs]);

  const lecternOptions = useMemo(() => {
    const set = new Set();
    invites.forEach((inv) => {
      if (inv.lectern_name) set.add(inv.lectern_name);
    });
    return Array.from(set).sort();
  }, [invites]);

  const groupOptions = useMemo(() => {
    const set = new Set();
    invites.forEach((inv) => {
      if (inv.group_name) set.add(inv.group_name);
    });
    return Array.from(set).sort();
  }, [invites]);

  const filteredInvites = useMemo(() => {
    return invites.filter((inv) => {
      if (inviteRoleFilter && inv.role !== inviteRoleFilter) return false;
      if (inviteLecternFilter && inv.lectern_name !== inviteLecternFilter) return false;
      if (inviteGroupFilter && inv.group_name !== inviteGroupFilter) return false;
      if (inviteSearch.trim()) {
        const q = inviteSearch.trim().toLowerCase();
        const name = (inv.student_name || inv.teacher_name || '').toLowerCase();
        const lectern = (inv.lectern_name || '').toLowerCase();
        const group = (inv.group_name || '').toLowerCase();
        const code = (inv.invite_code || '').toLowerCase();
        const reg = (inv.registered_as || '').toLowerCase();
        if (!name.includes(q) && !lectern.includes(q) && !group.includes(q) && !code.includes(q) && !reg.includes(q)) {
          return false;
        }
      }
      return true;
    });
  }, [invites, inviteRoleFilter, inviteLecternFilter, inviteGroupFilter, inviteSearch]);

  const selectedInvitesList = useMemo(() => {
    return invites.filter((inv) => selectedInviteIds.has(inv.invite_id));
  }, [invites, selectedInviteIds]);

  const toggleSelectInvite = (id) => {
    setSelectedInviteIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleSelectAllInvites = () => {
    const visibleIds = filteredInvites.map((inv) => inv.invite_id);
    const allSelected = visibleIds.length > 0 && visibleIds.every((id) => selectedInviteIds.has(id));
    setSelectedInviteIds((prev) => {
      const next = new Set(prev);
      if (allSelected) {
        visibleIds.forEach((id) => next.delete(id));
      } else {
        visibleIds.forEach((id) => next.add(id));
      }
      return next;
    });
  };

  const selectAllFilteredInvites = () => {
    const visibleIds = filteredInvites.map((inv) => inv.invite_id);
    setSelectedInviteIds((prev) => {
      const next = new Set(prev);
      visibleIds.forEach((id) => next.add(id));
      return next;
    });
  };

  const clearInviteSelection = () => setSelectedInviteIds(new Set());

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
    if (activeTab !== 'teacher_invites' && activeTab !== 'student_invites') return;
    setInvitesLoading(true);
    setError('');
    try {
      const roleToFetch = activeTab === 'teacher_invites' ? 'teacher' : 'student';
      const list = await api.getAdminInvites(token, {
        role: roleToFetch,
        status: inviteStatusFilter
      });
      setInvites(Array.isArray(list) ? list : []);
    } catch (requestError) {
      setInvites([]);
      setError(api.getErrorMessage(requestError, 'Не удалось загрузить список инвайтов'));
    } finally {
      setInvitesLoading(false);
    }
  }, [activeTab, inviteStatusFilter, token]);

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

  const openCreateInvite = (roleOverride) => {
    const role = roleOverride || (activeTab === 'student_invites' ? 'student' : 'teacher');
    setModal({ type: 'create_invite', form: { ...EMPTY_INVITE_FORM, role }, error: '', saving: false });
  };

  const openEdit = async (item) => {
    setModal({ type: 'edit', userId: item.user_id, loading: true, error: '', saving: false });
    try {
      const user = await api.getAdminUser(token, item.user_id);
      const assignedRoles = rolesOf(user);
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
          roles: assignedRoles,
          primary_role: user.role || assignedRoles[0] || 'student',
          status: user.status || 'active',
          password: '',
          student_id: user.student_id || '',
          teacher_id: user.teacher_id || '',
          lectern_id: user.lectern_id || '',
          faculty_id: user.faculty_id || ''
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

  const updateModalFormField = (fieldName, fieldValue) => {
    setModal((current) => {
      if (!current || !current.form) return current;
      return {
        ...current,
        error: '',
        form: { ...current.form, [fieldName]: fieldValue }
      };
    });
  };

  const toggleModalRole = (role) => {
    setModal((current) => {
      if (!current?.form) return current;
      const currentRoles = Array.isArray(current.form.roles) ? current.form.roles : [];
      const nextRoles = currentRoles.includes(role)
        ? currentRoles.filter((item) => item !== role)
        : [...currentRoles, role];
      const primaryRole = nextRoles.includes(current.form.primary_role)
        ? current.form.primary_role
        : (nextRoles[0] || '');
      return {
        ...current,
        error: '',
        form: { ...current.form, roles: nextRoles, primary_role: primaryRole }
      };
    });
  };

  const validateCreate = (form) => {
    if (!form.login.trim()) return 'Введите логин';
    if (form.password.trim().length < 8) return 'Пароль должен содержать не менее 8 символов';
    if (form.email && !form.email.includes('@')) return 'Введите корректный email';
    if (!Array.isArray(form.roles) || !form.roles.length) return 'Назначьте пользователю хотя бы одну роль';
    if (!form.roles.includes(form.primary_role)) return 'Основная роль должна входить в список назначенных';
    if (form.roles.some((role) => ['student', 'teacher', 'head'].includes(role)) && !form.full_name.trim()) return 'Введите ФИО';
    if (form.roles.some((role) => LECTERN_ROLES.includes(role)) && !numberOrUndefined(form.lectern_id)) return 'Выберите кафедру';
    if (form.roles.some((role) => FACULTY_ROLES.includes(role)) && !numberOrUndefined(form.faculty_id)) return 'Выберите факультет';
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
      roles: form.roles,
      primary_role: form.primary_role,
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
  const handleCreate = createUser;

  const handleCreateInvite = async (event) => {
    event.preventDefault();
    const form = modal.form;
    if (!form.full_name?.trim()) {
      setModal((current) => ({ ...current, error: 'Укажите ФИО' }));
      return;
    }
    setModal((current) => ({ ...current, saving: true, error: '' }));
    try {
      let res;
      if (form.role === 'student') {
        const payload = {
          full_name: form.full_name.trim(),
          group_id: numberOrUndefined(form.group_id) || 0,
          custom_code: (form.custom_code || '').trim()
        };
        res = await api.createStudentInvite(token, payload);
      } else {
        const payload = {
          full_name: form.full_name.trim(),
          lectern_id: numberOrUndefined(form.lectern_id) || 0,
          job_title: (form.job_title || '').trim(),
          custom_code: (form.custom_code || '').trim()
        };
        res = await api.createTeacherInvite(token, payload);
      }
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
    if (!Array.isArray(form.roles) || !form.roles.length) {
      setModal((current) => ({ ...current, error: 'Назначьте пользователю хотя бы одну роль' }));
      return;
    }
    if (!form.roles.includes(form.primary_role)) {
      setModal((current) => ({ ...current, error: 'Основная роль должна входить в список назначенных' }));
      return;
    }
    if (form.login.trim() !== original.login) payload.login = form.login.trim();
    if (form.email.trim() !== (original.email || '')) payload.email = form.email.trim();
    if (form.status !== original.status) payload.status = form.status;
    if (form.password.trim()) payload.password = form.password.trim();
    const originalRoles = rolesOf(original);
    const rolesChanged = form.roles.length !== originalRoles.length ||
      form.roles.some((role) => !originalRoles.includes(role));
    if (rolesChanged) payload.roles = form.roles;
    if (form.primary_role !== original.role) payload.primary_role = form.primary_role;

    const addedRoles = form.roles.filter((role) => !originalRoles.includes(role));
    if (addedRoles.includes('student') && !numberOrUndefined(form.student_id)) {
      setModal((current) => ({ ...current, error: 'Выберите профиль студента для новой роли' }));
      return;
    }
    if (addedRoles.includes('teacher') && !numberOrUndefined(form.teacher_id)) {
      setModal((current) => ({ ...current, error: 'Выберите профиль преподавателя для новой роли' }));
      return;
    }
    if (addedRoles.some((role) => LECTERN_ROLES.includes(role)) && !numberOrUndefined(form.lectern_id)) {
      setModal((current) => ({ ...current, error: 'Выберите кафедру для новой роли' }));
      return;
    }
    if (addedRoles.some((role) => FACULTY_ROLES.includes(role)) && !numberOrUndefined(form.faculty_id)) {
      setModal((current) => ({ ...current, error: 'Выберите факультет для новой роли' }));
      return;
    }
    if (addedRoles.includes('student') || Number(form.student_id || 0) !== Number(original.student_id || 0)) {
      if (numberOrUndefined(form.student_id)) payload.student_id = numberOrUndefined(form.student_id);
    }
    if (addedRoles.includes('teacher') || Number(form.teacher_id || 0) !== Number(original.teacher_id || 0)) {
      if (numberOrUndefined(form.teacher_id)) payload.teacher_id = numberOrUndefined(form.teacher_id);
    }
    if (addedRoles.some((role) => LECTERN_ROLES.includes(role)) || Number(form.lectern_id || 0) !== Number(original.lectern_id || 0)) {
      if (numberOrUndefined(form.lectern_id)) payload.lectern_id = numberOrUndefined(form.lectern_id);
    }
    if (addedRoles.some((role) => FACULTY_ROLES.includes(role)) || Number(form.faculty_id || 0) !== Number(original.faculty_id || 0)) {
      if (numberOrUndefined(form.faculty_id)) payload.faculty_id = numberOrUndefined(form.faculty_id);
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

  const handleSaveEdit = updateUser;

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
          <h1>
            {activeTab === 'users' && 'Учетные записи пользователей'}
            {activeTab === 'teacher_invites' && 'Инвайт-коды преподавателей'}
            {activeTab === 'student_invites' && 'Инвайт-коды студентов'}
          </h1>
          <p>Управление доступом и приглашениями</p>
        </div>
        {activeTab === 'users' && (
          <button type="button" className="admin-primary-button" onClick={openCreate}>
            <AdminIcon name="add" />
            <span>Создать пользователя</span>
          </button>
        )}
        {activeTab === 'teacher_invites' && (
          <button type="button" className="admin-primary-button" onClick={() => openCreateInvite('teacher')}>
            <AdminIcon name="key" />
            <span>Создать инвайт преподавателя</span>
          </button>
        )}
        {activeTab === 'student_invites' && (
          <button type="button" className="admin-primary-button" onClick={() => openCreateInvite('student')}>
            <AdminIcon name="key" />
            <span>Создать инвайт студента</span>
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
          className={`admin-secondary-button ${activeTab === 'teacher_invites' ? 'is-active' : ''}`}
          style={{ background: activeTab === 'teacher_invites' ? 'var(--primary-color, #2563eb)' : 'transparent', color: activeTab === 'teacher_invites' ? '#fff' : 'inherit' }}
          onClick={() => {
            setActiveTab('teacher_invites');
            setInviteSearch('');
            setInviteLecternFilter('');
            setInviteGroupFilter('');
          }}
        >
          <AdminIcon name="key" />
          <span> Инвайты преподавателей</span>
        </button>
        <button
          type="button"
          className={`admin-secondary-button ${activeTab === 'student_invites' ? 'is-active' : ''}`}
          style={{ background: activeTab === 'student_invites' ? 'var(--primary-color, #2563eb)' : 'transparent', color: activeTab === 'student_invites' ? '#fff' : 'inherit' }}
          onClick={() => {
            setActiveTab('student_invites');
            setInviteSearch('');
            setInviteLecternFilter('');
            setInviteGroupFilter('');
          }}
        >
          <AdminIcon name="key" />
          <span> Инвайты студентов</span>
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
              className="semester-icon-button ui-refresh-button"
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
                  <th>Роли</th>
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
                      <td data-label="Роли">
                        <div className="admin-role-list">
                          {rolesOf(item).map((role) => (
                            <span key={role} className={`admin-role-badge is-${role}`}>
                              {roleLabel(role)}{role === item.role ? ' · осн.' : ''}
                            </span>
                          ))}
                        </div>
                      </td>
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
            {/* TAB 2: TEACHER INVITES */}
      {activeTab === 'teacher_invites' && (
        <>
          <div className="admin-users-strip" aria-label="Сводка инвайтов преподавателей">
            <span><strong>{inviteStats.total}</strong> инвайтов преподавателей</span>
            <span><strong style={{ color: '#d97706' }}>{inviteStats.pending}</strong> ожидают регистрации</span>
            <span><strong style={{ color: '#16a34a' }}>{inviteStats.registered}</strong> зарегистрировано</span>
          </div>

          <div className="admin-users-toolbar">
            <label className="admin-search-field" style={{ flex: '1 1 200px' }}>
              <AdminIcon name="search" />
              <input
                type="search"
                value={inviteSearch}
                onChange={(event) => setInviteSearch(event.target.value)}
                placeholder="Поиск по ФИО преподавателя, кафедре или коду..."
              />
            </label>
            {lecternOptions.length > 0 && (
              <label className="admin-filter-field">
                <span>Кафедра</span>
                <select value={inviteLecternFilter} onChange={(event) => setInviteLecternFilter(event.target.value)}>
                  <option value="">Все кафедры ({lecternOptions.length})</option>
                  {lecternOptions.map((lName) => (
                    <option key={lName} value={lName}>{lName}</option>
                  ))}
                </select>
              </label>
            )}
            <label className="admin-filter-field">
              <span>Статус</span>
              <select value={inviteStatusFilter} onChange={(event) => setInviteStatusFilter(event.target.value)}>
                <option value="">Все статусы</option>
                <option value="pending">Ожидают регистрации</option>
                <option value="used">Зарегистрированы</option>
              </select>
            </label>
            <button
              type="button"
              className="semester-icon-button ui-refresh-button"
              onClick={() => setReloadKey((value) => value + 1)}
              title="Обновить список"
              aria-label="Обновить список"
            >
              <AdminIcon name="refresh" />
            </button>
          </div>

          <div className="no-print" style={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: '10px', background: '#eff6ff', border: '1px solid #bfdbfe', padding: '10px 16px', borderRadius: '8px', marginBottom: '16px' }}>
            <span style={{ color: '#1e40af', fontSize: '0.95em' }}>Выбрано карт для печати: <strong>{selectedInviteIds.size}</strong> шт.</span>
            <button type="button" className="admin-secondary-button" style={{ padding: '4px 10px', fontSize: '0.85em' }} onClick={selectAllFilteredInvites}>
              Выбрать все по фильтру ({filteredInvites.length})
            </button>
            {selectedInviteIds.size > 0 && (
              <button type="button" className="admin-secondary-button" style={{ padding: '4px 10px', fontSize: '0.85em' }} onClick={clearInviteSelection}>
                Снять выбор
              </button>
            )}
            {selectedInviteIds.size > 0 && (
              <button
                type="button"
                className="admin-primary-button"
                style={{ padding: '6px 14px', marginLeft: 'auto', background: '#16a34a', borderColor: '#16a34a' }}
                onClick={() => setModal({ type: 'print_invites' })}
              >
                🖨️ Печать выбранных ({selectedInviteIds.size})
              </button>
            )}
          </div>

          <div className={`admin-users-table-wrap ${invitesLoading ? 'is-loading' : ''}`}>
            <table className="admin-users-table">
              <thead>
                <tr>
                  <th style={{ width: '40px', textAlign: 'center' }}>
                    <input
                      type="checkbox"
                      checked={filteredInvites.length > 0 && filteredInvites.every((inv) => selectedInviteIds.has(inv.invite_id))}
                      onChange={toggleSelectAllInvites}
                      title="Выбрать все в текущей таблице"
                    />
                  </th>
                  <th>Инвайт-код</th>
                  <th>Преподаватель (ФИО)</th>
                  <th>Кафедра</th>
                  <th>Статус</th>
                  <th>Создан / Зарегистрирован</th>
                  <th>Учетная запись</th>
                  <th><span className="sr-only">Действия</span></th>
                </tr>
              </thead>
              <tbody>
                {!invitesLoading && filteredInvites.map((item) => {
                  const isUsed = !!item.used_at;
                  const isSelected = selectedInviteIds.has(item.invite_id);
                  return (
                    <tr key={item.invite_id} style={{ background: isSelected ? '#f0f9ff' : undefined }}>
                      <td style={{ textAlign: 'center' }}>
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => toggleSelectInvite(item.invite_id)}
                        />
                      </td>
                      <td data-label="Инвайт-код">
                        <strong style={{ fontFamily: 'monospace', fontSize: '1.05em', color: 'var(--primary-color, #2563eb)' }}>
                          {item.invite_code}
                        </strong>
                      </td>
                      <td data-label="Преподаватель">
                        <strong>{item.teacher_name || '—'}</strong>
                      </td>
                      <td data-label="Кафедра">
                        <span style={{ fontSize: '0.9em', color: '#475569' }}>
                          {item.lectern_name ? `Каф.: ${item.lectern_name}` : '—'}
                        </span>
                      </td>
                      <td data-label="Статус">
                        <span className={`admin-status-badge ${isUsed ? 'is-active' : 'is-blocked'}`}>
                          {isUsed ? 'Зарегистрирован' : 'Ожидает входа'}
                        </span>
                      </td>
                      <td data-label="Даты">
                        <div><small>Создан: {formatDateTime(item.created_at)}</small></div>
                        {isUsed && <div><small style={{ color: '#16a34a' }}>Зарег.: {formatDateTime(item.used_at)}</small></div>}
                      </td>
                      <td data-label="Логин">
                        {item.registered_as ? <strong>{item.registered_as}</strong> : <span style={{ color: '#94a3b8' }}>—</span>}
                      </td>
                      <td data-label="Действия">
                        <div className="admin-row-actions">
                          <button
                            type="button"
                            onClick={() => copyInviteLink(item.invite_code)}
                            title="Скопировать ссылку регистрации"
                          >
                            <AdminIcon name="copy" />
                          </button>
                          {!isUsed && (
                            <button
                              type="button"
                              className="is-danger"
                              onClick={() => revokeInvite(item.invite_id, item.invite_code)}
                              title="Отозвать инвайт"
                            >
                              <AdminIcon name="trash" />
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
                {!invitesLoading && !filteredInvites.length && (
                  <tr className="admin-empty-row"><td colSpan={8}>Инвайт-коды преподавателей не найдены</td></tr>
                )}
                {invitesLoading && (
                  <tr className="admin-empty-row"><td colSpan={8}>Загрузка...</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </>
      )}

      {/* TAB 3: STUDENT INVITES */}
      {activeTab === 'student_invites' && (
        <>
          <div className="admin-users-strip" aria-label="Сводка инвайтов студентов">
            <span><strong>{inviteStats.total}</strong> инвайтов студентов</span>
            <span><strong style={{ color: '#d97706' }}>{inviteStats.pending}</strong> ожидают регистрации</span>
            <span><strong style={{ color: '#16a34a' }}>{inviteStats.registered}</strong> зарегистрировано</span>
          </div>

          <div className="admin-users-toolbar">
            <label className="admin-search-field" style={{ flex: '1 1 200px' }}>
              <AdminIcon name="search" />
              <input
                type="search"
                value={inviteSearch}
                onChange={(event) => setInviteSearch(event.target.value)}
                placeholder="Поиск по ФИО студента, группе или коду..."
              />
            </label>
            {groupOptions.length > 0 && (
              <label className="admin-filter-field">
                <span>Группа</span>
                <select value={inviteGroupFilter} onChange={(event) => setInviteGroupFilter(event.target.value)}>
                  <option value="">Все группы ({groupOptions.length})</option>
                  {groupOptions.map((gName) => (
                    <option key={gName} value={gName}>{gName}</option>
                  ))}
                </select>
              </label>
            )}
            <label className="admin-filter-field">
              <span>Статус</span>
              <select value={inviteStatusFilter} onChange={(event) => setInviteStatusFilter(event.target.value)}>
                <option value="">Все статусы</option>
                <option value="pending">Ожидают регистрации</option>
                <option value="used">Зарегистрированы</option>
              </select>
            </label>
            <button
              type="button"
              className="semester-icon-button ui-refresh-button"
              onClick={() => setReloadKey((value) => value + 1)}
              title="Обновить список"
              aria-label="Обновить список"
            >
              <AdminIcon name="refresh" />
            </button>
          </div>

          <div className="no-print" style={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: '10px', background: '#eff6ff', border: '1px solid #bfdbfe', padding: '10px 16px', borderRadius: '8px', marginBottom: '16px' }}>
            <span style={{ color: '#1e40af', fontSize: '0.95em' }}>Выбрано карт для печати: <strong>{selectedInviteIds.size}</strong> шт.</span>
            <button type="button" className="admin-secondary-button" style={{ padding: '4px 10px', fontSize: '0.85em' }} onClick={selectAllFilteredInvites}>
              Выбрать все {inviteGroupFilter ? `в гр. ${inviteGroupFilter}` : 'по фильтру'} ({filteredInvites.length})
            </button>
            {selectedInviteIds.size > 0 && (
              <button type="button" className="admin-secondary-button" style={{ padding: '4px 10px', fontSize: '0.85em' }} onClick={clearInviteSelection}>
                Снять выбор
              </button>
            )}
            {selectedInviteIds.size > 0 && (
              <button
                type="button"
                className="admin-primary-button"
                style={{ padding: '6px 14px', marginLeft: 'auto', background: '#16a34a', borderColor: '#16a34a' }}
                onClick={() => setModal({ type: 'print_invites' })}
              >
                🖨️ Печать выбранных ({selectedInviteIds.size})
              </button>
            )}
          </div>

          <div className={`admin-users-table-wrap ${invitesLoading ? 'is-loading' : ''}`}>
            <table className="admin-users-table">
              <thead>
                <tr>
                  <th style={{ width: '40px', textAlign: 'center' }}>
                    <input
                      type="checkbox"
                      checked={filteredInvites.length > 0 && filteredInvites.every((inv) => selectedInviteIds.has(inv.invite_id))}
                      onChange={toggleSelectAllInvites}
                      title="Выбрать все в текущей таблице"
                    />
                  </th>
                  <th>Инвайт-код</th>
                  <th>Студент (ФИО)</th>
                  <th>Группа</th>
                  <th>Статус</th>
                  <th>Создан / Зарегистрирован</th>
                  <th>Учетная запись</th>
                  <th><span className="sr-only">Действия</span></th>
                </tr>
              </thead>
              <tbody>
                {!invitesLoading && filteredInvites.map((item) => {
                  const isUsed = !!item.used_at;
                  const isSelected = selectedInviteIds.has(item.invite_id);
                  return (
                    <tr key={item.invite_id} style={{ background: isSelected ? '#f0f9ff' : undefined }}>
                      <td style={{ textAlign: 'center' }}>
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => toggleSelectInvite(item.invite_id)}
                        />
                      </td>
                      <td data-label="Инвайт-код">
                        <strong style={{ fontFamily: 'monospace', fontSize: '1.05em', color: 'var(--primary-color, #2563eb)' }}>
                          {item.invite_code}
                        </strong>
                      </td>
                      <td data-label="Студент">
                        <strong>{item.student_name || '—'}</strong>
                      </td>
                      <td data-label="Группа">
                        <span style={{ fontSize: '0.9em', color: '#1e40af', fontWeight: 600 }}>
                          {item.group_name ? `ГР. ${item.group_name}` : '—'}
                        </span>
                      </td>
                      <td data-label="Статус">
                        <span className={`admin-status-badge ${isUsed ? 'is-active' : 'is-blocked'}`}>
                          {isUsed ? 'Зарегистрирован' : 'Ожидает входа'}
                        </span>
                      </td>
                      <td data-label="Даты">
                        <div><small>Создан: {formatDateTime(item.created_at)}</small></div>
                        {isUsed && <div><small style={{ color: '#16a34a' }}>Зарег.: {formatDateTime(item.used_at)}</small></div>}
                      </td>
                      <td data-label="Логин">
                        {item.registered_as ? <strong>{item.registered_as}</strong> : <span style={{ color: '#94a3b8' }}>—</span>}
                      </td>
                      <td data-label="Действия">
                        <div className="admin-row-actions">
                          <button
                            type="button"
                            onClick={() => copyInviteLink(item.invite_code)}
                            title="Скопировать ссылку регистрации"
                          >
                            <AdminIcon name="copy" />
                          </button>
                          {!isUsed && (
                            <button
                              type="button"
                              className="is-danger"
                              onClick={() => revokeInvite(item.invite_id, item.invite_code)}
                              title="Отозвать инвайт"
                            >
                              <AdminIcon name="trash" />
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
                {!invitesLoading && !filteredInvites.length && (
                  <tr className="admin-empty-row"><td colSpan={8}>Инвайт-коды студентов не найдены</td></tr>
                )}
                {invitesLoading && (
                  <tr className="admin-empty-row"><td colSpan={8}>Загрузка...</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </>
      )}

      {modal && (
        <div className={`admin-modal-backdrop ${modal.type === 'print_invites' ? 'is-print-backdrop' : ''}`} onMouseDown={(event) => event.target === event.currentTarget && !modal.saving && setModal(null)}>
          <section className={`admin-modal ${modal.type === 'archive' ? 'is-confirm' : modal.type === 'print_invites' ? 'is-print-preview' : ''}`} role="dialog" aria-modal="true">
            <header className="no-print">
              <div>
                <h2>
                  {modal.type === 'create' ? 'Новая учетная запись' :
                   modal.type === 'create_invite' ? 'Генерация инвайта' :
                   modal.type === 'edit' ? 'Настройки пользователя' :
                   modal.type === 'print_invites' ? 'Печать инвайт-карт' : 'Подтверждение'}
                </h2>
                <p>
                  {modal.type === 'create' ? 'Создать пользователя' :
                   modal.type === 'create_invite' ? `Новый инвайт (${modal.form?.role === 'student' ? 'Студент' : 'Преподаватель'})` :
                   modal.type === 'edit' ? 'Редактировать пользователя' :
                   modal.type === 'print_invites' ? `Готово к печати: ${selectedInvitesList.length} карт` : 'Архивировать пользователя?'}
                </p>
              </div>
              <button type="button" onClick={() => setModal(null)} disabled={modal.saving} aria-label="Закрыть" title="Закрыть">
                <AdminIcon name="close" />
              </button>
            </header>

            {modal.type === 'create_invite' && (
              <form onSubmit={handleCreateInvite}>
                <div className="admin-form-grid">
                  {modal.form.role === 'teacher' ? (
                    <>
                      <label className="admin-form-field admin-form-field--full">
                        <span>ФИО Преподавателя *</span>
                        <input name="full_name" value={modal.form.full_name} onChange={updateModalForm} placeholder="Иванов Иван Иванович" autoComplete="name" />
                      </label>
                      <Field label="Кафедра" required wide>
                        <SearchableSelect
                          options={(catalogs.lecterns || []).map((l) => ({ id: l.id, name: l.name, sub: l.faculty_name ? `Факультет: ${l.faculty_name}` : '' }))}
                          value={modal.form.lectern_id}
                          onChange={(id) => updateModalFormField('lectern_id', id)}
                          placeholder="Выберите кафедру из списка..."
                        />
                      </Field>
                      <label className="admin-form-field admin-form-field--full">
                        <span>Должность</span>
                        <input name="job_title" value={modal.form.job_title} onChange={updateModalForm} placeholder="Старший преподаватель" />
                      </label>
                    </>
                  ) : (
                    <>
                      <label className="admin-form-field admin-form-field--full">
                        <span>ФИО Студента *</span>
                        <input name="full_name" value={modal.form.full_name} onChange={updateModalForm} placeholder="Петров Петр Петрович" autoComplete="name" />
                      </label>
                      <Field label="Группа" required wide>
                        <SearchableSelect
                          options={(catalogs.groups || []).map((g) => ({ id: g.id, name: g.name, sub: g.lectern_name ? `Кафедра: ${g.lectern_name}` : '' }))}
                          value={modal.form.group_id}
                          onChange={(id) => updateModalFormField('group_id', id)}
                          placeholder="Выберите группу из списка..."
                        />
                      </Field>
                    </>
                  )}
                  <label className="admin-form-field admin-form-field--full">
                    <span>Собственный инвайт-код (необязательно)</span>
                    <input name="custom_code" value={modal.form.custom_code} onChange={updateModalForm} placeholder="Оставьте пустым для генерации 16-значного кода" />
                  </label>
                </div>
                {modal.error && <div className="admin-form-error" role="alert">{modal.error}</div>}
                <footer>
                  <button type="button" className="admin-secondary-button" onClick={() => setModal(null)} disabled={modal.saving}>Отмена</button>
                  <button type="submit" className="admin-primary-button" disabled={modal.saving}>{modal.saving ? 'Создание...' : 'Сгенерировать инвайт'}</button>
                </footer>
              </form>
            )}

            {modal.type === 'create' && (
              <form onSubmit={handleCreate}>
                <div className="admin-form-grid">
                  <label className="admin-form-field">
                    <span>Логин *</span>
                    <input name="login" value={modal.form.login} onChange={updateModalForm} autoComplete="off" />
                  </label>
                  <label className="admin-form-field">
                    <span>Email *</span>
                    <input name="email" type="email" value={modal.form.email} onChange={updateModalForm} autoComplete="off" />
                  </label>
                  <label className="admin-form-field">
                    <span>Пароль *</span>
                    <input name="password" type="password" value={modal.form.password} onChange={updateModalForm} autoComplete="new-password" />
                  </label>
                  <fieldset className="admin-role-picker admin-form-field is-wide">
                    <legend>Назначенные роли *</legend>
                    <p>Выберите все режимы кабинета, которые будут доступны пользователю.</p>
                    <div className="admin-role-picker-grid">
                      {ROLE_OPTIONS.map((option) => (
                        <label key={option.value} className={modal.form.roles.includes(option.value) ? 'is-selected' : ''}>
                          <input
                            type="checkbox"
                            checked={modal.form.roles.includes(option.value)}
                            onChange={() => toggleModalRole(option.value)}
                          />
                          <span>{option.label}</span>
                        </label>
                      ))}
                    </div>
                  </fieldset>
                  <label className="admin-form-field is-wide">
                    <span>Основная роль *</span>
                    <select name="primary_role" value={modal.form.primary_role} onChange={updateModalForm} disabled={!modal.form.roles.length}>
                      {modal.form.roles.map((role) => <option key={role} value={role}>{roleLabel(role)}</option>)}
                    </select>
                  </label>
                  <RoleSpecificCreateFields form={modal.form} onChange={updateModalForm} onSelectChange={updateModalFormField} catalogs={catalogs} />
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
              <form onSubmit={handleSaveEdit}>
                <div className="admin-form-grid">
                  <label className="admin-form-field">
                    <span>Логин *</span>
                    <input name="login" value={modal.form.login} onChange={updateModalForm} />
                  </label>
                  <label className="admin-form-field">
                    <span>Email *</span>
                    <input name="email" type="email" value={modal.form.email} onChange={updateModalForm} />
                  </label>
                  <label className="admin-form-field">
                    <span>Новый пароль</span>
                    <input name="password" type="password" value={modal.form.password} onChange={updateModalForm} autoComplete="new-password" />
                  </label>
                  <label className="admin-form-field">
                    <span>Статус</span>
                    <select name="status" value={modal.form.status} onChange={updateModalForm} disabled={isEditingSelf}>
                      <option value="active">Активен</option>
                      <option value="archived">В архиве</option>
                    </select>
                  </label>
                  <fieldset className="admin-role-picker admin-form-field is-wide" disabled={isEditingSelf}>
                    <legend>Назначенные роли</legend>
                    <p>Можно выбрать несколько ролей. Пользователь сможет переключать режим кабинета.</p>
                    <div className="admin-role-picker-grid">
                      {ROLE_OPTIONS.map((option) => (
                        <label key={option.value} className={modal.form.roles.includes(option.value) ? 'is-selected' : ''}>
                          <input
                            type="checkbox"
                            checked={modal.form.roles.includes(option.value)}
                            onChange={() => toggleModalRole(option.value)}
                          />
                          <span>{option.label}</span>
                        </label>
                      ))}
                    </div>
                  </fieldset>
                  <label className="admin-form-field is-wide">
                    <span>Основная роль</span>
                    <select name="primary_role" value={modal.form.primary_role} onChange={updateModalForm} disabled={isEditingSelf || !modal.form.roles.length}>
                      {modal.form.roles.map((role) => <option key={role} value={role}>{roleLabel(role)}</option>)}
                    </select>
                  </label>
                  {modal.form.roles.includes('student') && (
                    <RoleTargetField role="student" fieldName="student_id" value={modal.form.student_id} onSelectChange={updateModalFormField} catalogs={catalogs} />
                  )}
                  {modal.form.roles.includes('teacher') && (
                    <RoleTargetField role="teacher" fieldName="teacher_id" value={modal.form.teacher_id} onSelectChange={updateModalFormField} catalogs={catalogs} />
                  )}
                  {modal.form.roles.some((role) => LECTERN_ROLES.includes(role)) && (
                    <RoleTargetField role="head" fieldName="lectern_id" value={modal.form.lectern_id} onSelectChange={updateModalFormField} catalogs={catalogs} />
                  )}
                  {modal.form.roles.some((role) => FACULTY_ROLES.includes(role)) && (
                    <RoleTargetField role="dean" fieldName="faculty_id" value={modal.form.faculty_id} onSelectChange={updateModalFormField} catalogs={catalogs} />
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

            {modal.type === 'print_invites' && (
              <div className="admin-modal--print-preview" style={{ padding: '4px' }}>
                <header className="no-print" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px', paddingBottom: '12px', borderBottom: '1px solid #e2e8f0' }}>
                  <div>
                    <h3 style={{ margin: 0, fontSize: '1.2em' }}>Печать карточек инвайтов ({selectedInvitesList.length} шт.)</h3>
                    <p style={{ margin: '2px 0 0', fontSize: '0.85em', color: '#64748b' }}>Форматирование адаптировано для листа A4 (3 колонки, пунктирная рамочка ✂️ для резки).</p>
                  </div>
                  <div style={{ display: 'flex', gap: '8px' }}>
                    <button type="button" className="admin-secondary-button" onClick={() => setModal(null)}>Отмена</button>
                    <button
                      type="button"
                      className="admin-primary-button"
                      style={{ background: '#16a34a', borderColor: '#16a34a' }}
                      onClick={() => window.print()}
                    >
                      🖨️ Распечатать
                    </button>
                  </div>
                </header>

                <div id="printable-invite-cards" className="printable-cards-grid">
                  {selectedInvitesList.map((inv) => {
                    const regUrl = `${window.location.origin}/#/register?code=${encodeURIComponent(inv.invite_code)}`;
                    const isStudent = inv.role === 'student' || !!inv.student_name || !!inv.group_name;
                    const displayName = isStudent ? (inv.student_name || inv.teacher_name || 'Студент') : (inv.teacher_name || 'Преподаватель');
                    const subtitle = isStudent
                      ? `ГРУППА: ${inv.group_name || '—'}`
                      : `Каф.: ${inv.lectern_name || '—'}`;
                    const badgeText = isStudent ? 'Студент' : 'Преподаватель';

                    return (
                      <div key={inv.invite_id} className="invite-cutout-card">
                        <div className="invite-card-cut-icon">✂️</div>
                        <div className="invite-card-header">
                          <span className="invite-card-logo">СибГУТИ</span>
                          <span className="invite-card-type">{badgeText}</span>
                        </div>
                        <div className="invite-card-body">
                          <div className="invite-card-info">
                            <div className="invite-card-name" title={displayName}>{displayName}</div>
                            <div className="invite-card-sub" title={subtitle} style={{ fontWeight: isStudent ? 700 : 500, color: isStudent ? '#1e40af' : '#475569' }}>
                              {subtitle}
                            </div>
                            <div className="invite-card-code-box">
                              <span className="invite-card-code-label">КОД ДОСТУПА:</span>
                              <span className="invite-card-code-value">{inv.invite_code}</span>
                            </div>
                            <div className="invite-card-instructions">
                              Вход: <strong>{window.location.host}/#/register</strong>
                            </div>
                          </div>
                          <div className="invite-card-qr">
                            <QRCodeSVG value={regUrl} size={40} level="M" />
                            <span className="invite-card-qr-label">Регистрация</span>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </section>
        </div>
      )}
    </section>
  );
};

export default AdminUsersPage;
