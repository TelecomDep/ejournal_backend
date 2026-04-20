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

func (r *AttendanceRepository) CreateSessionWithGroups(
	ctx context.Context,
	teacherID, subjectID int32,
	groupIDs []int32,
	expiresAt time.Time,
) (AttendanceSession, int32, error) {
	if teacherID <= 0 {
		return AttendanceSession{}, 0, fmt.Errorf("teacher id is required")
	}
	if subjectID <= 0 {
		return AttendanceSession{}, 0, fmt.Errorf("subject id is required")
	}
	if len(groupIDs) == 0 {
		return AttendanceSession{}, 0, fmt.Errorf("at least one group id is required")
	}
	if expiresAt.IsZero() {
		return AttendanceSession{}, 0, fmt.Errorf("session expiration time is required")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AttendanceSession{}, 0, fmt.Errorf("begin attendance session tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var out AttendanceSession
	err = tx.QueryRow(
		ctx,
		`INSERT INTO attendance_sessions (teacher_id, subject_id, expires_at)
		 VALUES ($1, $2, $3)
		 RETURNING session_id, teacher_id, subject_id, expires_at, created_at`,
		teacherID,
		subjectID,
		expiresAt.UTC(),
	).Scan(&out.ID, &out.TeacherID, &out.SubjectID, &out.ExpiresAt, &out.CreatedAt)
	if err != nil {
		return AttendanceSession{}, 0, fmt.Errorf("insert attendance session: %w", err)
	}

	for _, groupID := range groupIDs {
		if groupID <= 0 {
			return AttendanceSession{}, 0, fmt.Errorf("group id is required")
		}
		if _, err = tx.Exec(
			ctx,
			`INSERT INTO attendance_session_groups (session_id, group_id)
			 VALUES ($1, $2)
			 ON CONFLICT (session_id, group_id) DO NOTHING`,
			out.ID,
			groupID,
		); err != nil {
			return AttendanceSession{}, 0, fmt.Errorf("insert attendance session group: %w", err)
		}
	}

	cmd, err := tx.Exec(
		ctx,
		`INSERT INTO attendance_session_students (session_id, student_id, group_id_snapshot, status)
		 SELECT $1, st.student_id, st.group_id, 'absent'
		 FROM students st
		 WHERE st.group_id = ANY($2)
		 ON CONFLICT (session_id, student_id) DO NOTHING`,
		out.ID,
		groupIDs,
	)
	if err != nil {
		return AttendanceSession{}, 0, fmt.Errorf("seed attendance roster: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return AttendanceSession{}, 0, fmt.Errorf("commit attendance session tx: %w", err)
	}

	return out, int32(cmd.RowsAffected()), nil
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

func (r *AttendanceRepository) MarkStudentPresent(ctx context.Context, sessionID, studentID int32, markedAt time.Time) (string, error) {
	if sessionID <= 0 {
		return "", fmt.Errorf("session id is required")
	}
	if studentID <= 0 {
		return "", fmt.Errorf("student id is required")
	}
	if markedAt.IsZero() {
		markedAt = time.Now().UTC()
	}

	var markResult string
	err := r.pool.QueryRow(
		ctx,
		`WITH target AS (
		    SELECT status
		    FROM attendance_session_students
		    WHERE session_id = $1 AND student_id = $2
		  ),
		  upd AS (
		    UPDATE attendance_session_students
		    SET status = 'present',
		        marked_at = $3,
		        marked_by = 'self'
		    WHERE session_id = $1
		      AND student_id = $2
		      AND status <> 'present'
		    RETURNING 1
		  )
		  SELECT CASE
		    WHEN EXISTS (SELECT 1 FROM upd) THEN 'updated'
		    WHEN EXISTS (SELECT 1 FROM target) THEN 'already'
		    ELSE 'not_found'
		  END`,
		sessionID,
		studentID,
		markedAt.UTC(),
	).Scan(&markResult)
	if err != nil {
		return "", fmt.Errorf("update attendance roster status: %w", err)
	}

	return markResult, nil
}

func (r *AttendanceRepository) GetTeacherGroupAttendanceStats(
	ctx context.Context,
	teacherID int32,
	groupID int32,
	subjectID *int32,
) ([]AttendanceGroupStat, error) {
	if teacherID <= 0 {
		return nil, fmt.Errorf("teacher id is required")
	}
	if groupID <= 0 {
		return nil, fmt.Errorf("group id is required")
	}

	var subjectArg any
	if subjectID != nil {
		subjectArg = *subjectID
	}

	rows, err := r.pool.Query(
		ctx,
		`WITH scoped_sessions AS (
		     SELECT s.session_id
		     FROM attendance_sessions s
		     INNER JOIN attendance_session_groups sg
		             ON sg.session_id = s.session_id
		     WHERE s.teacher_id = $1
		       AND sg.group_id = $2
		       AND ($3::INTEGER IS NULL OR s.subject_id = $3::INTEGER)
		   ),
		   agg AS (
		     SELECT ass.student_id,
		            COUNT(*)::INTEGER AS total_sessions,
		            SUM(CASE WHEN ass.status = 'present' THEN 1 ELSE 0 END)::INTEGER AS attended_sessions,
		            MAX(CASE WHEN ass.status = 'present' THEN ass.marked_at ELSE NULL END) AS last_marked_at
		     FROM attendance_session_students ass
		     INNER JOIN scoped_sessions ss
		             ON ss.session_id = ass.session_id
		     WHERE ass.group_id_snapshot = $2
		     GROUP BY ass.student_id
		   )
		   SELECT st.student_id,
		          st.student_name,
		          COALESCE(agg.total_sessions, 0),
		          COALESCE(agg.attended_sessions, 0),
		          agg.last_marked_at
		   FROM students st
		   LEFT JOIN agg ON agg.student_id = st.student_id
		   WHERE st.group_id = $2
		   ORDER BY st.student_name, st.student_id`,
		teacherID,
		groupID,
		subjectArg,
	)
	if err != nil {
		return nil, fmt.Errorf("list teacher group attendance stats: %w", err)
	}
	defer rows.Close()

	result := make([]AttendanceGroupStat, 0)
	for rows.Next() {
		var item AttendanceGroupStat
		if err := rows.Scan(
			&item.StudentID,
			&item.StudentName,
			&item.TotalSessions,
			&item.AttendedSessions,
			&item.LastMarkedAt,
		); err != nil {
			return nil, fmt.Errorf("scan teacher group attendance stat: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teacher group attendance stats rows: %w", err)
	}

	return result, nil
}
