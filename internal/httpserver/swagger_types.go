package httpserver

type teacherAttendanceLinkRequest struct {
	SubjectID      *int32  `json:"subject_id,omitempty" example:"2"`
	GroupIDs       []int32 `json:"group_ids,omitempty" example:"1"`
	LessonName     string  `json:"lesson_name,omitempty" example:"Networks"`
	ExpiresMinutes int     `json:"expires_minutes,omitempty" example:"20"`
}

type registerResult struct {
	UserID string `json:"user_id" example:"5"`
	Login  string `json:"login" example:"teacher1"`
	Role   string `json:"role" example:"teacher"`
}

type registerResponse struct {
	ID     string         `json:"id" example:"http-register"`
	OK     bool           `json:"ok" example:"true"`
	Result registerResult `json:"result"`
	Error  string         `json:"error" example:""`
}

type registerByInviteResult struct {
	UserID      string `json:"user_id" example:"12"`
	Login       string `json:"login" example:"student_iks_21"`
	Role        string `json:"role" example:"student"`
	StudentID   int32  `json:"student_id" example:"56"`
	StudentName string `json:"student_name" example:"Демин Сергей А."`
	GroupID     *int32 `json:"group_id,omitempty" example:"237"`
}

type registerByInviteResponse struct {
	ID     string                 `json:"id" example:"http-register-by-invite"`
	OK     bool                   `json:"ok" example:"true"`
	Result registerByInviteResult `json:"result"`
	Error  string                 `json:"error" example:""`
}

type loginResult struct {
	Token  string `json:"token" example:"<jwt_token>"`
	UserID string `json:"user_ID" example:"3"`
	Login  string `json:"login" example:"teacher_test"`
	Role   string `json:"role" example:"teacher"`
}

type loginResponse struct {
	ID     string      `json:"id" example:"http-login"`
	OK     bool        `json:"ok" example:"true"`
	Result loginResult `json:"result"`
	Error  string      `json:"error" example:""`
}

type profileResult struct {
	UserID string `json:"user_id" example:"3"`
	Login  string `json:"login" example:"teacher_test"`
	Role   string `json:"role" example:"teacher"`
}

type profileResponse struct {
	ID     string        `json:"id" example:"http-profile"`
	OK     bool          `json:"ok" example:"true"`
	Result profileResult `json:"result"`
	Error  string        `json:"error" example:""`
}

type teacherAttendanceLinkResult struct {
	LessonID       string  `json:"lesson_id" example:"5"`
	SubjectID      int32   `json:"subject_id" example:"2"`
	LessonName     string  `json:"lesson_name" example:"Networks"`
	InviteToken    string  `json:"invite_token" example:"<attendance_invite_jwt>"`
	URL            string  `json:"url" example:"http://localhost:3000/attendance/join?token=<attendance_invite_jwt>"`
	JoinURL        string  `json:"join_url" example:"http://localhost:3000/attendance/join?token=<attendance_invite_jwt>"`
	QRPayload      string  `json:"qr_payload" example:"http://localhost:3000/attendance/join?token=<attendance_invite_jwt>"`
	GroupIDs       []int32 `json:"group_ids"`
	RosterSize     int32   `json:"roster_size" example:"25"`
	TeacherID      string  `json:"teacher_id" example:"3"`
	ScheduleStart  string  `json:"schedule_start" example:"2026-04-21T21:15:24+07:00"`
	ScheduleEnd    string  `json:"schedule_end" example:"2026-04-21T22:25:24+07:00"`
	Timezone       string  `json:"timezone" example:"Asia/Novosibirsk"`
	ExpiresAt      string  `json:"expires_at" example:"2026-04-21T21:25:37+07:00"`
	ExpiresMinutes int     `json:"expires_minutes" example:"20"`
}

type teacherAttendanceLinkResponse struct {
	ID     string                      `json:"id" example:"http-attendance-link"`
	OK     bool                        `json:"ok" example:"true"`
	Result teacherAttendanceLinkResult `json:"result"`
	Error  string                      `json:"error" example:""`
}

type studentAttendanceConfirmResult struct {
	LessonID   string `json:"lesson_id" example:"5"`
	StudentID  string `json:"student_id" example:"4"`
	TeacherID  string `json:"teacher_id" example:"3"`
	SubjectID  int32  `json:"subject_id" example:"2"`
	MarkedAt   string `json:"marked_at" example:"2026-04-20T20:13:07+07:00"`
	Attendance string `json:"attendance" example:"confirmed"`
}

type studentAttendanceConfirmResponse struct {
	ID     string                         `json:"id" example:"http-attendance-confirm"`
	OK     bool                           `json:"ok" example:"true"`
	Result studentAttendanceConfirmResult `json:"result"`
	Error  string                         `json:"error" example:""`
}

type teacherAttendanceGroupStudent struct {
	StudentID         int32   `json:"student_id" example:"4"`
	StudentName       string  `json:"student_name" example:"Test Student"`
	TotalSessions     int32   `json:"total_sessions" example:"3"`
	AttendedSessions  int32   `json:"attended_sessions" example:"2"`
	AttendancePercent float64 `json:"attendance_percent" example:"66.67"`
	LastMarkedAt      *string `json:"last_marked_at,omitempty" example:"2026-04-20T20:13:07+07:00"`
}

type teacherAttendanceGroupSummary struct {
	StudentsCount int   `json:"students_count" example:"1"`
	SessionsCount int32 `json:"sessions_count" example:"3"`
}

type teacherAttendanceGroupResult struct {
	GroupID   int32                           `json:"group_id" example:"1"`
	SubjectID *int32                          `json:"subject_id,omitempty" example:"2"`
	Timezone  string                          `json:"timezone" example:"Asia/Novosibirsk"`
	Students  []teacherAttendanceGroupStudent `json:"students"`
	Summary   teacherAttendanceGroupSummary   `json:"summary"`
}

type teacherAttendanceGroupResponse struct {
	ID     string                       `json:"id" example:"http-teacher-attendance-group"`
	OK     bool                         `json:"ok" example:"true"`
	Result teacherAttendanceGroupResult `json:"result"`
	Error  string                       `json:"error" example:""`
}
