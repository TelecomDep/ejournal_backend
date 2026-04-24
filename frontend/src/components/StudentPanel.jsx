import React, { useState } from 'react';

function StudentPanel({ onConfirmAttendance, loading, confirmResult }) {
  const [inviteToken, setInviteToken] = useState('');

  return (
    <section className="card">
      <h2>Студент: подтверждение посещаемости</h2>
      <div className="field">
        <span>Invite token</span>
        <input
          placeholder="Вставьте invite token"
          value={inviteToken}
          onChange={(e) => setInviteToken(e.target.value)}
        />
      </div>

      <button
        className="btn btn-primary"
        onClick={() => onConfirmAttendance(inviteToken.trim())}
        disabled={loading || !inviteToken.trim()}
      >
        Подтвердить посещаемость
      </button>

      {confirmResult && (
        <div className="result-box">
          <div><strong>Статус:</strong> {confirmResult.attendance || '-'}</div>
          <div><strong>Session:</strong> {confirmResult.session_id || '-'}</div>
          <div><strong>Student:</strong> {confirmResult.student_id || '-'}</div>
          <div><strong>Time:</strong> {confirmResult.marked_at || '-'}</div>
        </div>
      )}
    </section>
  );
}

export default StudentPanel;
