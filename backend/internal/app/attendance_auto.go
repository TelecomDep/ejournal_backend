package app

import (
	"context"
	"fmt"
	"math"

	"github.com/TelecomDep/ejournal_backend/internal/db"
)

// updateAutoAttendanceGrades recalculates the attendance_auto grade for a specific student or all students in a subject.
func (s *Service) updateAutoAttendanceGrades(ctx context.Context, subjectID int32, studentID *int32, teacherID int32) error {
	semester, err := s.currentSemester(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve current semester: %w", err)
	}

	// 1. Find the attendance_auto grade item for this subject
	items, err := s.store.Grades.GetGradeItemsBySubject(ctx, subjectID, semester.ID)
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

	// 2. Fetch the students to update
	var targetStudentIDs []int32
	if studentID != nil {
		targetStudentIDs = append(targetStudentIDs, *studentID)
	} else {
		// If studentID is nil, we need to find all students who have a schedule for this subject
		// or all students who have attendance history for this subject.
		// A simple way is to use the store.Pool to get distinct students from attendance_sessions.
		rows, err := s.store.Pool().Query(
			ctx,
			`SELECT DISTINCT ass.student_id 
			 FROM attendance_session_students ass
			 INNER JOIN attendance_sessions sess ON sess.session_id = ass.session_id
			 WHERE sess.subject_id = $1`,
			subjectID,
		)
		if err != nil {
			return fmt.Errorf("failed to query students for subject attendance: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var sid int32
			if err := rows.Scan(&sid); err == nil {
				targetStudentIDs = append(targetStudentIDs, sid)
			}
		}
	}

	// 3. For each student, calculate attendance and upsert the grade
	for _, sid := range targetStudentIDs {
		history, err := s.store.Attendance.GetStudentSubjectAttendanceHistory(ctx, sid, subjectID, semester.ID)
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

		// Calculate score, ceiling it in favor of the student
		ratio := float64(attended) / float64(totalSessions)
		score := int32(math.Ceil(ratio * float64(autoItem.MaxScore)))

		// Ensure it doesn't exceed max_score
		if score > autoItem.MaxScore {
			score = autoItem.MaxScore
		}

		// Upsert grade
		_, err = s.store.Grades.UpsertGrade(ctx, db.Grade{
			StudentID: sid,
			ItemID:    autoItem.ID,
			TeacherID: &teacherID,
			Score:     score,
			Comment:   func(s string) *string { return &s }("Автоматическая оценка за посещаемость"),
		}, teacherID) // We pass teacherID as the actor user ID as well
		if err != nil {
			fmt.Printf("Failed to upsert auto attendance grade for student %d: %v\n", sid, err)
		}
	}

	return nil
}
