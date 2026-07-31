package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const maxAttendanceDistanceMeters = 200.0

type AndroidLessonCreateData struct {
	Subject        string   `json:"subject"`
	SubjectID      int32    `json:"subject_id,omitempty"`
	TeacherID      int32    `json:"teacher_id,omitempty"`
	Groups         []string `json:"groups"`
	GroupIDs       []int32  `json:"group_ids,omitempty"`
	Lat            *float64 `json:"lat"`
	Lon            *float64 `json:"lon"`
	LessonName     string   `json:"lesson_name,omitempty"`
	ExpiresMinutes int      `json:"expires_minutes,omitempty"`
}

type AndroidAttendanceMarkData struct {
	LessonID    int32    `json:"lesson_id"`
	InviteToken string   `json:"invite_token,omitempty"`
	DeviceID    string   `json:"device_id"`
	Lat         *float64 `json:"lat"`
	Lon         *float64 `json:"lon"`
}

func validCoordinates(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func distanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000.0
	toRadians := func(value float64) float64 {
		return value * math.Pi / 180
	}

	latDelta := toRadians(lat2 - lat1)
	lonDelta := toRadians(lon2 - lon1)
	a := math.Sin(latDelta/2)*math.Sin(latDelta/2) +
		math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*
			math.Sin(lonDelta/2)*math.Sin(lonDelta/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func (s *Service) subjectIDForAndroidLesson(ctx context.Context, data AndroidLessonCreateData) (int32, error) {
	if data.SubjectID > 0 {
		_, found, err := s.store.Subjects.GetByID(ctx, data.SubjectID)
		if err != nil {
			return 0, errors.New("failed to load subject")
		}
		if !found {
			return 0, errors.New("subject not found")
		}
		return data.SubjectID, nil
	}

	subjectName := strings.TrimSpace(data.Subject)
	if subjectName == "" {
		return 0, errors.New("subject or subject_id is required")
	}

	var subjectID int32
	err := s.store.Pool().QueryRow(
		ctx,
		`SELECT subject_id
		 FROM subjects
		 WHERE LOWER(name) = LOWER($1)
		 ORDER BY subject_id
		 LIMIT 1`,
		subjectName,
	).Scan(&subjectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, errors.New("subject not found")
	}
	if err != nil {
		return 0, errors.New("failed to load subject")
	}
	return subjectID, nil
}

func (s *Service) groupIDsForAndroidLesson(ctx context.Context, data AndroidLessonCreateData) ([]int32, error) {
	groupIDs := normalizeGroupIDs(data.GroupIDs)
	for _, groupName := range data.Groups {
		groupName = strings.TrimSpace(groupName)
		if groupName == "" {
			continue
		}

		var groupID int32
		err := s.store.Pool().QueryRow(
			ctx,
			`SELECT group_id
			 FROM groups
			 WHERE LOWER(group_name) = LOWER($1)
			 ORDER BY group_id
			 LIMIT 1`,
			groupName,
		).Scan(&groupID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("group not found: %s", groupName)
		}
		if err != nil {
			return nil, errors.New("failed to load group")
		}
		groupIDs = append(groupIDs, groupID)
	}

	groupIDs = normalizeGroupIDs(groupIDs)
	if len(groupIDs) == 0 {
		return nil, errors.New("groups or group_ids are required")
	}
	sort.Slice(groupIDs, func(i, j int) bool { return groupIDs[i] < groupIDs[j] })
	return groupIDs, nil
}

func (s *Service) createLessonForAndroid(sessionToken string, data AndroidLessonCreateData) Response {
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
	if data.TeacherID > 0 && data.TeacherID != teacherProfile.ID {
		return Response{OK: false, Error: "forbidden: teacher_id does not match current user"}
	}
	if data.Lat == nil || data.Lon == nil || !validCoordinates(*data.Lat, *data.Lon) {
		return Response{OK: false, Error: "valid lat and lon are required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	subjectID, err := s.subjectIDForAndroidLesson(ctx, data)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	groupIDs, err := s.groupIDsForAndroidLesson(ctx, data)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	semester, err := s.semesterForWrite(ctx, nil)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if !isDemoTeacher(teacherUser) {
		allowed, err := s.teacherCanManageSubject(ctx, teacherProfile.ID, subjectID, semester.ID)
		if err != nil {
			return Response{OK: false, Error: "failed to check teacher subject access"}
		}
		if !allowed {
			return Response{OK: false, Error: "forbidden: teacher is not assigned to subject"}
		}
	}

	expiresMinutes := normalizeInviteTTL(data.ExpiresMinutes)
	lessonName := strings.TrimSpace(data.LessonName)
	if lessonName == "" {
		lessonName = strings.TrimSpace(data.Subject)
	}
	session, rosterSize, err := s.store.Attendance.CreateSessionWithGroupsAndLocation(
		ctx,
		teacherProfile.ID,
		subjectID,
		semester.ID,
		groupIDs,
		time.Now().Add(time.Duration(expiresMinutes)*time.Minute),
		lessonName,
		data.Lat,
		data.Lon,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to create attendance session"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"id":              session.ID,
			"lesson_id":       session.ID,
			"teacher_id":      teacherProfile.ID,
			"subject_id":      subjectID,
			"semester_id":     semester.ID,
			"group_ids":       groupIDs,
			"roster_size":     rosterSize,
			"expires_at":      formatAPITime(session.ExpiresAt),
			"expires_minutes": expiresMinutes,
		},
	}
}

func (s *Service) attendanceLinkForExistingLessonByTeacher(sessionToken string, lessonID int32) Response {
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

	session, found, err := s.store.Attendance.GetSessionByID(ctx, lessonID)
	if err != nil {
		return Response{OK: false, Error: "failed to load attendance session"}
	}
	if !found {
		return Response{OK: false, Error: "attendance session not found"}
	}
	if session.TeacherID != teacherProfile.ID {
		return Response{OK: false, Error: "forbidden: lesson belongs to another teacher"}
	}
	if time.Now().UTC().After(session.ExpiresAt.UTC()) {
		return Response{OK: false, Error: "attendance session expired"}
	}

	lessonIDText := strconv.FormatInt(int64(session.ID), 10)
	teacherIDText := strconv.FormatInt(int64(session.TeacherID), 10)
	inviteToken, _, err := s.generateAttendanceInviteTokenUntil(lessonIDText, teacherIDText, session.ExpiresAt)
	if err != nil {
		return Response{OK: false, Error: "failed to generate invite token"}
	}

	return Response{
		OK:     true,
		Result: buildAttendanceJoinURL(s.siteBaseURL, inviteToken),
	}
}

func (s *Service) studentAndroidResult(ctx context.Context, user User, isFraud bool) (map[string]any, error) {
	var studentID int32
	var groupName sql.NullString
	var nfcID sql.NullString
	var avatarURL sql.NullString
	var totalCheatAttempts int32
	err := s.store.Pool().QueryRow(
		ctx,
		`SELECT s.student_id, g.group_name, s.nfc_id, s.total_cheat_attempts, u.avatar_url
		 FROM students s
		 JOIN users u ON u.id = $1
		 LEFT JOIN groups g ON g.group_id = s.group_id
		 WHERE s.user_id = $1 OR s.student_id = $1
		 ORDER BY CASE WHEN s.user_id = $1 THEN 0 ELSE 1 END
		 LIMIT 1`,
		user.ID,
	).Scan(&studentID, &groupName, &nfcID, &totalCheatAttempts, &avatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("student profile not found")
	}
	if err != nil {
		return nil, errors.New("failed to load student profile")
	}

	result := map[string]any{
		"user_ID":              user.UserID,
		"student_id":           studentID,
		"login":                user.Login,
		"group":                "",
		"nfc_tag":              "",
		"role":                 user.Role,
		"is_fraud":             isFraud,
		"total_cheat_attempts": totalCheatAttempts,
	}
	if groupName.Valid {
		result["group"] = groupName.String
	}
	if nfcID.Valid {
		result["nfc_tag"] = nfcID.String
	}
	if avatarURL.Valid {
		result["avatar"] = avatarURL.String
	}
	return result, nil
}

func (s *Service) markAttendanceForAndroid(sessionToken string, data AndroidAttendanceMarkData) Response {
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
	if strings.TrimSpace(data.DeviceID) == "" {
		return Response{OK: false, Error: "device_id is required"}
	}
	if data.Lat == nil || data.Lon == nil || !validCoordinates(*data.Lat, *data.Lon) {
		return Response{OK: false, Error: "valid lat and lon are required"}
	}

	var claims *AttendanceInviteClaims
	if strings.TrimSpace(data.InviteToken) != "" {
		claims, err = s.parseAttendanceInviteToken(data.InviteToken)
		if err != nil {
			return Response{OK: false, Error: err.Error()}
		}
		tokenLessonID, err := strconv.ParseInt(claims.LessonID, 10, 32)
		if err != nil {
			return Response{OK: false, Error: "invalid invite token"}
		}
		if data.LessonID > 0 && data.LessonID != int32(tokenLessonID) {
			return Response{OK: false, Error: "lesson_id does not match invite token"}
		}
		data.LessonID = int32(tokenLessonID)
	}
	if data.LessonID <= 0 {
		return Response{OK: false, Error: "lesson_id or invite_token is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	session, found, err := s.store.Attendance.GetSessionByID(ctx, data.LessonID)
	if err != nil {
		return Response{OK: false, Error: "failed to load attendance session"}
	}
	if !found {
		return Response{OK: false, Error: "attendance session not found"}
	}
	if claims != nil && claims.TeacherID != strconv.FormatInt(int64(session.TeacherID), 10) {
		return Response{OK: false, Error: "invite token is not valid"}
	}
	if time.Now().UTC().After(session.ExpiresAt.UTC()) {
		return Response{OK: false, Error: "attendance session expired"}
	}

	fraudReason := ""
	if session.Lat != nil && session.Lon != nil &&
		distanceMeters(*session.Lat, *session.Lon, *data.Lat, *data.Lon) > maxAttendanceDistanceMeters {
		fraudReason = "student is too far from lesson location"
	}
	markResult, err := s.store.Attendance.MarkStudentPresentFromDevice(
		ctx,
		session.ID,
		studentProfile.ID,
		time.Now().UTC(),
		data.DeviceID,
		*data.Lat,
		*data.Lon,
		fraudReason,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to mark attendance"}
	}
	switch markResult {
	case "not_found":
		return Response{OK: false, Error: "forbidden: student is not in session roster"}
	case "already":
		return Response{OK: false, Error: "attendance already confirmed"}
	}

	result, err := s.studentAndroidResult(ctx, studentUser, markResult == "fraud")
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	return Response{OK: true, Result: result}
}

func (s *Service) UpdateAvatarByToken(sessionToken, avatarURL string) Response {
	user, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	avatarURL = strings.TrimSpace(avatarURL)
	if avatarURL == "" {
		return Response{OK: false, Error: "avatar URL is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()
	cmd, err := s.store.Pool().Exec(ctx, `UPDATE users SET avatar_url = $2 WHERE id = $1`, user.ID, avatarURL)
	if err != nil {
		return Response{OK: false, Error: "failed to save avatar URL"}
	}
	if cmd.RowsAffected() == 0 {
		return Response{OK: false, Error: "user not found"}
	}
	return Response{OK: true, Result: avatarURL}
}
