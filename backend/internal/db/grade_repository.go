package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GradeRepository struct {
	pool *pgxpool.Pool
}

type GradeItemDeleteResult struct {
	Item           GradeItem
	AffectedGrades int32
}

func NewGradeRepository(pool *pgxpool.Pool) *GradeRepository {
	return &GradeRepository{pool: pool}
}

func scanGradeItem(row pgx.Row) (GradeItem, error) {
	var out GradeItem
	var deadline, deletedAt sql.NullTime
	var createdBy, deletedBy, semesterID sql.NullInt32
	var deleteReason sql.NullString
	err := row.Scan(
		&out.ID,
		&out.SubjectID,
		&semesterID,
		&createdBy,
		&out.Title,
		&out.MaxScore,
		&out.ItemType,
		&deadline,
		&out.CreatedAt,
		&deletedAt,
		&deletedBy,
		&deleteReason,
	)
	if err != nil {
		return GradeItem{}, err
	}
	if semesterID.Valid {
		out.SemesterID = semesterID.Int32
	}
	if createdBy.Valid {
		out.CreatedByTeacherID = &createdBy.Int32
	}
	if deadline.Valid {
		out.Deadline = &deadline.Time
	}
	if deletedAt.Valid {
		out.DeletedAt = &deletedAt.Time
	}
	if deletedBy.Valid {
		out.DeletedByUserID = &deletedBy.Int32
	}
	if deleteReason.Valid {
		out.DeleteReason = &deleteReason.String
	}
	return out, nil
}

func scanGrade(row pgx.Row) (Grade, error) {
	var out Grade
	var deletedAt sql.NullTime
	var deletedBy sql.NullInt32
	var deleteReason sql.NullString
	err := row.Scan(
		&out.ID,
		&out.StudentID,
		&out.ItemID,
		&out.TeacherID,
		&out.Score,
		&out.SessionID,
		&out.Comment,
		&out.CreatedAt,
		&out.UpdatedAt,
		&deletedAt,
		&deletedBy,
		&deleteReason,
	)
	if err != nil {
		return Grade{}, err
	}
	if deletedAt.Valid {
		out.DeletedAt = &deletedAt.Time
	}
	if deletedBy.Valid {
		out.DeletedByUserID = &deletedBy.Int32
	}
	if deleteReason.Valid {
		out.DeleteReason = &deleteReason.String
	}
	return out, nil
}

func (r *GradeRepository) CreateGradeItem(ctx context.Context, item GradeItem, actorUserID int32) (GradeItem, error) {
	item.Title = strings.TrimSpace(item.Title)
	item.ItemType = strings.TrimSpace(item.ItemType)
	if item.SubjectID <= 0 {
		return GradeItem{}, fmt.Errorf("subject id is required")
	}
	if item.SemesterID <= 0 {
		return GradeItem{}, fmt.Errorf("semester id is required")
	}
	if item.CreatedByTeacherID == nil || *item.CreatedByTeacherID <= 0 {
		return GradeItem{}, fmt.Errorf("grade item owner is required")
	}
	if actorUserID <= 0 {
		return GradeItem{}, fmt.Errorf("actor user id is required")
	}
	if item.Title == "" {
		return GradeItem{}, fmt.Errorf("grade item title is required")
	}
	if item.MaxScore <= 0 {
		return GradeItem{}, fmt.Errorf("max score must be positive")
	}
	if item.ItemType == "" {
		item.ItemType = "current"
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return GradeItem{}, fmt.Errorf("begin create grade item transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	out, err := scanGradeItem(tx.QueryRow(
		ctx,
		`INSERT INTO grade_items (subject_id, semester_id, created_by_teacher_id, title, max_score, item_type, deadline)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING item_id, subject_id, semester_id, created_by_teacher_id, title, max_score, item_type, deadline,
		           created_at, deleted_at, deleted_by_user_id, delete_reason`,
		item.SubjectID,
		item.SemesterID,
		*item.CreatedByTeacherID,
		item.Title,
		item.MaxScore,
		item.ItemType,
		item.Deadline,
	))
	if err != nil {
		return GradeItem{}, fmt.Errorf("create grade item: %w", err)
	}
	if err := insertGradeEvent(ctx, tx, nil, out.ID, nil, actorUserID, "grade_item_created", nil, nil, ""); err != nil {
		return GradeItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return GradeItem{}, fmt.Errorf("commit create grade item transaction: %w", err)
	}
	return out, nil
}

