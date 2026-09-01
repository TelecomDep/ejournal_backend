package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TelecomDep/ejournal_backend/internal/db"
	"github.com/jackc/pgx/v5"
)

type GradeItemCreateData struct {
	SubjectID  int32      `json:"subject_id"`
	SemesterID *int32     `json:"semester_id,omitempty"`
	Title      string     `json:"title"`
	MaxScore   int32      `json:"max_score"`
	ItemType   string     `json:"item_type,omitempty"`
	Deadline   *time.Time `json:"deadline,omitempty"`
}

type GradeSubjectData struct {
	SubjectID  int32  `json:"subject_id"`
	SemesterID *int32 `json:"semester_id,omitempty"`
}

type GradeUpsertData struct {
	StudentID int32   `json:"student_id"`
	ItemID    int32   `json:"item_id"`
	Score     int32   `json:"score"`
	SessionID *int32  `json:"session_id,omitempty"`
	Comment   *string `json:"comment,omitempty"`
}

type TeacherStudentGradesData struct {
	StudentID  int32  `json:"student_id"`
	SubjectID  int32  `json:"subject_id"`
	SemesterID *int32 `json:"semester_id,omitempty"`
}

type TeacherStudentRadarData struct {
	StudentID  int32  `json:"student_id"`
	SemesterID *int32 `json:"semester_id,omitempty"`
}

type GradeDeleteData struct {
	GradeID int32  `json:"grade_id"`
	Reason  string `json:"reason,omitempty"`
}

type GradeItemDeleteData struct {
	ItemID  int32  `json:"item_id"`
	Cascade bool   `json:"cascade,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type GradeRestoreData struct {
	GradeID int32 `json:"grade_id"`
}

type GradeItemRestoreData struct {
	ItemID int32 `json:"item_id"`
}

func (s *Service) studentProfileByUser(user User) (db.Student, error) {
	ctx, cancel := s.dbContext()
	defer cancel()

	var out db.Student
	err := s.store.Pool().QueryRow(
		ctx,
		`SELECT student_id, student_name, group_id
		 FROM students
		 WHERE user_id = $1 OR student_id = $1
		 ORDER BY CASE WHEN user_id = $1 THEN 0 ELSE 1 END
		 LIMIT 1`,
		user.ID,
	).Scan(&out.ID, &out.StudentName, &out.GroupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Student{}, errors.New("student profile not found")
	}
	if err != nil {
		return db.Student{}, errors.New("failed to load student profile")
	}

	return out, nil
}

func (s *Service) teacherCanManageSubject(ctx context.Context, teacherID, subjectID, semesterID int32) (bool, error) {
	var allowed bool
	err := s.store.Pool().QueryRow(
		ctx,
		`SELECT EXISTS (
		     SELECT 1
		     FROM schedules
		     WHERE teacher_id = $1 AND subject_id = $2 AND semester_id = $3
		 )`,
		teacherID,
		subjectID,
		semesterID,
	).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check teacher subject access: %w", err)
	}
	return allowed, nil
}

func (s *Service) teacherCanAccessGroup(ctx context.Context, teacherID, groupID, semesterID int32, subjectID *int32) (bool, error) {
	query := `SELECT EXISTS (
		SELECT 1 FROM schedules
		WHERE teacher_id = $1 AND group_id = $2 AND semester_id = $3`
	args := []any{teacherID, groupID, semesterID}
	if subjectID != nil {
		query += ` AND subject_id = $4`
		args = append(args, *subjectID)
	}
	query += `)`

	var allowed bool
	if err := s.store.Pool().QueryRow(ctx, query, args...).Scan(&allowed); err != nil {
		return false, fmt.Errorf("check teacher group access: %w", err)
	}
	return allowed, nil
}

func (s *Service) teacherCanAccessStudentSubject(ctx context.Context, teacherID, studentID, subjectID, semesterID int32) (bool, error) {
	var allowed bool
	err := s.store.Pool().QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM schedules sch
			JOIN students st ON st.group_id = sch.group_id
			WHERE sch.teacher_id = $1
			  AND st.student_id = $2
			  AND sch.subject_id = $3
			  AND sch.semester_id = $4
		)`, teacherID, studentID, subjectID, semesterID).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check teacher student subject access: %w", err)
	}
	return allowed, nil
}

