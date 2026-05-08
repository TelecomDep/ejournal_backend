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

func NewGradeRepository(pool *pgxpool.Pool) *GradeRepository {
	return &GradeRepository{pool: pool}
}

func (r *GradeRepository) CreateGradeItem(ctx context.Context, item GradeItem) (GradeItem, error) {
	item.Title = strings.TrimSpace(item.Title)
	item.ItemType = strings.TrimSpace(item.ItemType)
	if item.SubjectID <= 0 {
		return GradeItem{}, fmt.Errorf("subject id is required")
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

	var out GradeItem
	var deadline sql.NullTime
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO grade_items (subject_id, title, max_score, item_type, deadline)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING item_id, subject_id, title, max_score, item_type, deadline, created_at`,
		item.SubjectID,
		item.Title,
		item.MaxScore,
		item.ItemType,
		item.Deadline,
	).Scan(&out.ID, &out.SubjectID, &out.Title, &out.MaxScore, &out.ItemType, &deadline, &out.CreatedAt)
	if err != nil {
		return GradeItem{}, fmt.Errorf("create grade item: %w", err)
	}
	if deadline.Valid {
		out.Deadline = &deadline.Time
	}

	return out, nil
}

func (r *GradeRepository) GetGradeItemByID(ctx context.Context, itemID int32) (GradeItem, bool, error) {
	if itemID <= 0 {
		return GradeItem{}, false, fmt.Errorf("grade item id is required")
	}

	var out GradeItem
	var deadline sql.NullTime
	err := r.pool.QueryRow(
		ctx,
		`SELECT item_id, subject_id, title, max_score, item_type, deadline, created_at
		 FROM grade_items
		 WHERE item_id = $1`,
		itemID,
	).Scan(&out.ID, &out.SubjectID, &out.Title, &out.MaxScore, &out.ItemType, &deadline, &out.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return GradeItem{}, false, nil
	}
	if err != nil {
		return GradeItem{}, false, fmt.Errorf("get grade item by id: %w", err)
	}
	if deadline.Valid {
		out.Deadline = &deadline.Time
	}

	return out, true, nil
}

func (r *GradeRepository) GetGradeItemsBySubject(ctx context.Context, subjectID int32) ([]GradeItem, error) {
	if subjectID <= 0 {
		return nil, fmt.Errorf("subject id is required")
	}

	rows, err := r.pool.Query(
		ctx,
		`SELECT item_id, subject_id, title, max_score, item_type, deadline, created_at
		 FROM grade_items
		 WHERE subject_id = $1
		 ORDER BY deadline NULLS LAST, created_at, item_id`,
		subjectID,
	)
	if err != nil {
		return nil, fmt.Errorf("query grade items: %w", err)
	}
	defer rows.Close()

	result := make([]GradeItem, 0)
	for rows.Next() {
		var item GradeItem
		var deadline sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.SubjectID,
			&item.Title,
			&item.MaxScore,
			&item.ItemType,
			&deadline,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan grade item: %w", err)
		}
		if deadline.Valid {
			item.Deadline = &deadline.Time
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate grade item rows: %w", err)
	}

	return result, nil
}

func (r *GradeRepository) UpsertGrade(ctx context.Context, grade Grade) (Grade, error) {
	if grade.StudentID <= 0 {
		return Grade{}, fmt.Errorf("student id is required")
	}
	if grade.ItemID <= 0 {
		return Grade{}, fmt.Errorf("grade item id is required")
	}
	if grade.Score < 0 {
		return Grade{}, fmt.Errorf("score must be non-negative")
	}

	var out Grade
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO grades (student_id, item_id, teacher_id, score, session_id, comment)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (student_id, item_id)
		 DO UPDATE SET
		     score = EXCLUDED.score,
		     teacher_id = EXCLUDED.teacher_id,
		     session_id = EXCLUDED.session_id,
		     comment = EXCLUDED.comment
		 RETURNING grade_id, student_id, item_id, teacher_id, score, session_id, comment, created_at, updated_at`,
		grade.StudentID,
		grade.ItemID,
		grade.TeacherID,
		grade.Score,
		grade.SessionID,
		grade.Comment,
	).Scan(
		&out.ID,
		&out.StudentID,
		&out.ItemID,
		&out.TeacherID,
		&out.Score,
		&out.SessionID,
		&out.Comment,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return Grade{}, fmt.Errorf("upsert grade: %w", err)
	}

	return out, nil
}

func (r *GradeRepository) GetStudentGradesBySubject(ctx context.Context, studentID, subjectID int32) ([]StudentGradePoint, error) {
	if studentID <= 0 {
		return nil, fmt.Errorf("student id is required")
	}
	if subjectID <= 0 {
		return nil, fmt.Errorf("subject id is required")
	}

	rows, err := r.pool.Query(
		ctx,
		`SELECT gi.item_id,
		        gi.title,
		        gi.max_score,
		        gi.item_type,
		        gi.deadline,
		        COALESCE(g.score, 0) AS score,
		        g.updated_at AS graded_at
		 FROM grade_items gi
		 LEFT JOIN grades g ON g.item_id = gi.item_id AND g.student_id = $1
		 WHERE gi.subject_id = $2
		 ORDER BY gi.deadline NULLS LAST, gi.created_at, gi.item_id`,
		studentID,
		subjectID,
	)
	if err != nil {
		return nil, fmt.Errorf("get student grades: %w", err)
	}
	defer rows.Close()

	result := make([]StudentGradePoint, 0)
	for rows.Next() {
		var item StudentGradePoint
		var deadline sql.NullTime
		var gradedAt sql.NullTime
		if err := rows.Scan(
			&item.ItemID,
			&item.Title,
			&item.MaxScore,
			&item.ItemType,
			&deadline,
			&item.Score,
			&gradedAt,
		); err != nil {
			return nil, fmt.Errorf("scan student grade point: %w", err)
		}
		if deadline.Valid {
			item.Deadline = &deadline.Time
		}
		if gradedAt.Valid {
			item.GradedAt = &gradedAt.Time
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student grade rows: %w", err)
	}

	return result, nil
}

func (r *GradeRepository) GetSubjectStatsForPrediction(ctx context.Context, studentID, subjectID int32) (PredictionStats, error) {
	if studentID <= 0 {
		return PredictionStats{}, fmt.Errorf("student id is required")
	}
	if subjectID <= 0 {
		return PredictionStats{}, fmt.Errorf("subject id is required")
	}

	var stats PredictionStats
	err := r.pool.QueryRow(
		ctx,
		`SELECT COALESCE(SUM(gi.max_score), 0)::INTEGER AS total_max,
		        COALESCE(SUM(CASE WHEN gi.deadline < now() THEN gi.max_score ELSE 0 END), 0)::INTEGER AS passed_max,
		        COALESCE(SUM(g.score), 0)::INTEGER AS current_score
		 FROM grade_items gi
		 LEFT JOIN grades g ON g.item_id = gi.item_id AND g.student_id = $1
		 WHERE gi.subject_id = $2`,
		studentID,
		subjectID,
	).Scan(&stats.TotalMax, &stats.PassedMax, &stats.CurrentScore)
	if err != nil {
		return PredictionStats{}, fmt.Errorf("get prediction stats: %w", err)
	}

	return stats, nil
}
