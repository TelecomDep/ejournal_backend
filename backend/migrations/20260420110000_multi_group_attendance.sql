-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS attendance_session_groups (
    session_id INTEGER NOT NULL REFERENCES attendance_sessions(session_id) ON DELETE CASCADE,
    group_id INTEGER NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    PRIMARY KEY (session_id, group_id)
);

CREATE TABLE IF NOT EXISTS attendance_session_students (
    session_id INTEGER NOT NULL REFERENCES attendance_sessions(session_id) ON DELETE CASCADE,
    student_id INTEGER NOT NULL REFERENCES students(student_id) ON DELETE CASCADE,
    group_id_snapshot INTEGER NOT NULL REFERENCES groups(group_id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'absent' CHECK (status IN ('absent', 'present', 'late', 'excused')),
    marked_at TIMESTAMPTZ,
    marked_by VARCHAR(20) CHECK (marked_by IN ('self', 'teacher')),
    PRIMARY KEY (session_id, student_id)
);

CREATE INDEX IF NOT EXISTS idx_attendance_session_groups_group_session
    ON attendance_session_groups (group_id, session_id);

CREATE INDEX IF NOT EXISTS idx_attendance_session_students_group_session
    ON attendance_session_students (group_id_snapshot, session_id);

CREATE INDEX IF NOT EXISTS idx_attendance_session_students_student_session
    ON attendance_session_students (student_id, session_id);

CREATE INDEX IF NOT EXISTS idx_attendance_session_students_session_status
    ON attendance_session_students (session_id, status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_attendance_session_students_session_status;
DROP INDEX IF EXISTS idx_attendance_session_students_student_session;
DROP INDEX IF EXISTS idx_attendance_session_students_group_session;
DROP INDEX IF EXISTS idx_attendance_session_groups_group_session;

DROP TABLE IF EXISTS attendance_session_students;
DROP TABLE IF EXISTS attendance_session_groups;
-- +goose StatementEnd
