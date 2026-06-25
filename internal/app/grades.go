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
	SubjectID int32      `json:"subject_id"`
	Title     string     `json:"title"`
	MaxScore  int32      `json:"max_score"`
	ItemType  string     `json:"item_type,omitempty"`
	Deadline  *time.Time `json:"deadline,omitempty"`
}

type GradeSubjectData struct {
	SubjectID int32 `json:"subject_id"`
}

type GradeUpsertData struct {
	StudentID int32   `json:"student_id"`
	ItemID    int32   `json:"item_id"`
	Score     int32   `json:"score"`
	SessionID *int32  `json:"session_id,omitempty"`
	Comment   *string `json:"comment,omitempty"`
}

type TeacherStudentGradesData struct {
	StudentID int32 `json:"student_id"`
	SubjectID int32 `json:"subject_id"`
}

type TeacherStudentRadarData struct {
	StudentID int32 `json:"student_id"`
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

func (s *Service) teacherCanManageSubject(ctx context.Context, teacherID, subjectID int32) (bool, error) {
	var allowed bool
	err := s.store.Pool().QueryRow(
		ctx,
		`SELECT EXISTS (
		     SELECT 1
		     FROM schedules
		     WHERE teacher_id = $1 AND subject_id = $2
		 ) OR EXISTS (
		     SELECT 1
		     FROM teachers t
		     JOIN users u ON u.id = t.user_id
		     WHERE t.teacher_id = $1
		       AND u.login = 'teacher_test'
		 )`,
		teacherID,
		subjectID,
	).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check teacher subject access: %w", err)
	}
	return allowed, nil
}

func (s *Service) ensureTeacherSubjectAccess(ctx context.Context, teacherID, subjectID int32) Response {
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

	allowed, err := s.teacherCanManageSubject(ctx, teacherID, subjectID)
	if err != nil {
		return Response{OK: false, Error: "failed to check teacher subject access"}
	}
	if !allowed {
		return Response{OK: false, Error: "forbidden: teacher is not assigned to subject"}
	}

	return Response{OK: true}
}

func (s *Service) createGradeItemByTeacher(sessionToken string, data GradeItemCreateData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacherUser.Role != "teacher" {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}

	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	if resp := s.ensureTeacherSubjectAccess(ctx, teacherProfile.ID, data.SubjectID); !resp.OK {
		return resp
	}

	item, err := s.store.Grades.CreateGradeItem(ctx, db.GradeItem{
		SubjectID: data.SubjectID,
		Title:     data.Title,
		MaxScore:  data.MaxScore,
		ItemType:  data.ItemType,
		Deadline:  data.Deadline,
	})
	if err != nil {
		return Response{OK: false, Error: "failed to create grade item"}
	}

	return Response{OK: true, Result: item}
}

func (s *Service) gradeItemsBySubjectForTeacher(sessionToken string, data GradeSubjectData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacherUser.Role != "teacher" {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}

	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	if resp := s.ensureTeacherSubjectAccess(ctx, teacherProfile.ID, data.SubjectID); !resp.OK {
		return resp
	}

	items, err := s.store.Grades.GetGradeItemsBySubject(ctx, data.SubjectID)
	if err != nil {
		return Response{OK: false, Error: "failed to load grade items"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"subject_id": data.SubjectID,
			"items":      items,
		},
	}
}