func (r *GradeRepository) GetGradeItemByID(ctx context.Context, itemID int32) (GradeItem, bool, error) {
	return r.getGradeItemByID(ctx, itemID, false)
}

func (r *GradeRepository) GetGradeItemByIDIncludingDeleted(ctx context.Context, itemID int32) (GradeItem, bool, error) {
	return r.getGradeItemByID(ctx, itemID, true)
}

func (r *GradeRepository) getGradeItemByID(ctx context.Context, itemID int32, includeDeleted bool) (GradeItem, bool, error) {
	if itemID <= 0 {
		return GradeItem{}, false, fmt.Errorf("grade item id is required")
	}

	query := `SELECT item_id, subject_id, semester_id, created_by_teacher_id, title, max_score, item_type, deadline,
	                 created_at, deleted_at, deleted_by_user_id, delete_reason
	          FROM grade_items WHERE item_id = $1`
	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	}
	out, err := scanGradeItem(r.pool.QueryRow(ctx, query, itemID))
	if errors.Is(err, pgx.ErrNoRows) {
		return GradeItem{}, false, nil
	}
	if err != nil {
		return GradeItem{}, false, fmt.Errorf("get grade item by id: %w", err)
	}
	return out, true, nil
}

func (r *GradeRepository) GetGradeItemsBySubject(ctx context.Context, subjectID, semesterID int32) ([]GradeItem, error) {
	if subjectID <= 0 {
		return nil, fmt.Errorf("subject id is required")
	}
	if semesterID <= 0 {
		return nil, fmt.Errorf("semester id is required")
	}
	rows, err := r.pool.Query(
		ctx,
		`SELECT item_id, subject_id, semester_id, created_by_teacher_id, title, max_score, item_type, deadline,
		        created_at, deleted_at, deleted_by_user_id, delete_reason
		 FROM grade_items
		 WHERE subject_id = $1 AND semester_id = $2 AND deleted_at IS NULL
		 ORDER BY deadline NULLS LAST, created_at, item_id`,
		subjectID,
		semesterID,
	)
	if err != nil {
		return nil, fmt.Errorf("query grade items: %w", err)
	}
	defer rows.Close()

	result := make([]GradeItem, 0)
	for rows.Next() {
		item, scanErr := scanGradeItem(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan grade item: %w", scanErr)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate grade item rows: %w", err)
	}
	return result, nil
}

