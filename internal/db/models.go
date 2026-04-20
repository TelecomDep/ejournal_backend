package db

import "time"

type User struct {
	ID           int32     `json:"id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Lectern struct {
	ID   int32  `json:"lectern_id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type ControlType struct {
	ID   int32  `json:"type_id"`
	Name string `json:"type_name"`
}

type Subject struct {
	ID           int32  `json:"subject_id"`
	SubjectIndex string `json:"subject_index"`
	Name         string `json:"name"`
	InPlan       bool   `json:"in_plan"`
	LecternID    *int32 `json:"lectern_id,omitempty"`
}

type SubjectMetric struct {
	SubjectID      int32  `json:"subject_id"`
	ZetExpert      *int32 `json:"zet_expert,omitempty"`
	ZetFact        *int32 `json:"zet_fact,omitempty"`
	HoursExpert    *int32 `json:"hours_expert,omitempty"`
	HoursByPlan    *int32 `json:"hours_by_plan,omitempty"`
	HoursContrWork *int32 `json:"hours_contr_work,omitempty"`
	HoursAuditory  *int32 `json:"hours_auditory,omitempty"`
	HoursSelfStudy *int32 `json:"hours_self_study,omitempty"`
	HoursControl   *int32 `json:"hours_control,omitempty"`
	HoursPrep      *int32 `json:"hours_prep,omitempty"`
}

type SemesterLoad struct {
	ID          int32    `json:"load_id"`
	SubjectID   int32    `json:"subject_id"`
	SemesterNum int32    `json:"semester_num"`
	ZetValue    *float64 `json:"zet_value,omitempty"`
}

type SubjectControl struct {
	ID          int32 `json:"control_id"`
	SubjectID   int32 `json:"subject_id"`
	TypeID      int32 `json:"type_id"`
	SemesterNum int32 `json:"semester_num"`
}

type Teacher struct {
	ID        int32  `json:"teacher_id"`
	UserID    *int32 `json:"user_id,omitempty"`
	Name      string `json:"name"`
	LecternID *int32 `json:"lectern_id,omitempty"`
	JobTitle  string `json:"job_title"`
}

type Group struct {
	ID        int32  `json:"group_id"`
	GroupName string `json:"group_name"`
	LecternID *int32 `json:"lectern_id,omitempty"`
}

type Student struct {
	ID          int32  `json:"student_id"`
	StudentName string `json:"student_name"`
	GroupID     *int32 `json:"group_id,omitempty"`
}

type AttendanceSession struct {
	ID        int32     `json:"session_id"`
	TeacherID int32     `json:"teacher_id"`
	SubjectID int32     `json:"subject_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type AttendanceMark struct {
	SessionID int32     `json:"session_id"`
	StudentID int32     `json:"student_id"`
	MarkedAt  time.Time `json:"marked_at"`
}

type AttendanceGroupStat struct {
	StudentID        int32      `json:"student_id"`
	StudentName      string     `json:"student_name"`
	TotalSessions    int32      `json:"total_sessions"`
	AttendedSessions int32      `json:"attended_sessions"`
	LastMarkedAt     *time.Time `json:"last_marked_at,omitempty"`
}