func (s *Service) ensureTeacherStudentSubjectAccess(ctx context.Context, teacherID, studentID, subjectID, semesterID int32) Response {
	allowed, err := s.teacherCanAccessStudentSubject(ctx, teacherID, studentID, subjectID, semesterID)
	if err != nil {
		return Response{OK: false, Error: "failed to check teacher student access"}
	}
	if !allowed {
		return Response{OK: false, Error: "forbidden: student is not assigned to this teacher and subject"}
	}
	return Response{OK: true}
}

func (s *Service) ensureTeacherSubjectAccess(ctx context.Context, teacherID, subjectID, semesterID int32) Response {
	if subjectID <= 0 {
		return Response{OK: false, Error: "subject_id is required"}
	}

	_, found, err := s.store.Subjects.GetByID(ctx, subjectID)
	if err != nil {
		return Response{OK: false, Error: "failed to load subject"}
	}
	if !found {
		return Response{OK: false, Error: "subject not found"}
	}

	allowed, err := s.teacherCanManageSubject(ctx, teacherID, subjectID, semesterID)
	if err != nil {
		return Response{OK: false, Error: "failed to check teacher subject access"}
	}
	if !allowed {
		return Response{OK: false, Error: "forbidden: teacher is not assigned to subject"}
	}

	return Response{OK: true}
}

