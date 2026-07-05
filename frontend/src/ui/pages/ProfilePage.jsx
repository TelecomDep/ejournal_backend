import React from 'react';
import { getRoleLabel } from '../navigation';

const pickDisplayName = (user) => (
  user?.student_name || user?.teacher_name || user?.name || user?.login || 'Пользователь'
);

const getInitials = (name = '') =>
  name
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0])
    .join('')
    .toUpperCase() || 'П';

const profileRows = (user) => [
  { label: 'Логин', value: user?.login },
  { label: 'Роль', value: getRoleLabel(user?.role) },
  { label: 'Группа', value: user?.group_name || user?.group },
  { label: 'Должность', value: user?.job_title },
  { label: 'ID студента', value: user?.student_id },
  { label: 'ID преподавателя', value: user?.teacher_id }
].filter((row) => row.value !== undefined && row.value !== null && row.value !== '');

const ProfilePage = ({ user }) => {
  const displayName = pickDisplayName(user);
  const rows = profileRows(user);

  return (
    <section className="profile-page">
      <header className="profile-hero">
        <div className="profile-avatar" aria-hidden="true">
          {user?.avatar ? (
            <img src={user.avatar} alt="" />
          ) : (
            <span>{getInitials(displayName)}</span>
          )}
        </div>

        <div>
          <p className="ui-kicker">Профиль</p>
          <h1>{displayName}</h1>
          <p>{getRoleLabel(user?.role)}</p>
        </div>
      </header>

      <div className="profile-details" aria-label="Данные профиля">
        {rows.map((row) => (
          <div className="profile-detail" key={row.label}>
            <span>{row.label}</span>
            <strong>{row.value}</strong>
          </div>
        ))}
      </div>
    </section>
  );
};

export default ProfilePage;
