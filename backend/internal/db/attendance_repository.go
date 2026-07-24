package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	teacherID, subjectID, semesterID int32,
	groupIDs []int32,
	expiresAt time.Time,
) (AttendanceSession, int32, error) {
	return r.CreateSessionWithGroupsAndLocation(ctx, teacherID, subjectID, semesterID, groupIDs, expiresAt, "", nil, nil)
}

func (r *AttendanceRepository) CreateSessionWithGroupsAndLocation(
	ctx context.Context,
	teacherID, subjectID, semesterID int32,
	groupIDs []int32,
	expiresAt time.Time,
	lessonName string,
	lat, lon *float64,
) (AttendanceSession, int32, error) {
	if teacherID <= 0 {
		return AttendanceSession{}, 0, fmt.Errorf("teacher id is required")
	}
	if subjectID <= 0 {
		return AttendanceSession{}, 0, fmt.Errorf("subject id is required")
	}
	if semesterID <= 0 {
		return AttendanceSession{}, 0, fmt.Errorf("semester id is required")
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
		`INSERT INTO attendance_sessions (teacher_id, subject_id, semester_id, expires_at, lesson_name, lat, lon)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7)
		 RETURNING session_id, teacher_id, subject_id, semester_id, COALESCE(lesson_name, ''), lat, lon, expires_at, created_at`,
		teacherID,
		subjectID,
		semesterID,
		expiresAt.UTC(),
		strings.TrimSpace(lessonName),
		lat,
		lon,
	).Scan(&out.ID, &out.TeacherID, &out.SubjectID, &out.SemesterID, &out.LessonName, &out.Lat, &out.Lon, &out.ExpiresAt, &out.CreatedAt)
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
		`SELECT session_id, teacher_id, subject_id, semester_id, COALESCE(lesson_name, ''), lat, lon, expires_at, created_at
		 FROM attendance_sessions
		 WHERE session_id = $1`,
		sessionID,
	).Scan(&out.ID, &out.TeacherID, &out.SubjectID, &out.SemesterID, &out.LessonName, &out.Lat, &out.Lon, &out.ExpiresAt, &out.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AttendanceSession{}, false, nil
	}
	if err != nil {
		return AttendanceSession{}, false, fmt.Errorf("get attendance session by id: %w", err)
	}

	return out, true, nil
}

