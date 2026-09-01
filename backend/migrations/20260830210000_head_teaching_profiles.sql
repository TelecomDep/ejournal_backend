-- +goose Up
-- A head of department may also teach. Keep the organisational head role on
-- users while providing the personal teacher identity used by schedules,
-- attendance sessions and grades.
INSERT INTO teachers (role, user_id, name, lectern_id, job_title)
SELECT
    'teacher',
    u.id,
    u.login,
    MIN(os.lectern_id),
    'Заведующий кафедрой'
FROM users u
JOIN org_scopes os ON os.user_id = u.id AND os.lectern_id IS NOT NULL
WHERE u.role = 'head'
  AND NOT EXISTS (
      SELECT 1 FROM teachers t WHERE t.user_id = u.id
  )
GROUP BY u.id, u.login;

-- +goose Down
DELETE FROM teachers t
USING users u
WHERE t.user_id = u.id
  AND u.role = 'head'
  AND t.job_title = 'Заведующий кафедрой'
  AND NOT EXISTS (
      SELECT 1 FROM schedules s WHERE s.teacher_id = t.teacher_id
  )
  AND NOT EXISTS (
      SELECT 1 FROM attendance_sessions a WHERE a.teacher_id = t.teacher_id
  )
  AND NOT EXISTS (
      SELECT 1 FROM grade_items gi WHERE gi.created_by_teacher_id = t.teacher_id
  );
