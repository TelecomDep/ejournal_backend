package app

import (
	"math"

	"github.com/TelecomDep/ejournal_backend/internal/db"
)

type studentAttendanceMetrics struct {
	TotalSessions    int32
	AttendedSessions int32
	ExcusedSessions  int32
	MissedSessions   int32
	Percent          float64
}

func makeStudentAttendanceMetrics(total, attended, excused int32) studentAttendanceMetrics {
	missed := total - attended - excused
	if missed < 0 {
		missed = 0
	}

	countedSessions := total - excused
	percent := 0.0
	if countedSessions > 0 {
		percent = math.Round(float64(attended)*10000/float64(countedSessions)) / 100
	}

	return studentAttendanceMetrics{
		TotalSessions:    total,
		AttendedSessions: attended,
		ExcusedSessions:  excused,
		MissedSessions:   missed,
		Percent:          percent,
	}
}

func (metrics studentAttendanceMetrics) toMap() map[string]any {
	return map[string]any{
		"total_sessions":     metrics.TotalSessions,
		"attended_sessions":  metrics.AttendedSessions,
		"excused_sessions":   metrics.ExcusedSessions,
		"missed_sessions":    metrics.MissedSessions,
		"attendance_percent": metrics.Percent,
	}
}

func attendanceMetricsBySubject(rows []db.StudentSubjectAttendanceStat) map[int32]studentAttendanceMetrics {
	result := make(map[int32]studentAttendanceMetrics, len(rows))
	for _, row := range rows {
		result[row.SubjectID] = makeStudentAttendanceMetrics(
			row.TotalSessions,
			row.AttendedSessions,
			row.ExcusedSessions,
		)
	}
	return result
}

func (s *Service) studentAttendanceSummary(sessionToken string, data SemesterSelectionData) Response {
	studentUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if studentUser.Role != RoleStudent {
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

	rows, err := s.store.Attendance.GetStudentAttendanceBySubject(ctx, studentProfile.ID, semester.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to load attendance summary"}
	}

	subjects := make([]map[string]any, 0, len(rows))
	var totalSessions, attendedSessions, excusedSessions int32
	for _, row := range rows {
		metrics := makeStudentAttendanceMetrics(row.TotalSessions, row.AttendedSessions, row.ExcusedSessions)
		subject := metrics.toMap()
		subject["subject_id"] = row.SubjectID
		subject["subject_name"] = row.SubjectName
		subjects = append(subjects, subject)

		totalSessions += row.TotalSessions
		attendedSessions += row.AttendedSessions
		excusedSessions += row.ExcusedSessions
	}

	summary := makeStudentAttendanceMetrics(totalSessions, attendedSessions, excusedSessions).toMap()
	return Response{
		OK: true,
		Result: map[string]any{
			"student_id":  studentProfile.ID,
			"semester_id": semester.ID,
			"semester":    semesterToMap(semester),
			"subjects":    subjects,
			"summary":     summary,
		},
	}
}
