package app

import (
	"context"
	"fmt"
	"math"

	"github.com/TelecomDep/ejournal_backend/internal/db"
)

// updateAutoAttendanceGrades recalculates attendance points within one semester.
func (s *Service) updateAutoAttendanceGrades(
	ctx context.Context,
	subjectID, semesterID int32,
	studentID *int32,
	teacherID int32,
) error {
	if semesterID <= 0 {
		return fmt.Errorf("semester id is required")
	}

	items, err := s.store.Grades.GetGradeItemsBySubject(ctx, subjectID, semesterID)
	if err != nil {
		return fmt.Errorf("failed to get grade items: %w", err)
	}

	var autoItem *db.GradeItem
	for _, it := range items {
		if it.ItemType == "attendance_auto" {
			// Make a copy of the loop variable
			itCopy := it
			autoItem = &itCopy
			break
		}
	}

	// If there's no auto attendance item, do nothing.
	if autoItem == nil {
		return nil
	}

	var targetStudentIDs []int32
	if studentID != nil {
		targetStudentIDs = append(targetStudentIDs, *studentID)
	} else {
		rows, err := s.store.Pool().Query(
			ctx,
			`SELECT DISTINCT ass.student_id
			 FROM attendance_session_students ass
			 INNER JOIN attendance_sessions sess ON sess.session_id = ass.session_id
			 WHERE sess.subject_id = $1
			   AND sess.semester_id = $2`,
			subjectID,
			semesterID,
		)
		if err != nil {
			return fmt.Errorf("failed to query students for subject attendance: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var sid int32
			if err := rows.Scan(&sid); err != nil {
				return fmt.Errorf("scan student for attendance grade: %w", err)
			}
			targetStudentIDs = append(targetStudentIDs, sid)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate students for attendance grade: %w", err)
		}
	}

	var actorUserID int32
	err = s.store.Pool().QueryRow(
		ctx,
		`SELECT COALESCE(user_id, teacher_id)
		 FROM teachers
		 WHERE teacher_id = $1`,
		teacherID,
	).Scan(&actorUserID)
	if err != nil {
		return fmt.Errorf("resolve attendance grade actor: %w", err)
	}

	for _, sid := range targetStudentIDs {
		history, err := s.store.Attendance.GetStudentSubjectAttendanceHistory(ctx, sid, subjectID, semesterID)
		if err != nil {
			continue // Skip on error
		}

		totalSessions := len(history)
		if totalSessions == 0 {
			continue
		}

		attended := 0
		for _, h := range history {
			if h.Status == "present" || h.Status == "late" {
				attended++
			}
		}

		ratio := float64(attended) / float64(totalSessions)
		score := int32(math.Ceil(ratio * float64(autoItem.MaxScore)))

		if score > autoItem.MaxScore {
			score = autoItem.MaxScore
		}

		_, err = s.store.Grades.UpsertGrade(ctx, db.Grade{
			StudentID: sid,
			ItemID:    autoItem.ID,
			TeacherID: &teacherID,
			Score:     score,
			Comment:   func(s string) *string { return &s }("Автоматическая оценка за посещаемость"),
		}, actorUserID)
		if err != nil {
			fmt.Printf("Failed to upsert auto attendance grade for student %d: %v\n", sid, err)
		}
	}

	return nil
}