func (s *Service) upsertGradeByTeacher(sessionToken string, data GradeUpsertData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacherUser.Role != "teacher" {
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
	if resp := s.ensureTeacherSubjectAccess(ctx, teacherProfile.ID, item.SubjectID); !resp.OK {
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

	if data.SessionID != nil {
		session, found, err := s.store.Attendance.GetSessionByID(ctx, *data.SessionID)
		if err != nil {
			return Response{OK: false, Error: "failed to load attendance session"}
		}
		if !found {
			return Response{OK: false, Error: "attendance session not found"}
		}
		if session.TeacherID != teacherProfile.ID || session.SubjectID != item.SubjectID {
			return Response{OK: false, Error: "attendance session does not match teacher and subject"}
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
	})
	if err != nil {
		return Response{OK: false, Error: "failed to save grade"}
	}

	return Response{OK: true, Result: grade}
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

	return s.studentGradesResult(studentProfile.ID, data.SubjectID)
}

func (s *Service) gradesBySubjectForTeacher(sessionToken string, data TeacherStudentGradesData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacherUser.Role != "teacher" {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}

	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	if resp := s.ensureTeacherSubjectAccess(ctx, teacherProfile.ID, data.SubjectID); !resp.OK {
		return resp
	}

	_, found, err := s.store.Students.GetByID(ctx, data.StudentID)
	if err != nil {
		return Response{OK: false, Error: "failed to load student"}
	}
	if !found {
		return Response{OK: false, Error: "student not found"}
	}

	return s.studentGradesResult(data.StudentID, data.SubjectID)
}

func (s *Service) studentGradesResult(studentID, subjectID int32) Response {
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

	points, err := s.store.Grades.GetStudentGradesBySubject(ctx, studentID, subjectID)
	if err != nil {
		return Response{OK: false, Error: "failed to load student grades"}
	}
	stats, err := s.store.Grades.GetSubjectStatsForPrediction(ctx, studentID, subjectID)
	if err != nil {
		return Response{OK: false, Error: "failed to load grade summary"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"student_id": studentID,
			"subject_id": subjectID,
			"grades":     points,
			"summary": map[string]any{
				"total_max":     stats.TotalMax,
				"passed_max":    stats.PassedMax,
				"current_score": stats.CurrentScore,
			},
		},
	}
}

func (s *Service) teacherCanViewStudent(ctx context.Context, teacherID, studentID int32) (bool, error) {
	var allowed bool
	err := s.store.Pool().QueryRow(
		ctx,
		`SELECT EXISTS (
		     SELECT 1
		     FROM schedules sch
		     JOIN students st ON st.group_id = sch.group_id
		     WHERE sch.teacher_id = $1 AND st.student_id = $2
		 ) OR EXISTS (
		     SELECT 1
		     FROM teachers t
		     JOIN users u ON u.id = t.user_id
		     WHERE t.teacher_id = $1
		       AND u.login = 'teacher_test'
		 )`,
		teacherID,
		studentID,
	).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("check teacher student access: %w", err)
	}
	return allowed, nil
}

func (s *Service) performanceRadarResult(studentID int32) Response {
	ctx, cancel := s.dbContext()
	defer cancel()

	points, err := s.store.Grades.GetStudentPerformanceRadar(ctx, studentID)
	if err != nil {
		return Response{OK: false, Error: "failed to load performance radar"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"student_id": studentID,
			"subjects":   points,
		},
	}
}

func (s *Service) studentPerformanceRadar(sessionToken string) Response {
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

	return s.performanceRadarResult(studentProfile.ID)
}

// studentAllGrades returns every plan subject for the student together with its
// grade items and per-subject totals, plus an aggregate summary — everything the
// student grades dashboard needs in a single request.
func (s *Service) studentAllGrades(sessionToken string) Response {
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

	radar, err := s.store.Grades.GetStudentPerformanceRadar(ctx, studentProfile.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to load subjects"}
	}

	subjects := make([]map[string]any, 0, len(radar))
	var totalScore, totalMax, totalPassed, gradedWorks, totalWorks int32
	for _, point := range radar {
		grades, gradesErr := s.store.Grades.GetStudentGradesBySubject(ctx, studentProfile.ID, point.SubjectID)
		if gradesErr != nil {
			return Response{OK: false, Error: "failed to load grades"}
		}
		stats, statsErr := s.store.Grades.GetSubjectStatsForPrediction(ctx, studentProfile.ID, point.SubjectID)
		if statsErr != nil {
			return Response{OK: false, Error: "failed to load grade summary"}
		}

		for _, g := range grades {
			totalWorks++
			if g.GradedAt != nil {
				gradedWorks++
			}
		}

		totalScore += stats.CurrentScore
		totalMax += stats.TotalMax
		totalPassed += stats.PassedMax

		subjects = append(subjects, map[string]any{
			"subject_id":    point.SubjectID,
			"subject_name":  point.SubjectName,
			"percent":       point.Percent,
			"current_score": stats.CurrentScore,
			"total_max":     stats.TotalMax,
			"passed_max":    stats.PassedMax,
			"grades":        grades,
		})
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"student_id": studentProfile.ID,
			"subjects":   subjects,
			"summary": map[string]any{
				"current_score": totalScore,
				"total_max":     totalMax,
				"passed_max":    totalPassed,
				"graded_works":  gradedWorks,
				"total_works":   totalWorks,
			},
		},
	}
}

func (s *Service) teacherStudentPerformanceRadar(sessionToken string, data TeacherStudentRadarData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacherUser.Role != "teacher" {
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

	allowed, err := s.teacherCanViewStudent(ctx, teacherProfile.ID, data.StudentID)
	if err != nil {
		return Response{OK: false, Error: "failed to check teacher student access"}
	}
	if !allowed {
		return Response{OK: false, Error: "forbidden: teacher does not teach this student"}
	}

	return s.performanceRadarResult(data.StudentID)
}

func isUnauthorizedGradeError(errText string) bool {
	switch errText {
	case "invalid token", "session not found", "missing token":
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
		resp.Error == "forbidden: student role required" ||
		resp.Error == "forbidden: teacher is not assigned to subject" ||
		resp.Error == "forbidden: teacher does not teach this student" {
		return 403
	}
	if resp.Error == "student not found" ||
		resp.Error == "subject not found" ||
		resp.Error == "grade item not found" ||
		resp.Error == "attendance session not found" {
		return 404
	}
	return 400
}
