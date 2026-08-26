package app

import (
	"fmt"
	"strings"
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
	Page      int32  `json:"page"`
	PageSize  int32  `json:"page_size"`
	Search    string `json:"search,omitempty"`
	GroupID   int32  `json:"group_id,omitempty"`
	TeacherID int32  `json:"teacher_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
	DateFrom  string `json:"date_from,omitempty"`
	DateTo    string `json:"date_to,omitempty"`
}

type TopCheaterItem struct {
	StudentID          int32      `json:"student_id"`
	StudentName        string     `json:"student_name"`
	GroupName          string     `json:"group_name"`
	TotalCheatAttempts int32      `json:"total_cheat_attempts"`
	LastAttemptAt      *time.Time `json:"last_attempt_at,omitempty"`
}

func fraudFilterWhere(scope VisibilityScope, query FraudLogsQuery) (string, []any) {
	conditions := []string{"ass.is_fraud = TRUE"}
	args := make([]any, 0)
	add := func(template string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(template, len(args)))
	}

	if !scope.All {
		if scope.Role == RoleTeacher {
			add("g.group_id = ANY($%d)", nonNil(scope.GroupIDs))
		} else {
			add("g.lectern_id = ANY($%d)", nonNil(scope.LecternIDs))
		}
	}

	if search := strings.TrimSpace(query.Search); search != "" {
		args = append(args, "%"+search+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		conditions = append(conditions, `(st.student_name ILIKE `+placeholder+` OR
			COALESCE(g.group_name, '') ILIKE `+placeholder+` OR
			COALESCE(sub.name, '') ILIKE `+placeholder+` OR
			COALESCE(t.name, u_t.login, '') ILIKE `+placeholder+` OR
			COALESCE(ass.device_id, '') ILIKE `+placeholder+`)`)
	}
	if query.GroupID > 0 {
		add("g.group_id = $%d", query.GroupID)
	}
	if query.TeacherID > 0 {
		add("s.teacher_id = $%d", query.TeacherID)
	}

	switch strings.TrimSpace(query.Reason) {
	case "distance":
		conditions = append(conditions, "ass.fraud_reason = 'student is too far from lesson location'")
	case "device":
		conditions = append(conditions, "ass.fraud_reason = 'device_id already used in this lesson'")
	case "":
	default:
		add("COALESCE(ass.fraud_reason, '') ILIKE $%d", "%"+strings.TrimSpace(query.Reason)+"%")
	}

	if parsed, err := time.Parse("2006-01-02", strings.TrimSpace(query.DateFrom)); err == nil {
		add("ass.marked_at >= $%d", parsed)
	}
	if parsed, err := time.Parse("2006-01-02", strings.TrimSpace(query.DateTo)); err == nil {
		add("ass.marked_at < $%d", parsed.AddDate(0, 0, 1))
	}

	return strings.Join(conditions, " AND "), args
}

func (s *Service) admin_antifraud_logs(token string, query FraudLogsQuery) Response {
	actor, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if actor.Role != RoleAdmin && actor.Role != RoleMinister && actor.Role != RoleDean && actor.Role != RoleDirector && actor.Role != RoleTeacher {
		return Response{OK: false, Error: "forbidden: staff role required"}
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
	scope, err := s.scopeForUser(ctx, actor)
	if err != nil {
		return Response{OK: false, Error: "failed to resolve scope"}
	}
	filterWhere, filterArgs := fraudFilterWhere(scope, query)

	var total int32
	err = s.store.Pool().QueryRow(
		ctx,
		`SELECT COUNT(*)::INTEGER
		 FROM attendance_session_students ass
		 JOIN students st ON st.student_id = ass.student_id
		 LEFT JOIN groups g ON g.group_id = st.group_id
		 LEFT JOIN attendance_sessions s ON s.session_id = ass.session_id
		 LEFT JOIN subjects sub ON sub.subject_id = s.subject_id
		 LEFT JOIN teachers t ON t.teacher_id = s.teacher_id
		 LEFT JOIN users u_t ON u_t.id = t.user_id
		 WHERE `+filterWhere, filterArgs...,
	).Scan(&total)
	if err != nil {
		return Response{OK: false, Error: "failed to count fraud logs"}
	}

	listArgs := append(append([]any{}, filterArgs...), query.PageSize, offset)
	limitArg := len(filterArgs) + 1
	offsetArg := limitArg + 1
	rows, err := s.store.Pool().Query(
		ctx,
		fmt.Sprintf(`SELECT ass.session_id,
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
		 WHERE %s
		 ORDER BY ass.marked_at DESC, ass.session_id DESC
		 LIMIT $%d OFFSET $%d`, filterWhere, limitArg, offsetArg),
		listArgs...,
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
		); scanErr != nil {
			return Response{OK: false, Error: "failed to scan fraud logs"}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Response{OK: false, Error: "failed to iterate fraud logs"}
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

func (s *Service) admin_antifraud_top_cheaters(token string, query FraudLogsQuery) Response {
	actor, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if actor.Role != RoleAdmin && actor.Role != RoleMinister && actor.Role != RoleDean && actor.Role != RoleDirector && actor.Role != RoleTeacher {
		return Response{OK: false, Error: "forbidden: staff role required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()
	scope, err := s.scopeForUser(ctx, actor)
	if err != nil {
		return Response{OK: false, Error: "failed to resolve scope"}
	}
	filterWhere, args := fraudFilterWhere(scope, query)

	rows, err := s.store.Pool().Query(
		ctx,
		`SELECT st.student_id,
		        st.student_name,
		        COALESCE(g.group_name, ''),
		        COUNT(*)::INTEGER AS total_cheat_attempts,
		        MAX(ass.marked_at) AS last_attempt_at
		 FROM attendance_session_students ass
		 JOIN students st ON st.student_id = ass.student_id
		 LEFT JOIN groups g ON g.group_id = st.group_id
		 JOIN attendance_sessions s ON s.session_id = ass.session_id
		 LEFT JOIN subjects sub ON sub.subject_id = s.subject_id
		 LEFT JOIN teachers t ON t.teacher_id = s.teacher_id
		 LEFT JOIN users u_t ON u_t.id = t.user_id
		 WHERE `+filterWhere+`
		 GROUP BY st.student_id, st.student_name, g.group_name
		 ORDER BY total_cheat_attempts DESC, st.student_name ASC
		 LIMIT 50`, args...,
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
		); scanErr != nil {
			return Response{OK: false, Error: "failed to scan top cheaters"}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Response{OK: false, Error: "failed to iterate top cheaters"}
	}

	return Response{
		OK:     true,
		Result: items,
	}
}