func (r *AttendanceRepository) GetActiveSessionByTeacherID(ctx context.Context, teacherID int32, now time.Time) (AttendanceSession, bool, error) {
	if teacherID <= 0 {
		return AttendanceSession{}, false, fmt.Errorf("teacher id is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var out AttendanceSession
	err := r.pool.QueryRow(
		ctx,
		`SELECT session_id, teacher_id, subject_id, semester_id, COALESCE(lesson_name, ''), lat, lon, expires_at, created_at
		 FROM attendance_sessions
		 WHERE teacher_id = $1
		   AND expires_at > $2
		 ORDER BY expires_at DESC, session_id DESC
		 LIMIT 1`,
		teacherID,
		now.UTC(),
	).Scan(&out.ID, &out.TeacherID, &out.SubjectID, &out.SemesterID, &out.LessonName, &out.Lat, &out.Lon, &out.ExpiresAt, &out.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AttendanceSession{}, false, nil
	}
	if err != nil {
		return AttendanceSession{}, false, fmt.Errorf("get active attendance session: %w", err)
	}

	return out, true, nil
}

func (r *AttendanceRepository) GetSessionProgress(ctx context.Context, sessionID int32) (AttendanceSessionProgress, error) {
	if sessionID <= 0 {
		return AttendanceSessionProgress{}, fmt.Errorf("session id is required")
	}

	var out AttendanceSessionProgress
	err := r.pool.QueryRow(
		ctx,
		`SELECT COUNT(*)::INTEGER AS roster_size,
		        COUNT(*) FILTER (WHERE status <> 'absent')::INTEGER AS marked_count
		 FROM attendance_session_students
		 WHERE session_id = $1`,
		sessionID,
	).Scan(&out.RosterSize, &out.MarkedCount)
	if err != nil {
		return AttendanceSessionProgress{}, fmt.Errorf("get attendance session progress: %w", err)
	}

	return out, nil
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

func (r *AttendanceRepository) MarkStudentPresentFromDevice(
	ctx context.Context,
	sessionID, studentID int32,
	markedAt time.Time,
	deviceID string,
	lat, lon float64,
	fraudReason string,
) (string, error) {
	if sessionID <= 0 {
		return "", fmt.Errorf("session id is required")
	}
	if studentID <= 0 {
		return "", fmt.Errorf("student id is required")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return "", fmt.Errorf("device id is required")
	}
	if markedAt.IsZero() {
		markedAt = time.Now().UTC()
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("begin mobile attendance tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var currentStatus string
	err = tx.QueryRow(
		ctx,
		`SELECT status
		 FROM attendance_session_students
		 WHERE session_id = $1 AND student_id = $2
		 FOR UPDATE`,
		sessionID,
		studentID,
	).Scan(&currentStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return "not_found", nil
	}
	if err != nil {
		return "", fmt.Errorf("load attendance roster row: %w", err)
	}
	if currentStatus == "present" {
		return "already", nil
	}

	if fraudReason == "" {
		var duplicateDevice bool
		err = tx.QueryRow(
			ctx,
			`SELECT EXISTS (
			     SELECT 1
			     FROM attendance_session_students
			     WHERE session_id = $1
			       AND student_id <> $2
			       AND device_id = $3
			       AND status = 'present'
			 )`,
			sessionID,
			studentID,
			deviceID,
		).Scan(&duplicateDevice)
		if err != nil {
			return "", fmt.Errorf("check duplicate attendance device: %w", err)
		}
		if duplicateDevice {
			fraudReason = "device_id already used in this lesson"
		}
	}

	if fraudReason != "" {
		_, err = tx.Exec(
			ctx,
			`UPDATE attendance_session_students
			 SET marked_at = $3,
			     marked_by = 'self',
			     device_id = $4,
			     check_in_lat = $5,
			     check_in_lon = $6,
			     is_fraud = TRUE,
			     fraud_reason = $7
			 WHERE session_id = $1 AND student_id = $2`,
			sessionID,
			studentID,
			markedAt.UTC(),
			deviceID,
			lat,
			lon,
			fraudReason,
		)
		if err != nil {
			return "", fmt.Errorf("save attendance fraud attempt: %w", err)
		}
		_, err = tx.Exec(
			ctx,
			`UPDATE students
			 SET total_cheat_attempts = total_cheat_attempts + 1
			 WHERE student_id = $1`,
			studentID,
		)
		if err != nil {
			return "", fmt.Errorf("increment student cheat attempts: %w", err)
		}
		if err = tx.Commit(ctx); err != nil {
			return "", fmt.Errorf("commit attendance fraud attempt: %w", err)
		}
		return "fraud", nil
	}

	_, err = tx.Exec(
		ctx,
		`UPDATE attendance_session_students
		 SET status = 'present',
		     marked_at = $3,
		     marked_by = 'self',
		     device_id = $4,
		     check_in_lat = $5,
		     check_in_lon = $6,
		     is_fraud = FALSE,
		     fraud_reason = NULL
		 WHERE session_id = $1 AND student_id = $2`,
		sessionID,
		studentID,
		markedAt.UTC(),
		deviceID,
		lat,
		lon,
	)
	if err != nil {
		return "", fmt.Errorf("mark mobile student present: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit mobile attendance tx: %w", err)
	}

	return "updated", nil
}

func (r *AttendanceRepository) SetStudentAttendanceStatus(
	ctx context.Context,
	sessionID, studentID int32,
	status string,
	markedAt time.Time,
) (string, error) {
	if sessionID <= 0 {
		return "", fmt.Errorf("session id is required")
	}
	if studentID <= 0 {
		return "", fmt.Errorf("student id is required")
	}
	if markedAt.IsZero() {
		markedAt = time.Now().UTC()
	}

	cmd, err := r.pool.Exec(
		ctx,
		`UPDATE attendance_session_students
		 SET status = $3,
		     marked_at = $4,
		     marked_by = 'teacher'
		 WHERE session_id = $1 AND student_id = $2`,
		sessionID,
		studentID,
		status,
		markedAt.UTC(),
	)
	if err != nil {
		return "", fmt.Errorf("set student attendance status: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return "not_found", nil
	}

	return "updated", nil
}

func (r *AttendanceRepository) GetStudentSubjectAttendanceHistory(
	ctx context.Context,
	studentID, subjectID, semesterID int32,
) ([]AttendanceHistoryItem, error) {
	if studentID <= 0 {
		return nil, fmt.Errorf("student id is required")
	}
	if subjectID <= 0 {
		return nil, fmt.Errorf("subject id is required")
	}
	if semesterID <= 0 {
		return nil, fmt.Errorf("semester id is required")
	}

	rows, err := r.pool.Query(
		ctx,
		`SELECT (s.created_at AT TIME ZONE 'Asia/Novosibirsk')::date AS day,
		        COALESCE(s.lesson_name, ''),
		        ass.status
		 FROM attendance_session_students ass
		 INNER JOIN attendance_sessions s ON s.session_id = ass.session_id
		 WHERE ass.student_id = $1
		   AND s.subject_id = $2
		   AND s.semester_id = $3
		 ORDER BY s.created_at`,
		studentID,
		subjectID,
		semesterID,
	)
	if err != nil {
		return nil, fmt.Errorf("list student subject attendance history: %w", err)
	}
	defer rows.Close()

	result := make([]AttendanceHistoryItem, 0)
	for rows.Next() {
		var item AttendanceHistoryItem
		if err := rows.Scan(&item.Date, &item.LessonName, &item.Status); err != nil {
			return nil, fmt.Errorf("scan student subject attendance history: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student subject attendance history rows: %w", err)
	}

	return result, nil
}

func (r *AttendanceRepository) GetTeacherGroupAttendanceStats(
	ctx context.Context,
	teacherID int32,
	groupID int32,
	subjectID *int32,
	semesterID int32,
) ([]AttendanceGroupStat, error) {
	if teacherID <= 0 {
		return nil, fmt.Errorf("teacher id is required")
	}
	if groupID <= 0 {
		return nil, fmt.Errorf("group id is required")
	}
	if semesterID <= 0 {
		return nil, fmt.Errorf("semester id is required")
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
		       AND s.semester_id = $3
		       AND ($4::INTEGER IS NULL OR s.subject_id = $4::INTEGER)
		   ),
		   agg AS (
		     SELECT ass.student_id,
		            COUNT(*)::INTEGER AS total_sessions,
		            SUM(CASE WHEN ass.status IN ('present', 'late') THEN 1 ELSE 0 END)::INTEGER AS attended_sessions,
		            SUM(CASE WHEN ass.status = 'excused' THEN 1 ELSE 0 END)::INTEGER AS excused_sessions,
		            MAX(CASE WHEN ass.status IN ('present', 'late') THEN ass.marked_at ELSE NULL END) AS last_marked_at
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
		          COALESCE(agg.excused_sessions, 0),
		          agg.last_marked_at
		   FROM students st
		   LEFT JOIN agg ON agg.student_id = st.student_id
		   WHERE st.group_id = $2
		   ORDER BY st.student_name, st.student_id`,
		teacherID,
		groupID,
		semesterID,
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
			&item.ExcusedSessions,
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

// GetGroupSubjectPerformance returns, for each student of the group, combined
// attendance (sessions) and grade totals on the given subject.
func (r *AttendanceRepository) GetGroupSubjectPerformance(
	ctx context.Context,
	teacherID int32,
	groupID int32,
	subjectID int32,
	semesterID int32,
) ([]GroupSubjectPerformanceRow, error) {
	if teacherID <= 0 {
		return nil, fmt.Errorf("teacher id is required")
	}
	if groupID <= 0 {
		return nil, fmt.Errorf("group id is required")
	}
	if subjectID <= 0 {
		return nil, fmt.Errorf("subject id is required")
	}
	if semesterID <= 0 {
		return nil, fmt.Errorf("semester id is required")
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
		       AND s.subject_id = $3
		       AND s.semester_id = $4
		   ),
		   att AS (
		     SELECT ass.student_id,
		            COUNT(*)::INTEGER AS total_sessions,
		            SUM(CASE WHEN ass.status IN ('present', 'late') THEN 1 ELSE 0 END)::INTEGER AS attended_sessions,
		            SUM(CASE WHEN ass.status = 'excused' THEN 1 ELSE 0 END)::INTEGER AS excused_sessions
		     FROM attendance_session_students ass
		     INNER JOIN scoped_sessions ss
		             ON ss.session_id = ass.session_id
		     WHERE ass.group_id_snapshot = $2
		     GROUP BY ass.student_id
		   ),
		   grd AS (
		     SELECT st.student_id,
		            COALESCE(SUM(gi.max_score), 0)::INTEGER AS total_max,
		            COALESCE(SUM(CASE WHEN gi.deadline < now() THEN gi.max_score ELSE 0 END), 0)::INTEGER AS passed_max,
		            COALESCE(SUM(g.score), 0)::INTEGER AS current_score
		     FROM students st
		     LEFT JOIN grade_items gi ON gi.subject_id = $3 AND gi.semester_id = $4 AND gi.deleted_at IS NULL
		     LEFT JOIN grades g ON g.item_id = gi.item_id AND g.student_id = st.student_id AND g.deleted_at IS NULL
		     WHERE st.group_id = $2
		     GROUP BY st.student_id
		   )
		   SELECT st.student_id,
		          st.student_name,
		          COALESCE(att.total_sessions, 0),
		          COALESCE(att.attended_sessions, 0),
		          COALESCE(att.excused_sessions, 0),
		          COALESCE(grd.total_max, 0),
		          COALESCE(grd.passed_max, 0),
		          COALESCE(grd.current_score, 0)
		   FROM students st
		   LEFT JOIN att ON att.student_id = st.student_id
		   LEFT JOIN grd ON grd.student_id = st.student_id
		   WHERE st.group_id = $2
		   ORDER BY st.student_name, st.student_id`,
		teacherID,
		groupID,
		subjectID,
		semesterID,
	)
	if err != nil {
		return nil, fmt.Errorf("list group subject performance: %w", err)
	}
	defer rows.Close()

	result := make([]GroupSubjectPerformanceRow, 0)
	for rows.Next() {
		var item GroupSubjectPerformanceRow
		if err := rows.Scan(
			&item.StudentID,
			&item.StudentName,
			&item.TotalSessions,
			&item.AttendedSessions,
			&item.ExcusedSessions,
			&item.TotalMax,
			&item.PassedMax,
			&item.CurrentScore,
		); err != nil {
			return nil, fmt.Errorf("scan group subject performance row: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate group subject performance rows: %w", err)
	}

	return result, nil
}
