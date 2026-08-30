import React from 'react';

const STAFF_ROLES = ['admin', 'head', 'dean'];

const getDisplayName = (user) => (
  user?.teacher_name || user?.name || user?.login || 'Пользователь'
);

const roleTitle = (role) => {
  if (role === 'teacher') return 'Рабочая панель преподавателя';
  if (role === 'admin') return 'Панель администратора';
  if (role === 'head') return 'Панель заведующего кафедрой';
  if (role === 'dean') return 'Панель декана';
  if (STAFF_ROLES.includes(role)) return 'Сводная панель';
  return 'Личный кабинет студента';
};

const DashboardIcon = ({ name }) => {
  const icons = {
    schedule: (
      <>
        <rect x="4" y="5.5" width="16" height="14.5" rx="2.5" />
        <path d="M8 3.5v4M16 3.5v4M4 10h16" />
      </>
    ),
    grades: (
      <>
        <path d="M3.5 9 12 5l8.5 4-8.5 4-8.5-4Z" />
        <path d="M6.5 10.6v4.1c1.7 1.7 3.5 2.5 5.5 2.5s3.8-.8 5.5-2.5v-4.1" />
      </>
    ),
    attendance: (
      <>
        <circle cx="12" cy="12" r="8.5" />
        <path d="m8.6 12.2 2.3 2.3 4.8-5" />
      </>
    ),
    analytics: (
      <>
        <path d="M5 19V9.5h3.2V19H5Z" />
        <path d="M10.4 19V5.5h3.2V19h-3.2Z" />
        <path d="M15.8 19v-7h3.2v7h-3.2Z" />
      </>
    ),
    profile: (
      <>
        <circle cx="12" cy="8" r="3.4" />
        <path d="M5.4 19c.9-3.5 3.1-5.3 6.6-5.3s5.7 1.8 6.6 5.3" />
      </>
    ),
    staff: (
      <>
        <path d="M7.5 9.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5Z" />
        <path d="M16.5 9.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5Z" />
        <path d="M4.5 19v-1.2c0-2.4 1.4-4.1 3.4-4.1s3.4 1.7 3.4 4.1V19" />
        <path d="M12.7 19v-1.2c0-2.4 1.4-4.1 3.4-4.1s3.4 1.7 3.4 4.1V19" />
      </>
    )
  };

  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      {icons[name] || icons.profile}
    </svg>
  );
};

const QuickCard = ({ icon, title, text, to, navigate }) => (
  <button type="button" className="dashboard-quick-card" onClick={() => navigate(to)}>
    <span className="dashboard-card-icon"><DashboardIcon name={icon} /></span>
    <span>
      <strong>{title}</strong>
      <small>{text}</small>
    </span>
  </button>
);

const getQuickCards = (role) => {
  if (role === 'teacher') {
    return [
      { icon: 'attendance', title: 'Открыть QR', text: 'Создать отметку посещаемости', to: '/teacher/attendance' },
      { icon: 'grades', title: 'Оценки', text: 'Работы и баллы студентов', to: '/teacher/grades' },
      { icon: 'analytics', title: 'Статистика группы', text: 'Посещаемость и успеваемость', to: '/teacher/stats' },
      { icon: 'attendance', title: 'Антифрод', text: 'Нарушения при отметке', to: '/staff/antifraud' }
    ];
  }

  if (role === 'admin') {
    return [
      { icon: 'staff', title: 'Пользователи', text: 'Учетные записи и доступ', to: '/admin/users' },
      { icon: 'attendance', title: 'Уведомления', text: 'Сообщения пользователям', to: '/admin/notifications' },
      { icon: 'grades', title: 'Отчёты', text: 'Excel и PDF по семестрам', to: '/staff/reports' },
      { icon: 'analytics', title: 'Антифрод', text: 'Журнал и рейтинг нарушений', to: '/staff/antifraud' }
    ];
  }

  if (role === 'head') {
    return [
      { icon: 'attendance', title: 'Открыть пару', text: 'Отметить посещаемость студентов', to: '/teacher/attendance' },
      { icon: 'grades', title: 'Оценки', text: 'Работы и баллы студентов', to: '/teacher/grades' },
      { icon: 'schedule', title: 'Расписание', text: 'Пары на выбранный день', to: '/schedule' },
      { icon: 'staff', title: 'Сводка', text: 'Группы, студенты и преподаватели', to: '/staff/overview' }
    ];
  }

  if (STAFF_ROLES.includes(role)) {
    return [
      { icon: 'staff', title: 'Сводка', text: 'Группы, студенты и преподаватели', to: '/staff/overview' },
      { icon: 'analytics', title: 'Аналитика', text: 'Будущая расширенная статистика', to: '/staff/analytics' },
      { icon: 'profile', title: 'Профиль', text: 'Данные аккаунта', to: '/profile' }
    ];
  }

  return [
    { icon: 'schedule', title: 'Расписание', text: 'Пары на выбранный день', to: '/schedule' },
    { icon: 'grades', title: 'Оценки', text: 'Баллы и контрольные точки', to: '/grades' },
    { icon: 'attendance', title: 'Посещаемость', text: 'QR-отметки и история', to: '/attendance' },
    { icon: 'analytics', title: 'Аналитика', text: 'Будущая диаграмма прогресса', to: '/analytics' }
  ];
};

const DashboardPage = ({ user, navigate }) => {
  const role = user?.role || 'student';
  const isStaff = role === 'head' || role === 'dean';
  const name = getDisplayName(user);
  const quickCards = getQuickCards(role);

  return (
    <section className="dashboard-page">
      <div className="dashboard-hero">
        <div>
          <span>{roleTitle(role)}</span>
          <h1>{name}</h1>
          <p>
            Быстрый старт по основным разделам электронного журнала.
          </p>
        </div>
        <div className="dashboard-hero-mark" aria-hidden="true">
          <DashboardIcon name={isStaff ? 'staff' : role === 'teacher' ? 'analytics' : 'profile'} />
        </div>
      </div>

	  <div className="dashboard-quick-grid">
		{quickCards.map((card) => (
		  <QuickCard
			key={card.to}
			icon={card.icon}
			title={card.title}
			text={card.text}
			to={card.to}
			navigate={navigate}
		  />
		))}
      </div>
    </section>
  );
};

export default DashboardPage;
