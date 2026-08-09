package app

import (
	"time"
)

type FraudLogItem struct {
	SessionID   int32     `json:"session_id"`
	StudentID   int32     `json:"student_id"`
	StudentName string    `json:"student_name"`
	GroupName   string    `json:"group_name"`
	SubjectName string    `json:"subject_name"`
	TeacherName string    `json:"teacher_name"`
	DeviceID    string    `json:"device_id"`
	CheckInLat  *float64  `json:"check_in_lat"`
	CheckInLon  *float64  `json:"check_in_lon"`
	FraudReason string    `json:"fraud_reason"`
	MarkedAt    time.Time `json:"marked_at"`
}

type FraudLogsQuery struct {
	Page     int32 `json:"page"`
	PageSize int32 `json:"page_size"`
}

type TopCheaterItem struct {
	StudentID          int32  `json:"student_id"`
	StudentName        string `json:"student_name"`
	GroupName          string `json:"group_name"`
	TotalCheatAttempts int32  `json:"total_cheat_attempts"`
	LastAttemptAt      *time.Time `json:"last_attempt_at,omitempty"`
}

func (s *Service) admin_antifraud_logs(token string, query FraudLogsQuery) Response {
	actor, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if actor.Role != RoleAdmin && actor.Role != RoleMinister && actor.Role != RoleDean && actor.Role != RoleDirector {
		return Response{OK: false, Error: "forbidden: admin or supervisory role required"}
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 20
	}
	offset := (query.Page - 1) * query.PageSize

	ctx, cancel := s.dbContext()
	defer cancel()

	var total int32
	err = s.store.Pool().QueryRow(
		ctx,
		`SELECT COUNT(*)::INTEGER
		 FROM attendance_session_students ass
		 WHERE ass.is_fraud = TRUE`,
	).Scan(&total)
	if err != nil {
		return Response{OK: false, Error: "failed to count fraud logs"}
	}

	rows, err := s.store.Pool().Query(
		ctx,
		`SELECT ass.session_id,
		        ass.student_id,
		        st.student_name,
		        COALESCE(g.group_name, ''),
		        COALESCE(sub.name, ''),
		        COALESCE(t.name, u_t.login, ''),
		        COALESCE(ass.device_id, ''),
		        ass.check_in_lat,
		        ass.check_in_lon,
		        COALESCE(ass.fraud_reason, ''),
		        COALESCE(ass.marked_at, NOW())
		 FROM attendance_session_students ass
		 JOIN students st ON st.student_id = ass.student_id
		 LEFT JOIN groups g ON g.group_id = st.group_id
		 JOIN attendance_sessions s ON s.session_id = ass.session_id
		 LEFT JOIN subjects sub ON sub.subject_id = s.subject_id
		 LEFT JOIN teachers t ON t.teacher_id = s.teacher_id
		 LEFT JOIN users u_t ON u_t.id = t.user_id
		 WHERE ass.is_fraud = TRUE
		 ORDER BY ass.marked_at DESC, ass.session_id DESC
		 LIMIT $1 OFFSET $2`,
		query.PageSize,
		offset,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to fetch fraud logs"}
	}
	defer rows.Close()

	items := make([]FraudLogItem, 0)
	for rows.Next() {
		var item FraudLogItem
		if scanErr := rows.Scan(
			&item.SessionID,
			&item.StudentID,
			&item.StudentName,
			&item.GroupName,
			&item.SubjectName,
			&item.TeacherName,
			&item.DeviceID,
			&item.CheckInLat,
			&item.CheckInLon,
			&item.FraudReason,
			&item.MarkedAt,
		); scanErr == nil {
			items = append(items, item)
		}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"logs":      items,
			"total":     total,
			"page":      query.Page,
			"page_size": query.PageSize,
		},
	}
}

func (s *Service) admin_antifraud_top_cheaters(token string) Response {
	actor, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if actor.Role != RoleAdmin && actor.Role != RoleMinister && actor.Role != RoleDean && actor.Role != RoleDirector {
		return Response{OK: false, Error: "forbidden: admin or supervisory role required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	rows, err := s.store.Pool().Query(
		ctx,
		`SELECT st.student_id,
		        st.student_name,
		        COALESCE(g.group_name, ''),
		        st.total_cheat_attempts,
		        MAX(ass.marked_at) AS last_attempt_at
		 FROM students st
		 LEFT JOIN groups g ON g.group_id = st.group_id
		 LEFT JOIN attendance_session_students ass ON ass.student_id = st.student_id AND ass.is_fraud = TRUE
		 WHERE st.total_cheat_attempts > 0
		 GROUP BY st.student_id, st.student_name, g.group_name, st.total_cheat_attempts
		 ORDER BY st.total_cheat_attempts DESC, st.student_name ASC
		 LIMIT 50`,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to fetch top cheaters"}
	}
	defer rows.Close()

	items := make([]TopCheaterItem, 0)
	for rows.Next() {
		var item TopCheaterItem
		if scanErr := rows.Scan(
			&item.StudentID,
			&item.StudentName,
			&item.GroupName,
			&item.TotalCheatAttempts,
			&item.LastAttemptAt,
		); scanErr == nil {
			items = append(items, item)
		}
	}

	return Response{
		OK:     true,
		Result: items,
	}
}