func (s *Service) ensureSemesterWriteAccess(ctx context.Context, semesterID int32) Response {
	semester, err := s.semesterByID(ctx, semesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if err := semesterWriteError(semester, time.Now().UTC()); err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	return Response{OK: true}
}

func (s *Service) createGradeItemByTeacher(sessionToken string, data GradeItemCreateData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if !isTeachingRole(teacherUser.Role) {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}

	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	semester, err := s.semesterForWrite(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if resp := s.ensureTeacherSubjectAccess(ctx, teacherProfile.ID, data.SubjectID, semester.ID); !resp.OK {
		return resp
	}
	if data.Deadline != nil {
		deadline := data.Deadline.UTC()
		if deadline.Before(semester.StartsAt.UTC()) || deadline.After(semester.EndsAt.UTC()) {
			return Response{OK: false, Error: "deadline must be within semester date range"}
		}
	}

	item, err := s.store.Grades.CreateGradeItem(ctx, db.GradeItem{
		SubjectID:          data.SubjectID,
		SemesterID:         semester.ID,
		CreatedByTeacherID: &teacherProfile.ID,
		Title:              data.Title,
		MaxScore:           data.MaxScore,
		ItemType:           data.ItemType,
		Deadline:           data.Deadline,
	}, teacherUser.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to create grade item"}
	}

	if item.ItemType == "attendance_auto" {
		_ = s.updateAutoAttendanceGrades(ctx, item.SubjectID, item.SemesterID, nil, teacherProfile.ID)
	}

	return Response{OK: true, Result: item}
}

func (s *Service) gradeItemsBySubjectForTeacher(sessionToken string, data GradeSubjectData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if !isTeachingRole(teacherUser.Role) {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}

	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	semester, err := s.semesterForOptionalID(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if resp := s.ensureTeacherSubjectAccess(ctx, teacherProfile.ID, data.SubjectID, semester.ID); !resp.OK {
		return resp
	}

	items, err := s.store.Grades.GetGradeItemsBySubject(ctx, data.SubjectID, semester.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to load grade items"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"subject_id":  data.SubjectID,
			"semester_id": semester.ID,
			"items":       items,
		},
	}
}

func (s *Service) upsertGradeByTeacher(sessionToken string, data GradeUpsertData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if !isTeachingRole(teacherUser.Role) {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}

	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	item, found, err := s.store.Grades.GetGradeItemByID(ctx, data.ItemID)
	if err != nil {
		return Response{OK: false, Error: "failed to load grade item"}
	}
	if !found {
		return Response{OK: false, Error: "grade item not found"}
	}
	if resp := s.ensureSemesterWriteAccess(ctx, item.SemesterID); !resp.OK {
		return resp
	}
	if resp := s.ensureTeacherSubjectAccess(ctx, teacherProfile.ID, item.SubjectID, item.SemesterID); !resp.OK {
		return resp
	}

	if data.Score < 0 {
		return Response{OK: false, Error: "score must be non-negative"}
	}
	if data.Score > item.MaxScore {
		return Response{OK: false, Error: "score exceeds grade item max_score"}
	}

	_, found, err = s.store.Students.GetByID(ctx, data.StudentID)
	if err != nil {
		return Response{OK: false, Error: "failed to load student"}
	}
	if !found {
		return Response{OK: false, Error: "student not found"}
	}
	if resp := s.ensureTeacherStudentSubjectAccess(ctx, teacherProfile.ID, data.StudentID, item.SubjectID, item.SemesterID); !resp.OK {
		return resp
	}

	if data.SessionID != nil {
		session, found, err := s.store.Attendance.GetSessionByID(ctx, *data.SessionID)
		if err != nil {
			return Response{OK: false, Error: "failed to load attendance session"}
		}
		if !found {
			return Response{OK: false, Error: "attendance session not found"}
		}
		if session.TeacherID != teacherProfile.ID ||
			session.SubjectID != item.SubjectID ||
			session.SemesterID != item.SemesterID {
			return Response{OK: false, Error: "attendance session does not match teacher, subject and semester"}
		}
	}

	comment := data.Comment
	if comment != nil {
		trimmed := strings.TrimSpace(*comment)
		if trimmed == "" {
			comment = nil
		} else {
			comment = &trimmed
		}
	}
	teacherID := teacherProfile.ID

	grade, err := s.store.Grades.UpsertGrade(ctx, db.Grade{
		StudentID: data.StudentID,
		ItemID:    data.ItemID,
		TeacherID: &teacherID,
		Score:     data.Score,
		SessionID: data.SessionID,
		Comment:   comment,
	}, teacherUser.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to save grade"}
	}

	event_type := "grade_created"

	if !grade.CreatedAt.Equal(grade.UpdatedAt) {
		event_type = "grade_updated"
	}

	_ = s.create_grade_notification(
		ctx,
		grade,
		item,
		event_type,
	)

	return Response{OK: true, Result: grade}
}

func (s *Service) ensureGradeItemOwnerAccess(ctx context.Context, user User, item db.GradeItem) Response {
	if user.Role == RoleAdmin {
		return Response{OK: true}
	}
	if !isTeachingRole(user.Role) {
		return Response{OK: false, Error: "forbidden: teacher or admin role required"}
	}
	if item.CreatedByTeacherID == nil {
		return Response{OK: false, Error: "forbidden: grade item has no recorded owner"}
	}

	teacher, err := s.teacherProfileByUser(user)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if *item.CreatedByTeacherID != teacher.ID {
		return Response{OK: false, Error: "forbidden: only the grade item owner can modify it"}
	}
	return s.ensureTeacherSubjectAccess(ctx, teacher.ID, item.SubjectID, item.SemesterID)
}

func (s *Service) deleteGradeByTeacher(sessionToken string, data GradeDeleteData) Response {
	if data.GradeID <= 0 {
		return Response{OK: false, Error: "grade_id is required"}
	}
	user, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	ctx, cancel := s.dbContext()
	defer cancel()

	grade, found, err := s.store.Grades.GetGradeByID(ctx, data.GradeID)
	if err != nil {
		return Response{OK: false, Error: "failed to load grade"}
	}
	if !found {
		return Response{OK: false, Error: "grade not found"}
	}
	item, found, err := s.store.Grades.GetGradeItemByIDIncludingDeleted(ctx, grade.ItemID)
	if err != nil {
		return Response{OK: false, Error: "failed to load grade item"}
	}
	if !found {
		return Response{OK: false, Error: "grade item not found"}
	}
	if resp := s.ensureSemesterWriteAccess(ctx, item.SemesterID); !resp.OK {
		return resp
	}
	if resp := s.ensureGradeItemOwnerAccess(ctx, user, item); !resp.OK {
		return resp
	}
	if grade.DeletedAt != nil {
		return Response{OK: false, Error: "grade is already deleted"}
	}

	deleted, ok, err := s.store.Grades.SoftDeleteGrade(ctx, grade.ID, user.ID, data.Reason)
	if err != nil {
		return Response{OK: false, Error: "failed to delete grade"}
	}
	if !ok {
		return Response{OK: false, Error: "grade is already deleted"}
	}
	return Response{OK: true, Result: deleted}
}

func (s *Service) restoreGradeByTeacher(sessionToken string, data GradeRestoreData) Response {
	if data.GradeID <= 0 {
		return Response{OK: false, Error: "grade_id is required"}
	}
	user, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	ctx, cancel := s.dbContext()
	defer cancel()

	grade, found, err := s.store.Grades.GetGradeByID(ctx, data.GradeID)
	if err != nil {
		return Response{OK: false, Error: "failed to load grade"}
	}
	if !found {
		return Response{OK: false, Error: "grade not found"}
	}
	item, found, err := s.store.Grades.GetGradeItemByIDIncludingDeleted(ctx, grade.ItemID)
	if err != nil {
		return Response{OK: false, Error: "failed to load grade item"}
	}
	if !found {
		return Response{OK: false, Error: "grade item not found"}
	}
	if resp := s.ensureSemesterWriteAccess(ctx, item.SemesterID); !resp.OK {
		return resp
	}
	if resp := s.ensureGradeItemOwnerAccess(ctx, user, item); !resp.OK {
		return resp
	}
	if item.DeletedAt != nil {
		return Response{OK: false, Error: "grade item is deleted; restore it first"}
	}
	if grade.DeletedAt == nil {
		return Response{OK: false, Error: "grade is not deleted"}
	}

	restored, ok, err := s.store.Grades.RestoreGrade(ctx, grade.ID, user.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to restore grade"}
	}
	if !ok {
		return Response{OK: false, Error: "grade is not deleted"}
	}
	return Response{OK: true, Result: restored}
}

func (s *Service) deleteGradeItemByTeacher(sessionToken string, data GradeItemDeleteData) Response {
	if data.ItemID <= 0 {
		return Response{OK: false, Error: "item_id is required"}
	}
	user, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	ctx, cancel := s.dbContext()
	defer cancel()

	item, found, err := s.store.Grades.GetGradeItemByIDIncludingDeleted(ctx, data.ItemID)
	if err != nil {
		return Response{OK: false, Error: "failed to load grade item"}
	}
	if !found {
		return Response{OK: false, Error: "grade item not found"}
	}
	if resp := s.ensureSemesterWriteAccess(ctx, item.SemesterID); !resp.OK {
		return resp
	}
	if resp := s.ensureGradeItemOwnerAccess(ctx, user, item); !resp.OK {
		return resp
	}
	if item.DeletedAt != nil {
		return Response{OK: false, Error: "grade item is already deleted"}
	}
	activeGrades, err := s.store.Grades.CountActiveGradesByItem(ctx, item.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to count grades"}
	}
	if activeGrades > 0 && !data.Cascade {
		return Response{OK: false, Error: "grade item has active grades; set cascade=true to delete"}
	}

	deleted, ok, err := s.store.Grades.SoftDeleteGradeItem(ctx, item.ID, user.ID, data.Reason)
	if err != nil {
		return Response{OK: false, Error: "failed to delete grade item"}
	}
	if !ok {
		return Response{OK: false, Error: "grade item is already deleted"}
	}
	return Response{OK: true, Result: map[string]any{
		"item":            deleted.Item,
		"affected_grades": deleted.AffectedGrades,
	}}
}

func (s *Service) restoreGradeItemByTeacher(sessionToken string, data GradeItemRestoreData) Response {
	if data.ItemID <= 0 {
		return Response{OK: false, Error: "item_id is required"}
	}
	user, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	ctx, cancel := s.dbContext()
	defer cancel()

	item, found, err := s.store.Grades.GetGradeItemByIDIncludingDeleted(ctx, data.ItemID)
	if err != nil {
		return Response{OK: false, Error: "failed to load grade item"}
	}
	if !found {
		return Response{OK: false, Error: "grade item not found"}
	}
	if resp := s.ensureSemesterWriteAccess(ctx, item.SemesterID); !resp.OK {
		return resp
	}
	if resp := s.ensureGradeItemOwnerAccess(ctx, user, item); !resp.OK {
		return resp
	}
	if item.DeletedAt == nil {
		return Response{OK: false, Error: "grade item is not deleted"}
	}

	restored, ok, err := s.store.Grades.RestoreGradeItem(ctx, item.ID, user.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to restore grade item"}
	}
	if !ok {
		return Response{OK: false, Error: "grade item is not deleted"}
	}

	if item.ItemType == "attendance_auto" {
		if teacher, err := s.teacherProfileByUser(user); err == nil {
			_ = s.updateAutoAttendanceGrades(ctx, item.SubjectID, item.SemesterID, nil, teacher.ID)
		}
	}

	return Response{OK: true, Result: restored}
}

func (s *Service) gradesBySubjectForStudent(sessionToken string, data GradeSubjectData) Response {
	studentUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if studentUser.Role != "student" {
		return Response{OK: false, Error: "forbidden: student role required"}
	}

	studentProfile, err := s.studentProfileByUser(studentUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	return s.studentGradesResult(studentProfile.ID, data.SubjectID, data.SemesterID)
}

func (s *Service) gradesBySubjectForTeacher(sessionToken string, data TeacherStudentGradesData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if !isTeachingRole(teacherUser.Role) {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}

	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	semester, err := s.semesterForOptionalID(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if resp := s.ensureTeacherSubjectAccess(ctx, teacherProfile.ID, data.SubjectID, semester.ID); !resp.OK {
		return resp
	}

	_, found, err := s.store.Students.GetByID(ctx, data.StudentID)
	if err != nil {
		return Response{OK: false, Error: "failed to load student"}
	}
	if !found {
		return Response{OK: false, Error: "student not found"}
	}
	if resp := s.ensureTeacherStudentSubjectAccess(ctx, teacherProfile.ID, data.StudentID, data.SubjectID, semester.ID); !resp.OK {
		return resp
	}

	return s.studentGradesResult(data.StudentID, data.SubjectID, &semester.ID)
}

func (s *Service) studentGradesResult(studentID, subjectID int32, semesterID *int32) Response {
	if subjectID <= 0 {
		return Response{OK: false, Error: "subject_id is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	_, found, err := s.store.Subjects.GetByID(ctx, subjectID)
	if err != nil {
		return Response{OK: false, Error: "failed to load subject"}
	}
	if !found {
		return Response{OK: false, Error: "subject not found"}
	}

	semester, err := s.semesterForOptionalID(ctx, semesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	points, err := s.store.Grades.GetStudentGradesBySubject(ctx, studentID, subjectID, semester.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to load student grades"}
	}
	stats, err := s.store.Grades.GetSubjectStatsForPrediction(ctx, studentID, subjectID, semester.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to load grade summary"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"student_id":  studentID,
			"subject_id":  subjectID,
			"semester_id": semester.ID,
			"semester":    semesterToMap(semester),
			"grades":      points,
			"summary": map[string]any{
				"total_max":     stats.TotalMax,
				"passed_max":    stats.PassedMax,
				"current_score": stats.CurrentScore,
			},
		},
	}
}

func (s *Service) teacherCanViewStudent(ctx context.Context, teacherID, studentID, semesterID int32) (bool, error) {
	var allowed bool
	err := s.store.Pool().QueryRow(
		ctx,
		`SELECT EXISTS (
		     SELECT 1
		     FROM schedules sch
		     JOIN students st ON st.group_id = sch.group_id
		     WHERE sch.teacher_id = $1 AND st.student_id = $2 AND sch.semester_id = $3
		 )`,
		teacherID,
		studentID,
		semesterID,
	).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check teacher student access: %w", err)
	}
	return allowed, nil
}

func (s *Service) performanceRadarResult(studentID int32, semester db.Semester) Response {
	ctx, cancel := s.dbContext()
	defer cancel()

	points, err := s.store.Grades.GetStudentPerformanceRadar(ctx, studentID, semester.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to load performance radar"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"student_id":  studentID,
			"semester_id": semester.ID,
			"semester":    semesterToMap(semester),
			"subjects":    points,
		},
	}
}

func (s *Service) studentPerformanceRadar(sessionToken string, data SemesterSelectionData) Response {
	studentUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if studentUser.Role != "student" {
		return Response{OK: false, Error: "forbidden: student role required"}
	}

	studentProfile, err := s.studentProfileByUser(studentUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	semester, err := s.semesterForOptionalID(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	return s.performanceRadarResult(studentProfile.ID, semester)
}

// studentAllGrades returns every plan subject for the student together with its
// grade items, grade totals, and attendance totals, plus an aggregate summary.
func (s *Service) studentAllGrades(sessionToken string, data SemesterSelectionData) Response {
	studentUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if studentUser.Role != "student" {
		return Response{OK: false, Error: "forbidden: student role required"}
	}

	studentProfile, err := s.studentProfileByUser(studentUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	semester, err := s.semesterForOptionalID(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	allSubjects, err := s.store.Grades.GetStudentAllSubjectGrades(ctx, studentProfile.ID, semester.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to load subjects"}
	}
	attendanceRows, err := s.store.Attendance.GetStudentAttendanceBySubject(ctx, studentProfile.ID, semester.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to load attendance summary"}
	}
	attendanceBySubject := attendanceMetricsBySubject(attendanceRows)

	subjects := make([]map[string]any, 0, len(allSubjects))
	var totalScore, totalMax, totalPassed, gradedWorks, totalWorks int32
	var totalSessions, attendedSessions, excusedSessions int32
	for _, subject := range allSubjects {
		for _, g := range subject.Grades {
			totalWorks++
			if g.GradedAt != nil {
				gradedWorks++
			}
		}

		totalScore += subject.CurrentScore
		totalMax += subject.TotalMax
		totalPassed += subject.PassedMax
		percent := int32(0)
		if subject.PassedMax > 0 {
			percent = int32((float64(subject.PassedScore) / float64(subject.PassedMax)) * 100)
		}

		attendance := attendanceBySubject[subject.SubjectID]
		totalSessions += attendance.TotalSessions
		attendedSessions += attendance.AttendedSessions
		excusedSessions += attendance.ExcusedSessions

		subjectResult := map[string]any{
			"subject_id":    subject.SubjectID,
			"subject_name":  subject.SubjectName,
			"percent":       percent,
			"current_score": subject.CurrentScore,
			"total_max":     subject.TotalMax,
			"passed_max":    subject.PassedMax,
			"grades":        subject.Grades,
		}
		for key, value := range attendance.toMap() {
			subjectResult[key] = value
		}
		subjects = append(subjects, subjectResult)
	}
	overallAttendance := makeStudentAttendanceMetrics(totalSessions, attendedSessions, excusedSessions)

	return Response{
		OK: true,
		Result: map[string]any{
			"student_id":  studentProfile.ID,
			"semester_id": semester.ID,
			"semester":    semesterToMap(semester),
			"subjects":    subjects,
			"summary": map[string]any{
				"current_score":      totalScore,
				"total_max":          totalMax,
				"passed_max":         totalPassed,
				"graded_works":       gradedWorks,
				"total_works":        totalWorks,
				"total_sessions":     overallAttendance.TotalSessions,
				"attended_sessions":  overallAttendance.AttendedSessions,
				"excused_sessions":   overallAttendance.ExcusedSessions,
				"missed_sessions":    overallAttendance.MissedSessions,
				"attendance_percent": overallAttendance.Percent,
			},
		},
	}
}

func (s *Service) teacherStudentPerformanceRadar(sessionToken string, data TeacherStudentRadarData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if !isTeachingRole(teacherUser.Role) {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}
	if data.StudentID <= 0 {
		return Response{OK: false, Error: "student_id is required"}
	}

	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	_, found, err := s.store.Students.GetByID(ctx, data.StudentID)
	if err != nil {
		return Response{OK: false, Error: "failed to load student"}
	}
	if !found {
		return Response{OK: false, Error: "student not found"}
	}

	semester, err := s.semesterForOptionalID(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	allowed, err := s.teacherCanViewStudent(ctx, teacherProfile.ID, data.StudentID, semester.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to check teacher student access"}
	}
	if !allowed {
		return Response{OK: false, Error: "forbidden: teacher does not teach this student"}
	}

	return s.performanceRadarResult(data.StudentID, semester)
}

func isUnauthorizedGradeError(errText string) bool {
	switch errText {
	case "invalid token", "session not found", "session revoked", "missing token", "account is not active":
		return true
	default:
		return false
	}
}

func GradeHTTPStatus(resp Response) int {
	if resp.OK {
		return 200
	}
	if isUnauthorizedGradeError(resp.Error) {
		return 401
	}
	if resp.Error == "forbidden: teacher role required" ||
		resp.Error == "forbidden: teacher or admin role required" ||
		resp.Error == "forbidden: grade item has no recorded owner" ||
		resp.Error == "forbidden: only the grade item owner can modify it" ||
		resp.Error == "forbidden: student role required" ||
		resp.Error == "forbidden: teacher is not assigned to subject" ||
		resp.Error == "forbidden: teacher does not teach this student" {
		return 403
	}
	if resp.Error == "student not found" ||
		resp.Error == "subject not found" ||
		resp.Error == "grade not found" ||
		resp.Error == "grade item not found" ||
		resp.Error == "attendance session not found" ||
		resp.Error == "semester not found" ||
		resp.Error == "open semester not found" {
		return 404
	}
	if resp.Error == "grade item has active grades; set cascade=true to delete" ||
		resp.Error == "grade is already deleted" ||
		resp.Error == "grade item is already deleted" ||
		resp.Error == "grade is not deleted" ||
		resp.Error == "grade item is not deleted" ||
		resp.Error == "semester is not open for changes" ||
		resp.Error == "semester has not started" ||
		resp.Error == "semester has ended" ||
		resp.Error == "deadline must be within semester date range" ||
		resp.Error == "attendance session does not match teacher, subject and semester" {
		return 409
	}
	return 400
}
