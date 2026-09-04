const ROLE_LABELS = {
  admin: 'Администратор',
  dean: 'Декан',
  head: 'Зав. кафедрой',
  secretary: 'Секретарь',
  program_creator: 'Руководитель программы',
  director: 'Директор института',
  minister: 'Министр образования',
  teacher: 'Преподаватель',
  student: 'Студент'
};

const STUDENT_NAV = [
  { key: 'dashboard', label: 'Главная', title: 'Главная', route: '/dashboard', icon: '⌂' },
  { key: 'grades', label: 'Оценки', title: 'Оценки', route: '/grades', icon: '◇' },
  { key: 'attendance', label: 'Посещаемость', title: 'Посещаемость', route: '/attendance', icon: '✓' },
  { key: 'analytics', label: 'Аналитика', title: 'Аналитика', route: '/analytics', icon: '▥' },
  { key: 'profile', label: 'Профиль', title: 'Профиль студента', route: '/profile', icon: '○' }
];

const TEACHER_NAV = [
  { key: 'dashboard', label: 'Главная', title: 'Главная', route: '/dashboard', icon: '⌂' },
  { key: 'attendance', label: 'Посещаемость', title: 'Посещаемость группы', route: '/teacher/attendance', icon: '✓' },
  { key: 'stats', label: 'Аналитика', title: 'Статистика группы', route: '/teacher/stats', icon: '▥' },
  { key: 'grades', label: 'Оценки', title: 'Ведомость оценок', route: '/teacher/grades', icon: '◇' },
  { key: 'antifraud', label: 'Антифрод', title: 'Контроль нарушений', route: '/staff/antifraud', icon: '!' },
  { key: 'reports', label: 'Отчёты', title: 'Формирование отчётов', route: '/staff/reports', icon: '□' },
  { key: 'profile', label: 'Профиль', title: 'Профиль преподавателя', route: '/profile', icon: '○' }
];

const ADMIN_NAV = [
  { key: 'dashboard', label: 'Главная', title: 'Главная', route: '/dashboard', icon: '⌂' },
  { key: 'semesters', label: 'Семестры', title: 'Управление семестрами', route: '/admin/semesters', icon: '□' },
  { key: 'users', label: 'Пользователи', title: 'Управление пользователями', route: '/admin/users', icon: '○' },
  { key: 'notifications', label: 'Уведомления', title: 'Управление уведомлениями', route: '/admin/notifications', icon: '○' },
  { key: 'antifraud', label: 'Антифрод', title: 'Контроль нарушений', route: '/staff/antifraud', icon: '!' },
  { key: 'reports', label: 'Отчёты', title: 'Формирование отчётов', route: '/staff/reports', icon: '□' },
  { key: 'profile', label: 'Профиль', title: 'Профиль администратора', route: '/profile', icon: '○' }
];

const STAFF_NAV = [
  { key: 'dashboard', label: 'Главная', title: 'Главная', route: '/dashboard', icon: '⌂' },
  { key: 'overview', label: 'Сводка', title: 'Сводка по охвату', route: '/staff/overview', icon: '▥' },
  { key: 'analytics', label: 'Аналитика', title: 'Аналитика', route: '/staff/analytics', icon: '▥' },
  { key: 'reports', label: 'Отчёты', title: 'Формирование отчётов', route: '/staff/reports', icon: '□' },
  { key: 'profile', label: 'Профиль', title: 'Профиль', route: '/profile', icon: '○' }
];

const TEACHING_STAFF_NAV = [
  { key: 'dashboard', label: 'Главная', title: 'Главная', route: '/dashboard', icon: '⌂' },
  { key: 'attendance', label: 'Посещаемость', title: 'Посещаемость группы', route: '/teacher/attendance', icon: '✓' },
  { key: 'grades', label: 'Оценки', title: 'Ведомость оценок', route: '/teacher/grades', icon: '◇' },
  { key: 'stats', label: 'Статистика групп', title: 'Статистика группы', route: '/teacher/stats', icon: '▥' },
  { key: 'overview', label: 'Сводка', title: 'Сводка по охвату', route: '/staff/overview', icon: '▥' },
  { key: 'analytics', label: 'Аналитика кафедры', title: 'Аналитика', route: '/staff/analytics', icon: '▥' },
  { key: 'reports', label: 'Отчёты', title: 'Формирование отчётов', route: '/staff/reports', icon: '□' },
  { key: 'profile', label: 'Профиль', title: 'Профиль', route: '/profile', icon: '○' }
];

export const STAFF_ROLES = [
  'admin',
  'head',
  'dean',
  'secretary',
  'program_creator',
  'director',
  'minister'
];

export const getActiveRole = (user) => {
  const role = user?.active_role || user?.role || user?.primary_role;
  return typeof role === 'string' ? role.trim().toLowerCase() : '';
};

export const getRoleLabel = (role) => ROLE_LABELS[role] || 'Пользователь';

export const getNavigationForRole = (role) => {
  const normalizedRole = typeof role === 'string' ? role.trim().toLowerCase() : '';
  if (normalizedRole === 'admin') {
    return ADMIN_NAV;
  }
  if (normalizedRole === 'head') {
    return TEACHING_STAFF_NAV;
  }
  if (STAFF_ROLES.includes(normalizedRole)) {
    return STAFF_NAV;
  }
  if (normalizedRole === 'teacher') {
    return TEACHER_NAV;
  }
  return STUDENT_NAV;
};
