import React, { useMemo, useState } from 'react';

function parseGroupIds(input) {
  return input
    .split(',')
    .map((x) => Number(x.trim()))
    .filter((n) => Number.isInteger(n) && n > 0);
}

function TeacherPanel({ onCreateLink, onLoadStats, lastLink, stats, loading }) {
  const [subjectId, setSubjectId] = useState('1');
  const [groupIdsText, setGroupIdsText] = useState('1');
  const [lessonName, setLessonName] = useState('Networks');
  const [expires, setExpires] = useState('20');

  const [statsGroupId, setStatsGroupId] = useState('1');
  const [statsSubjectId, setStatsSubjectId] = useState('');

  const groupIds = useMemo(() => parseGroupIds(groupIdsText), [groupIdsText]);

  const submitCreate = () => {
    onCreateLink({
      subject_id: Number(subjectId),
      group_ids: groupIds,
      lesson_name: lessonName.trim(),
      expires_minutes: Number(expires)
    });
  };

  const submitStats = () => {
    const payload = { group_id: Number(statsGroupId) };
    if (statsSubjectId.trim()) {
      payload.subject_id = Number(statsSubjectId);
    }
    onLoadStats(payload);
  };

  const copyLink = async () => {
    if (!lastLink?.join_url) return;
    await navigator.clipboard.writeText(lastLink.join_url);
  };

  return (
    <div className="grid two">
      <section className="card">
        <h2>Преподаватель: создать сессию</h2>
        <div className="field"><span>Subject ID</span><input value={subjectId} onChange={(e) => setSubjectId(e.target.value)} /></div>
        <div className="field"><span>Group IDs (через запятую)</span><input value={groupIdsText} onChange={(e) => setGroupIdsText(e.target.value)} /></div>
        <div className="field"><span>Lesson name</span><input value={lessonName} onChange={(e) => setLessonName(e.target.value)} /></div>
        <div className="field"><span>Expires minutes</span><input value={expires} onChange={(e) => setExpires(e.target.value)} /></div>

        <div className="row gap-sm">
          <button className="btn btn-primary" onClick={submitCreate} disabled={loading}>Создать ссылку</button>
          <button className="btn" onClick={copyLink} disabled={!lastLink?.join_url}>Копировать ссылку</button>
        </div>

        {lastLink?.join_url && (
          <div className="result-box">
            <div><strong>Join URL:</strong></div>
            <div className="mono break">{lastLink.join_url}</div>
            <div><strong>Invite token:</strong></div>
            <div className="mono break">{lastLink.invite_token}</div>
          </div>
        )}
      </section>

      <section className="card">
        <h2>Преподаватель: статистика группы</h2>
        <div className="field"><span>Group ID</span><input value={statsGroupId} onChange={(e) => setStatsGroupId(e.target.value)} /></div>
        <div className="field"><span>Subject ID (опционально)</span><input value={statsSubjectId} onChange={(e) => setStatsSubjectId(e.target.value)} /></div>
        <button className="btn btn-primary" onClick={submitStats} disabled={loading}>Загрузить посещаемость</button>

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>student_id</th>
                <th>student_name</th>
                <th>attended</th>
                <th>total</th>
                <th>%</th>
              </tr>
            </thead>
            <tbody>
              {stats.length === 0 ? (
                <tr><td colSpan="5" className="muted">Нет данных</td></tr>
              ) : (
                stats.map((row) => (
                  <tr key={row.student_id}>
                    <td>{row.student_id}</td>
                    <td>{row.student_name}</td>
                    <td>{row.attended_sessions}</td>
                    <td>{row.total_sessions}</td>
                    <td>{Number(row.attendance_percent || 0).toFixed(2)}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}

export default TeacherPanel;
