package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttendanceRepository struct {
	pool *pgxpool.Pool
}

func NewAttendanceRepository(pool *pgxpool.Pool) *AttendanceRepository {
	return &AttendanceRepository{pool: pool}
}

func (r *AttendanceRepository) CreateSession(ctx context.Context, teacherID, subjectID int32, expiresAt time.Time) (AttendanceSession, error) {
	if teacherID <= 0 {
		return AttendanceSession{}, fmt.Errorf("teacher id is required")
	}
	if subjectID <= 0 {
		return AttendanceSession{}, fmt.Errorf("subject id is required")
	}
	if expiresAt.IsZero() {
		return AttendanceSession{}, fmt.Errorf("session expiration time is required")
	}

	var out AttendanceSession
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO attendance_sessions (teacher_id, subject_id, expires_at)
		 VALUES ($1, $2, $3)
		 RETURNING session_id, teacher_id, subject_id, expires_at, created_at`,
		teacherID,
		subjectID,
		expiresAt.UTC(),
	).Scan(&out.ID, &out.TeacherID, &out.SubjectID, &out.ExpiresAt, &out.CreatedAt)
	if err != nil {
		return AttendanceSession{}, fmt.Errorf("insert attendance session: %w", err)
	}

	return out, nil
}

func (r *AttendanceRepository) GetSessionByID(ctx context.Context, sessionID int32) (AttendanceSession, bool, error) {
	if sessionID <= 0 {
		return AttendanceSession{}, false, fmt.Errorf("session id is required")
	}

	var out AttendanceSession
	err := r.pool.QueryRow(
		ctx,
		`SELECT session_id, teacher_id, subject_id, expires_at, created_at
		 FROM attendance_sessions
		 WHERE session_id = $1`,
		sessionID,
	).Scan(&out.ID, &out.TeacherID, &out.SubjectID, &out.ExpiresAt, &out.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AttendanceSession{}, false, nil
	}
	if err != nil {
		return AttendanceSession{}, false, fmt.Errorf("get attendance session by id: %w", err)
	}

	return out, true, nil
}

func (r *AttendanceRepository) Mark(ctx context.Context, sessionID, studentID int32, markedAt time.Time) (bool, error) {
	if sessionID <= 0 {
		return false, fmt.Errorf("session id is required")
	}
	if studentID <= 0 {
		return false, fmt.Errorf("student id is required")
	}
	if markedAt.IsZero() {
		markedAt = time.Now().UTC()
	}

	cmd, err := r.pool.Exec(
		ctx,
		`INSERT INTO attendance_marks (session_id, student_id, marked_at)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (session_id, student_id) DO NOTHING`,
		sessionID,
		studentID,
		markedAt.UTC(),
	)
	if err != nil {
		return false, fmt.Errorf("insert attendance mark: %w", err)
	}

	return cmd.RowsAffected() > 0, nil
}