func (r *GradeRepository) UpsertGrade(ctx context.Context, grade Grade, actorUserID int32) (Grade, error) {
	if grade.StudentID <= 0 {
		return Grade{}, fmt.Errorf("student id is required")
	}
	if grade.ItemID <= 0 {
		return Grade{}, fmt.Errorf("grade item id is required")
	}
	if grade.Score < 0 {
		return Grade{}, fmt.Errorf("score must be non-negative")
	}
	if actorUserID <= 0 {
		return Grade{}, fmt.Errorf("actor user id is required")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Grade{}, fmt.Errorf("begin upsert grade transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	previous, previousFound, err := getGradeByStudentAndItem(ctx, tx, grade.StudentID, grade.ItemID)
	if err != nil {
		return Grade{}, err
	}

	out, err := scanGrade(tx.QueryRow(
		ctx,
		`INSERT INTO grades (student_id, item_id, teacher_id, score, session_id, comment)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (student_id, item_id)
		 DO UPDATE SET
		     score = EXCLUDED.score,
		     teacher_id = EXCLUDED.teacher_id,
		     session_id = EXCLUDED.session_id,
		     comment = EXCLUDED.comment,
		     deleted_at = NULL,
		     deleted_by_user_id = NULL,
		     delete_reason = NULL
		 RETURNING grade_id, student_id, item_id, teacher_id, score, session_id, comment, created_at, updated_at,
		           deleted_at, deleted_by_user_id, delete_reason`,
		grade.StudentID,
		grade.ItemID,
		grade.TeacherID,
		grade.Score,
		grade.SessionID,
		grade.Comment,
	))
	if err != nil {
		return Grade{}, fmt.Errorf("upsert grade: %w", err)
	}

	eventType := "grade_created"
	var oldScore *int32
	if previousFound {
		oldScore = &previous.Score
		if previous.DeletedAt != nil {
			eventType = "grade_restored"
		} else {
			eventType = "grade_updated"
		}
	}
	newScore := out.Score
	if err := insertGradeEvent(ctx, tx, &out.ID, out.ItemID, &out.StudentID, actorUserID, eventType, oldScore, &newScore, ""); err != nil {
		return Grade{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Grade{}, fmt.Errorf("commit upsert grade transaction: %w", err)
	}
	return out, nil
}

func (r *GradeRepository) GetGradeByID(ctx context.Context, gradeID int32) (Grade, bool, error) {
	if gradeID <= 0 {
		return Grade{}, false, fmt.Errorf("grade id is required")
	}
	grade, err := scanGrade(r.pool.QueryRow(
		ctx,
		`SELECT grade_id, student_id, item_id, teacher_id, score, session_id, comment, created_at, updated_at,
		        deleted_at, deleted_by_user_id, delete_reason
		 FROM grades WHERE grade_id = $1`,
		gradeID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Grade{}, false, nil
	}
	if err != nil {
		return Grade{}, false, fmt.Errorf("get grade by id: %w", err)
	}
	return grade, true, nil
}

func getGradeByStudentAndItem(ctx context.Context, tx pgx.Tx, studentID, itemID int32) (Grade, bool, error) {
	grade, err := scanGrade(tx.QueryRow(
		ctx,
		`SELECT grade_id, student_id, item_id, teacher_id, score, session_id, comment, created_at, updated_at,
		        deleted_at, deleted_by_user_id, delete_reason
		 FROM grades WHERE student_id = $1 AND item_id = $2 FOR UPDATE`,
		studentID,
		itemID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Grade{}, false, nil
	}
	if err != nil {
		return Grade{}, false, fmt.Errorf("load previous grade: %w", err)
	}
	return grade, true, nil
}

func (r *GradeRepository) CountActiveGradesByItem(ctx context.Context, itemID int32) (int32, error) {
	var count int32
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*)::INTEGER FROM grades WHERE item_id = $1 AND deleted_at IS NULL`, itemID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active grades: %w", err)
	}
	return count, nil
}

func (r *GradeRepository) SoftDeleteGrade(ctx context.Context, gradeID, actorUserID int32, reason string) (Grade, bool, error) {
	if gradeID <= 0 || actorUserID <= 0 {
		return Grade{}, false, fmt.Errorf("grade id and actor user id are required")
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Grade{}, false, fmt.Errorf("begin delete grade transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	grade, err := scanGrade(tx.QueryRow(
		ctx,
		`UPDATE grades
		 SET deleted_at = now(), deleted_by_user_id = $2, delete_reason = NULLIF($3, '')
		 WHERE grade_id = $1 AND deleted_at IS NULL
		 RETURNING grade_id, student_id, item_id, teacher_id, score, session_id, comment, created_at, updated_at,
		          deleted_at, deleted_by_user_id, delete_reason`,
		gradeID,
		actorUserID,
		strings.TrimSpace(reason),
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Grade{}, false, nil
	}
	if err != nil {
		return Grade{}, false, fmt.Errorf("soft delete grade: %w", err)
	}
	oldScore := grade.Score
	if err := insertGradeEvent(ctx, tx, &grade.ID, grade.ItemID, &grade.StudentID, actorUserID, "grade_deleted", &oldScore, nil, reason); err != nil {
		return Grade{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Grade{}, false, fmt.Errorf("commit delete grade transaction: %w", err)
	}
	return grade, true, nil
}

func (r *GradeRepository) RestoreGrade(ctx context.Context, gradeID, actorUserID int32) (Grade, bool, error) {
	if gradeID <= 0 || actorUserID <= 0 {
		return Grade{}, false, fmt.Errorf("grade id and actor user id are required")
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Grade{}, false, fmt.Errorf("begin restore grade transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	grade, err := scanGrade(tx.QueryRow(
		ctx,
		`UPDATE grades
		 SET deleted_at = NULL, deleted_by_user_id = NULL, delete_reason = NULL
		 WHERE grade_id = $1 AND deleted_at IS NOT NULL
		 RETURNING grade_id, student_id, item_id, teacher_id, score, session_id, comment, created_at, updated_at,
		          deleted_at, deleted_by_user_id, delete_reason`,
		gradeID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Grade{}, false, nil
	}
	if err != nil {
		return Grade{}, false, fmt.Errorf("restore grade: %w", err)
	}
	newScore := grade.Score
	if err := insertGradeEvent(ctx, tx, &grade.ID, grade.ItemID, &grade.StudentID, actorUserID, "grade_restored", nil, &newScore, ""); err != nil {
		return Grade{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Grade{}, false, fmt.Errorf("commit restore grade transaction: %w", err)
	}
	return grade, true, nil
}

func (r *GradeRepository) SoftDeleteGradeItem(ctx context.Context, itemID, actorUserID int32, reason string) (GradeItemDeleteResult, bool, error) {
	if itemID <= 0 || actorUserID <= 0 {
		return GradeItemDeleteResult{}, false, fmt.Errorf("grade item id and actor user id are required")
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return GradeItemDeleteResult{}, false, fmt.Errorf("begin delete grade item transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	item, err := scanGradeItem(tx.QueryRow(
		ctx,
		`UPDATE grade_items
		 SET deleted_at = now(), deleted_by_user_id = $2, delete_reason = NULLIF($3, '')
		 WHERE item_id = $1 AND deleted_at IS NULL
		 RETURNING item_id, subject_id, semester_id, created_by_teacher_id, title, max_score, item_type, deadline,
		          created_at, deleted_at, deleted_by_user_id, delete_reason`,
		itemID,
		actorUserID,
		strings.TrimSpace(reason),
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return GradeItemDeleteResult{}, false, nil
	}
	if err != nil {
		return GradeItemDeleteResult{}, false, fmt.Errorf("soft delete grade item: %w", err)
	}

	rows, err := tx.Query(
		ctx,
		`UPDATE grades
		 SET deleted_at = now(), deleted_by_user_id = $2, delete_reason = NULLIF($3, '')
		 WHERE item_id = $1 AND deleted_at IS NULL
		 RETURNING grade_id, student_id, score`,
		itemID,
		actorUserID,
		strings.TrimSpace(reason),
	)
	if err != nil {
		return GradeItemDeleteResult{}, false, fmt.Errorf("soft delete grades for grade item: %w", err)
	}
	deferredRows := make([]struct {
		id, studentID, score int32
	}, 0)
	for rows.Next() {
		var row struct {
			id, studentID, score int32
		}
		if err := rows.Scan(&row.id, &row.studentID, &row.score); err != nil {
			rows.Close()
			return GradeItemDeleteResult{}, false, fmt.Errorf("scan deleted grade: %w", err)
		}
		deferredRows = append(deferredRows, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return GradeItemDeleteResult{}, false, fmt.Errorf("iterate deleted grades: %w", err)
	}
	rows.Close()

	if err := insertGradeEvent(ctx, tx, nil, item.ID, nil, actorUserID, "grade_item_deleted", nil, nil, reason); err != nil {
		return GradeItemDeleteResult{}, false, err
	}
	for _, row := range deferredRows {
		if err := insertGradeEvent(ctx, tx, &row.id, item.ID, &row.studentID, actorUserID, "grade_deleted", &row.score, nil, reason); err != nil {
			return GradeItemDeleteResult{}, false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return GradeItemDeleteResult{}, false, fmt.Errorf("commit delete grade item transaction: %w", err)
	}
	return GradeItemDeleteResult{Item: item, AffectedGrades: int32(len(deferredRows))}, true, nil
}

func (r *GradeRepository) RestoreGradeItem(ctx context.Context, itemID, actorUserID int32) (GradeItem, bool, error) {
	if itemID <= 0 || actorUserID <= 0 {
		return GradeItem{}, false, fmt.Errorf("grade item id and actor user id are required")
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return GradeItem{}, false, fmt.Errorf("begin restore grade item transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	item, err := scanGradeItem(tx.QueryRow(
		ctx,
		`UPDATE grade_items
		 SET deleted_at = NULL, deleted_by_user_id = NULL, delete_reason = NULL
		 WHERE item_id = $1 AND deleted_at IS NOT NULL
		 RETURNING item_id, subject_id, semester_id, created_by_teacher_id, title, max_score, item_type, deadline,
		          created_at, deleted_at, deleted_by_user_id, delete_reason`,
		itemID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return GradeItem{}, false, nil
	}
	if err != nil {
		return GradeItem{}, false, fmt.Errorf("restore grade item: %w", err)
	}
	if err := insertGradeEvent(ctx, tx, nil, item.ID, nil, actorUserID, "grade_item_restored", nil, nil, ""); err != nil {
		return GradeItem{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return GradeItem{}, false, fmt.Errorf("commit restore grade item transaction: %w", err)
	}
	return item, true, nil
}

func insertGradeEvent(ctx context.Context, tx pgx.Tx, gradeID *int32, itemID int32, studentID *int32, actorUserID int32, eventType string, oldScore, newScore *int32, reason string) error {
	_, err := tx.Exec(
		ctx,
		`INSERT INTO grade_events (grade_id, grade_item_id, student_id, actor_user_id, event_type, old_score, new_score, reason)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''))`,
		gradeID,
		itemID,
		studentID,
		actorUserID,
		eventType,
		oldScore,
		newScore,
		strings.TrimSpace(reason),
	)
	if err != nil {
		return fmt.Errorf("insert grade event: %w", err)
	}
	return nil
}

func (r *GradeRepository) GetStudentGradesBySubject(ctx context.Context, studentID, subjectID, semesterID int32) ([]StudentGradePoint, error) {
	if studentID <= 0 || subjectID <= 0 {
		return nil, fmt.Errorf("student id and subject id are required")
	}
	if semesterID <= 0 {
		return nil, fmt.Errorf("semester id is required")
	}
	rows, err := r.pool.Query(
		ctx,
		`SELECT gi.item_id, gi.title, gi.max_score, gi.item_type, gi.deadline,
		        COALESCE(g.score, 0) AS score, g.updated_at AS graded_at
		 FROM grade_items gi
		 LEFT JOIN grades g ON g.item_id = gi.item_id AND g.student_id = $1 AND g.deleted_at IS NULL
		 WHERE gi.subject_id = $2 AND gi.semester_id = $3 AND gi.deleted_at IS NULL
		 ORDER BY gi.deadline NULLS LAST, gi.created_at, gi.item_id`,
		studentID,
		subjectID,
		semesterID,
	)
	if err != nil {
		return nil, fmt.Errorf("get student grades: %w", err)
	}
	defer rows.Close()

	result := make([]StudentGradePoint, 0)
	for rows.Next() {
		point, err := scanStudentGradePoint(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student grade rows: %w", err)
	}
	return result, nil
}

func scanStudentGradePoint(row pgx.Row) (StudentGradePoint, error) {
	var point StudentGradePoint
	var deadline, gradedAt sql.NullTime
	err := row.Scan(&point.ItemID, &point.Title, &point.MaxScore, &point.ItemType, &deadline, &point.Score, &gradedAt)
	if err != nil {
		return StudentGradePoint{}, fmt.Errorf("scan student grade point: %w", err)
	}
	if deadline.Valid {
		point.Deadline = &deadline.Time
	}
	if gradedAt.Valid {
		point.GradedAt = &gradedAt.Time
	}
	return point, nil
}

func (r *GradeRepository) GetSubjectStatsForPrediction(ctx context.Context, studentID, subjectID, semesterID int32) (PredictionStats, error) {
	if studentID <= 0 || subjectID <= 0 {
		return PredictionStats{}, fmt.Errorf("student id and subject id are required")
	}
	if semesterID <= 0 {
		return PredictionStats{}, fmt.Errorf("semester id is required")
	}
	var stats PredictionStats
	err := r.pool.QueryRow(
		ctx,
		`SELECT COALESCE(SUM(gi.max_score), 0)::INTEGER,
		        COALESCE(SUM(gi.max_score) FILTER (WHERE gi.deadline < now()), 0)::INTEGER,
		        COALESCE(SUM(g.score), 0)::INTEGER
		 FROM grade_items gi
		 LEFT JOIN grades g ON g.item_id = gi.item_id AND g.student_id = $1 AND g.deleted_at IS NULL
		 WHERE gi.subject_id = $2 AND gi.semester_id = $3 AND gi.deleted_at IS NULL`,
		studentID,
		subjectID,
		semesterID,
	).Scan(&stats.TotalMax, &stats.PassedMax, &stats.CurrentScore)
	if err != nil {
		return PredictionStats{}, fmt.Errorf("get prediction stats: %w", err)
	}
	return stats, nil
}

func (r *GradeRepository) GetStudentPerformanceRadar(ctx context.Context, studentID, semesterID int32) ([]SubjectPerformancePoint, error) {
	if studentID <= 0 {
		return nil, fmt.Errorf("student id is required")
	}
	if semesterID <= 0 {
		return nil, fmt.Errorf("semester id is required")
	}
	rows, err := r.pool.Query(
		ctx,
		`WITH student_subjects AS (
		     SELECT DISTINCT sch.subject_id
		     FROM students st
		     JOIN schedules sch ON sch.group_id = st.group_id
		     WHERE st.student_id = $1
		       AND sch.semester_id = $2
		 )
		 SELECT sub.subject_id, sub.name,
		        COALESCE(SUM(g.score) FILTER (WHERE gi.deadline < now()), 0)::INTEGER,
		        COALESCE(SUM(gi.max_score) FILTER (WHERE gi.deadline < now()), 0)::INTEGER
		 FROM student_subjects ss
		 JOIN subjects sub ON sub.subject_id = ss.subject_id
		 LEFT JOIN grade_items gi ON gi.subject_id = sub.subject_id AND gi.semester_id = $2 AND gi.deleted_at IS NULL
		 LEFT JOIN grades g ON g.item_id = gi.item_id AND g.student_id = $1 AND g.deleted_at IS NULL
		 GROUP BY sub.subject_id, sub.name
		 ORDER BY sub.name, sub.subject_id`,
		studentID,
		semesterID,
	)
	if err != nil {
		return nil, fmt.Errorf("get student performance radar: %w", err)
	}
	defer rows.Close()

	result := make([]SubjectPerformancePoint, 0)
	for rows.Next() {
		var point SubjectPerformancePoint
		if err := rows.Scan(&point.SubjectID, &point.SubjectName, &point.Score, &point.MaxScore); err != nil {
			return nil, fmt.Errorf("scan student performance radar row: %w", err)
		}
		if point.MaxScore > 0 {
			point.Percent = int32((float64(point.Score) / float64(point.MaxScore)) * 100)
		}
		result = append(result, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student performance radar rows: %w", err)
	}
	return result, nil
}

func (r *GradeRepository) GetStudentAllSubjectGrades(ctx context.Context, studentID, semesterID int32) ([]StudentSubjectGrades, error) {
	if studentID <= 0 {
		return nil, fmt.Errorf("student id is required")
	}
	if semesterID <= 0 {
		return nil, fmt.Errorf("semester id is required")
	}
	rows, err := r.pool.Query(
		ctx,
		`WITH student_subjects AS (
		     SELECT DISTINCT sch.subject_id
		     FROM students st
		     JOIN schedules sch ON sch.group_id = st.group_id
		     WHERE st.student_id = $1
		       AND sch.semester_id = $2
		 )
		 SELECT sub.subject_id,
		        sub.name,
		        COALESCE(SUM(g.score) FILTER (WHERE gi.deadline < now()) OVER (PARTITION BY sub.subject_id), 0)::INTEGER,
		        COALESCE(SUM(gi.max_score) FILTER (WHERE gi.deadline < now()) OVER (PARTITION BY sub.subject_id), 0)::INTEGER,
		        COALESCE(SUM(g.score) OVER (PARTITION BY sub.subject_id), 0)::INTEGER,
		        COALESCE(SUM(gi.max_score) OVER (PARTITION BY sub.subject_id), 0)::INTEGER,
		        gi.item_id,
		        gi.title,
		        gi.max_score,
		        gi.item_type,
		        gi.deadline,
		        COALESCE(g.score, 0),
		        g.updated_at
		 FROM student_subjects ss
		 JOIN subjects sub ON sub.subject_id = ss.subject_id
		 LEFT JOIN grade_items gi ON gi.subject_id = sub.subject_id AND gi.semester_id = $2 AND gi.deleted_at IS NULL
		 LEFT JOIN grades g ON g.item_id = gi.item_id AND g.student_id = $1 AND g.deleted_at IS NULL
		 ORDER BY sub.name, sub.subject_id, gi.deadline NULLS LAST, gi.created_at, gi.item_id`,
		studentID,
		semesterID,
	)
	if err != nil {
		return nil, fmt.Errorf("get all student subject grades: %w", err)
	}
	defer rows.Close()

	bySubject := make(map[int32]*StudentSubjectGrades)
	order := make([]int32, 0)
	for rows.Next() {
		var subjectID, passedScore, passedMax, currentScore, totalMax int32
		var subjectName string
		var itemID sql.NullInt32
		var title, itemType sql.NullString
		var maxScore sql.NullInt32
		var deadline, gradedAt sql.NullTime
		var score int32
		if err := rows.Scan(&subjectID, &subjectName, &passedScore, &passedMax, &currentScore, &totalMax,
			&itemID, &title, &maxScore, &itemType, &deadline, &score, &gradedAt); err != nil {
			return nil, fmt.Errorf("scan all student subject grades: %w", err)
		}
		subject := bySubject[subjectID]
		if subject == nil {
			subject = &StudentSubjectGrades{SubjectID: subjectID, SubjectName: subjectName, PassedScore: passedScore, PassedMax: passedMax, CurrentScore: currentScore, TotalMax: totalMax, Grades: make([]StudentGradePoint, 0)}
			bySubject[subjectID] = subject
			order = append(order, subjectID)
		}
		if !itemID.Valid {
			continue
		}
		point := StudentGradePoint{ItemID: itemID.Int32, Title: title.String, MaxScore: maxScore.Int32, ItemType: itemType.String, Score: score}
		if deadline.Valid {
			point.Deadline = &deadline.Time
		}
		if gradedAt.Valid {
			point.GradedAt = &gradedAt.Time
		}
		subject.Grades = append(subject.Grades, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all student subject grades: %w", err)
	}
	result := make([]StudentSubjectGrades, 0, len(order))
	for _, subjectID := range order {
		result = append(result, *bySubject[subjectID])
	}
	return result, nil
}
