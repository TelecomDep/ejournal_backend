const ROLE_LABELS = {
  admin: 'Администратор',
  dean: 'Декан',
  head: 'Зав. кафедрой',
  teacher: 'Преподаватель',
  student: 'Студент'
};

const STUDENT_NAV = [
  { key: 'profile', label: 'Профиль', title: 'Профиль студента', route: '/profile', icon: 'П' },
  { key: 'attendance', label: 'Посещаемость', title: 'Посещаемость', route: '/attendance', icon: 'Н' },
  { key: 'grades', label: 'Оценки', title: 'Оценки', route: '/grades', icon: 'О' }
];

const TEACHER_NAV = [
  { key: 'profile', label: 'Профиль', title: 'Профиль преподавателя', route: '/profile', icon: 'П' },
  { key: 'attendance', label: 'Посещаемость', title: 'Посещаемость группы', route: '/teacher/attendance', icon: 'Н' },
  { key: 'stats', label: 'Статистика', title: 'Статистика группы', route: '/teacher/stats', icon: 'С' },
  { key: 'grades', label: 'Оценки', title: 'Ведомость оценок', route: '/teacher/grades', icon: 'О' }
];

const STAFF_NAV = [
  { key: 'profile', label: 'Профиль', title: 'Профиль', route: '/profile', icon: 'П' },
  { key: 'overview', label: 'Сводка', title: 'Сводка по охвату', route: '/staff/overview', icon: 'С' }
];

const STAFF_ROLES = ['admin', 'head', 'dean'];

export const getRoleLabel = (role) => ROLE_LABELS[role] || 'Пользователь';

export const getNavigationForRole = (role) => {
  if (STAFF_ROLES.includes(role)) {
    return STAFF_NAV;
  }
  if (role === 'teacher') {
    return TEACHER_NAV;
  }
  return STUDENT_NAV;
};
