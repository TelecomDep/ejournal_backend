const ROLE_LABELS = {
  admin: 'Администратор',
  dean: 'Декан',
  head: 'Зав. кафедрой',
  teacher: 'Преподаватель',
  student: 'Студент'
};

const STUDENT_NAV = [
  { key: 'dashboard', label: 'Главная', title: 'Главная', route: '/dashboard', icon: '⌂' },
  { key: 'schedule', label: 'Расписание', title: 'Расписание', route: '/schedule', icon: '□' },
  { key: 'grades', label: 'Оценки', title: 'Оценки', route: '/grades', icon: '◇' },
  { key: 'attendance', label: 'Посещаемость', title: 'Посещаемость', route: '/attendance', icon: '✓' },
  { key: 'analytics', label: 'Аналитика', title: 'Аналитика', route: '/analytics', icon: '▥' },
  { key: 'profile', label: 'Профиль', title: 'Профиль студента', route: '/profile', icon: '○' }
];

const TEACHER_NAV = [
  { key: 'dashboard', label: 'Главная', title: 'Главная', route: '/dashboard', icon: '⌂' },
  { key: 'schedule', label: 'Расписание', title: 'Расписание', route: '/schedule', icon: '□' },
  { key: 'attendance', label: 'Посещаемость', title: 'Посещаемость группы', route: '/teacher/attendance', icon: '✓' },
  { key: 'stats', label: 'Аналитика', title: 'Статистика группы', route: '/teacher/stats', icon: '▥' },
  { key: 'grades', label: 'Оценки', title: 'Ведомость оценок', route: '/teacher/grades', icon: '◇' },
  { key: 'reports', label: 'Отчёты', title: 'Формирование отчётов', route: '/staff/reports', icon: '□' },
  { key: 'profile', label: 'Профиль', title: 'Профиль преподавателя', route: '/profile', icon: '○' }
];

const ADMIN_NAV = [
  { key: 'dashboard', label: 'Главная', title: 'Главная', route: '/dashboard', icon: '⌂' },
  { key: 'semesters', label: 'Семестры', title: 'Управление семестрами', route: '/admin/semesters', icon: '□' },
  { key: 'users', label: 'Пользователи', title: 'Управление пользователями', route: '/admin/users', icon: '○' },
  { key: 'notifications', label: 'Уведомления', title: 'Управление уведомлениями', route: '/admin/notifications', icon: '○' },
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

const STAFF_ROLES = ['admin', 'head', 'dean'];

export const getRoleLabel = (role) => ROLE_LABELS[role] || 'Пользователь';

export const getNavigationForRole = (role) => {
  if (role === 'admin') {
    return ADMIN_NAV;
  }
  if (STAFF_ROLES.includes(role)) {
    return STAFF_NAV;
  }
  if (role === 'teacher') {
    return TEACHER_NAV;
  }
  return STUDENT_NAV;
};
