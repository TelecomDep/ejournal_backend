import React, { useEffect, useMemo, useState } from 'react';
import api from '../../services/api';

const STAFF_ROLES = ['admin', 'head', 'dean'];

const getDisplayName = (user) => (
  user?.teacher_name || user?.name || user?.login || 'Пользователь'
);

const roleTitle = (role) => {
  if (role === 'teacher') return 'Рабочая панель преподавателя';
  if (STAFF_ROLES.includes(role)) return 'Сводная панель';
  return 'Личный кабинет студента';
};

const clampPct = (value) => Math.max(0, Math.min(100, Math.round(Number(value) || 0)));

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

const MetricCard = ({ icon, label, value, helper }) => (
  <article className="dashboard-metric-card">
    <span className="dashboard-card-icon"><DashboardIcon name={icon} /></span>
    <span>
      <small>{label}</small>
      <strong>{value}</strong>
      <em>{helper}</em>
    </span>
  </article>
);

const StaffSnapshot = ({ token, navigate }) => {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    api.getStaffOverview(token)
      .then((payload) => {
        if (!cancelled) {
          setData(payload);
          setError('');
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(api.getErrorMessage(err, 'Не удалось загрузить сводку'));
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [token]);

  const stats = useMemo(() => {
    const students = data?.students || [];
    const counts = data?.counts || {};
    const average = students.length
      ? Math.round(students.reduce((sum, item) => sum + clampPct(item.attendance_pct), 0) / students.length)
      : 0;
    const risk = students.filter((item) => clampPct(item.attendance_pct) < 60).length;
    return { counts, average, risk };
  }, [data]);

  if (loading) {
    return <div className="dashboard-panel dashboard-panel-muted">Загрузка сводки...</div>;
  }

  if (error) {
    return <div className="dashboard-panel dashboard-panel-error">{error}</div>;
  }

  return (
    <section className="dashboard-panel">
      <div className="dashboard-panel-head">
        <div>
          <span>Сводка доступа</span>
          <h2>{data?.label || 'Область контроля'}</h2>
        </div>
        <button type="button" onClick={() => navigate('/staff/overview')}>Открыть сводку</button>
      </div>

      <div className="dashboard-metric-grid is-staff">
        <MetricCard icon="staff" label="Студенты" value={stats.counts.students ?? 0} helper="в области" />
        <MetricCard icon="analytics" label="Средняя посещаемость" value={`${stats.average}%`} helper="по студентам" />
        <MetricCard icon="attendance" label="Зона риска" value={stats.risk} helper="ниже 60%" />
      </div>
    </section>
  );
};

const getQuickCards = (role) => {
  if (role === 'teacher') {
    return [
      { icon: 'attendance', title: 'Открыть QR', text: 'Создать отметку посещаемости', to: '/teacher/attendance' },
      { icon: 'grades', title: 'Оценки', text: 'Работы и баллы студентов', to: '/teacher/grades' },
      { icon: 'analytics', title: 'Статистика группы', text: 'Посещаемость и успеваемость', to: '/teacher/stats' },
      { icon: 'profile', title: 'Профиль', text: 'Данные аккаунта', to: '/profile' }
    ];
  }

  if (role === 'admin') {
    return [
      { icon: 'staff', title: 'Пользователи', text: 'Учетные записи и доступ', to: '/admin/users' },
      { icon: 'analytics', title: 'Сводка', text: 'Группы и показатели', to: '/staff/overview' },
      { icon: 'analytics', title: 'Аналитика', text: 'Расширенная статистика', to: '/staff/analytics' },
      { icon: 'profile', title: 'Профиль', text: 'Данные аккаунта', to: '/profile' }
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

const DashboardPage = ({ user, token, navigate }) => {
  const role = user?.role || 'student';
  const isStaff = STAFF_ROLES.includes(role);
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

      {isStaff ? (
        <StaffSnapshot token={token} navigate={navigate} />
      ) : (
        <section className="dashboard-panel">
          <div className="dashboard-panel-head">
            <div>
              <span>Следующие блоки</span>
              <h2>{role === 'teacher' ? 'Преподавательский контур' : 'Студенческий контур'}</h2>
            </div>
          </div>
          <div className="dashboard-metric-grid">
            <MetricCard icon="schedule" label="Навигация" value="Готова" helper="основные разделы" />
            <MetricCard icon="attendance" label="Посещаемость" value="В работе" helper="QR и история" />
            <MetricCard icon="grades" label="Оценки" value="В работе" helper="баллы и ведомость" />
          </div>
        </section>
      )}
    </section>
  );
};

export default DashboardPage;
