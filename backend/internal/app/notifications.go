package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/TelecomDep/ejournal_backend/internal/db"
)

const (
	notification_category_grades     = "grades"
	notification_category_schedule   = "schedule"
	notification_category_attendance = "attendance"
	notification_category_system     = "system"
)

type NotificationsListData struct {
	Page       int32  `json:"page"`
	PageSize   int32  `json:"page_size"`
	Category   string `json:"category"`
	UnreadOnly bool   `json:"unread_only"`
}

type NotificationIDData struct {
	NotificationID int64 `json:"notification_id"`
}

type NotificationSettingsData struct {
	Grades     bool `json:"grades"`
	Schedule   bool `json:"schedule"`
	Attendance bool `json:"attendance"`
	System     bool `json:"system"`
}

type AdminNotificationCreateData struct {
	Category  string     `json:"category,omitempty"`
	EventType string     `json:"event_type,omitempty"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	Audience  string     `json:"audience"`
	Role      string     `json:"role,omitempty"`
	UserIDs   []int32    `json:"user_ids,omitempty"`
	GroupIDs  []int32    `json:"group_ids,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type AdminNotificationUpdateData struct {
	NotificationID int64      `json:"notification_id"`
	Title          *string    `json:"title,omitempty"`
	Message        *string    `json:"message,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

func valid_notification_category(category string) bool {
	switch category {
	case notification_category_grades,
		notification_category_schedule,
		notification_category_attendance,
		notification_category_system:
		return true
	default:
		return false
	}
}

func valid_notification_event(category string, event_type string) bool {
	switch category {
	case notification_category_grades:
		return event_type == "grade_created" || event_type == "grade_updated"

	case notification_category_schedule:
		return event_type == "lesson_created" ||
			event_type == "lesson_rescheduled" ||
			event_type == "lesson_cancelled"

	case notification_category_attendance:
		return event_type == "attendance_opened" ||
			event_type == "attendance_marked" ||
			event_type == "attendance_rejected"

	case notification_category_system:
		return event_type == "fraud" || event_type == "admin_update"

	default:
		return false
	}
}

func normalize_notifications_page(data NotificationsListData) NotificationsListData {
	if data.Page <= 0 {
		data.Page = 1
	}

	if data.PageSize <= 0 {
		data.PageSize = 20
	}

	if data.PageSize > 100 {
		data.PageSize = 100
	}

	data.Category = strings.ToLower(strings.TrimSpace(data.Category))

	return data
}

func (s *Service) notifications_list(
	session_token string,
	data NotificationsListData,
) Response {
	user, err := s.userBySessionToken(session_token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	data = normalize_notifications_page(data)

	if data.Category != "" && !valid_notification_category(data.Category) {
		return Response{OK: false, Error: "invalid notification category"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	result, err := s.store.Notifications.ListForUser(
		ctx,
		user.ID,
		db.NotificationListFilter{
			Page:       data.Page,
			PageSize:   data.PageSize,
			Category:   data.Category,
			UnreadOnly: data.UnreadOnly,
		},
	)
	if err != nil {
		return Response{OK: false, Error: "failed to load notifications"}
	}

	pages := int32(0)

	if result.Total > 0 {
		pages = (result.Total + data.PageSize - 1) / data.PageSize
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"items":        result.Items,
			"unread_count": result.UnreadCount,
			"pagination": map[string]any{
				"page":      data.Page,
				"page_size": data.PageSize,
				"total":     result.Total,
				"pages":     pages,
			},
		},
	}
}

func (s *Service) notifications_unread_count(session_token string) Response {
	user, err := s.userBySessionToken(session_token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	count, err := s.store.Notifications.UnreadCount(ctx, user.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to count unread notifications"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"unread_count": count,
		},
	}
}

func (s *Service) notification_mark_read(
	session_token string,
	notification_id int64,
) Response {
	user, err := s.userBySessionToken(session_token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	if notification_id <= 0 {
		return Response{OK: false, Error: "notification_id is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	found, err := s.store.Notifications.MarkRead(ctx, user.ID, notification_id)
	if err != nil {
		return Response{OK: false, Error: "failed to mark notification as read"}
	}

	if !found {
		return Response{OK: false, Error: "notification not found"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"notification_id": notification_id,
			"is_read":         true,
		},
	}
}

func (s *Service) notifications_mark_all_read(session_token string) Response {
	user, err := s.userBySessionToken(session_token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	updated, err := s.store.Notifications.MarkAllRead(ctx, user.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to mark notifications as read"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"updated": updated,
		},
	}
}

func (s *Service) notification_delete(
	session_token string,
	notification_id int64,
) Response {
	user, err := s.userBySessionToken(session_token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	if notification_id <= 0 {
		return Response{OK: false, Error: "notification_id is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	deleted, err := s.store.Notifications.DeleteForUser(ctx, user.ID, notification_id)
	if err != nil {
		return Response{OK: false, Error: "failed to delete notification"}
	}

	if !deleted {
		return Response{OK: false, Error: "notification not found"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"notification_id": notification_id,
			"deleted":         true,
		},
	}
}

func (s *Service) notification_settings_get(session_token string) Response {
	user, err := s.userBySessionToken(session_token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	settings, err := s.store.Notifications.GetSettings(ctx, user.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to load notification settings"}
	}

	return Response{OK: true, Result: settings}
}

func (s *Service) notification_settings_update(
	session_token string,
	data NotificationSettingsData,
) Response {
	user, err := s.userBySessionToken(session_token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	settings, err := s.store.Notifications.UpdateSettings(
		ctx,
		db.NotificationSettings{
			UserID:     user.ID,
			Grades:     data.Grades,
			Schedule:   data.Schedule,
			Attendance: data.Attendance,
			System:     data.System,
		},
	)
	if err != nil {
		return Response{OK: false, Error: "failed to update notification settings"}
	}

	return Response{OK: true, Result: settings}
}

func (s *Service) admin_notifications_create(
	session_token string,
	data AdminNotificationCreateData,
) Response {
	admin_user, auth := s.require_admin(session_token)
	if !auth.OK {
		return auth
	}

	data.Title = strings.TrimSpace(data.Title)
	data.Message = strings.TrimSpace(data.Message)
	data.Category = strings.ToLower(strings.TrimSpace(data.Category))
	data.EventType = strings.ToLower(strings.TrimSpace(data.EventType))
	data.Audience = strings.ToLower(strings.TrimSpace(data.Audience))
	data.Role = strings.ToLower(strings.TrimSpace(data.Role))

	if data.Category == "" {
		data.Category = notification_category_system
	}

	if data.EventType == "" && data.Category == notification_category_system {
		data.EventType = "admin_update"
	}

	if !valid_notification_category(data.Category) {
		return Response{OK: false, Error: "invalid notification category"}
	}

	if !valid_notification_event(data.Category, data.EventType) {
		return Response{OK: false, Error: "invalid notification event_type"}
	}

	if data.Title == "" {
		return Response{OK: false, Error: "title is required"}
	}

	if len(data.Title) > 255 {
		return Response{OK: false, Error: "title is too long"}
	}

	if data.Message == "" {
		return Response{OK: false, Error: "message is required"}
	}

	if len(data.Message) > 10000 {
		return Response{OK: false, Error: "message is too long"}
	}

	if data.Audience == "" {
		data.Audience = "all"
	}

	if data.Audience != "all" &&
		data.Audience != "role" &&
		data.Audience != "users" &&
		data.Audience != "groups" {
		return Response{OK: false, Error: "invalid notification audience"}
	}

	if data.Audience == "role" && !valid_admin_role(data.Role) {
		return Response{OK: false, Error: "invalid user role"}
	}

	if data.Audience == "users" && len(data.UserIDs) == 0 {
		return Response{OK: false, Error: "user_ids are required"}
	}

	if data.Audience == "groups" && len(data.GroupIDs) == 0 {
		return Response{OK: false, Error: "group_ids are required"}
	}

	if data.ExpiresAt != nil && data.ExpiresAt.Before(time.Now().UTC()) {
		return Response{OK: false, Error: "expires_at must be in the future"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	metadata, _ := json.Marshal(map[string]any{
		"source": "admin",
	})

	item, err := s.store.Notifications.Create(
		ctx,
		db.NotificationCreate{
			Category:  data.Category,
			EventType: data.EventType,
			Title:     data.Title,
			Message:   data.Message,
			CreatedBy: &admin_user.ID,
			Metadata:  metadata,
			ExpiresAt: data.ExpiresAt,
		},
		db.NotificationRecipients{
			Audience: data.Audience,
			Role:     data.Role,
			UserIDs:  data.UserIDs,
			GroupIDs: data.GroupIDs,
		},
	)
	if err != nil {
		if strings.Contains(err.Error(), "no recipients") {
			return Response{OK: false, Error: "notification has no recipients"}
		}

		return Response{OK: false, Error: "failed to create notification"}
	}

	return Response{OK: true, Result: item}
}

func (s *Service) admin_notifications_list(
	session_token string,
	data NotificationsListData,
) Response {
	if _, auth := s.require_admin(session_token); !auth.OK {
		return auth
	}

	data = normalize_notifications_page(data)

	if data.Category != "" && !valid_notification_category(data.Category) {
		return Response{OK: false, Error: "invalid notification category"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	items, total, err := s.store.Notifications.AdminList(
		ctx,
		db.NotificationListFilter{
			Page:     data.Page,
			PageSize: data.PageSize,
			Category: data.Category,
		},
	)
	if err != nil {
		return Response{OK: false, Error: "failed to load notifications"}
	}

	pages := int32(0)

	if total > 0 {
		pages = (total + data.PageSize - 1) / data.PageSize
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"items": items,
			"pagination": map[string]any{
				"page":      data.Page,
				"page_size": data.PageSize,
				"total":     total,
				"pages":     pages,
			},
		},
	}
}

func (s *Service) admin_notifications_update(
	session_token string,
	data AdminNotificationUpdateData,
) Response {
	if _, auth := s.require_admin(session_token); !auth.OK {
		return auth
	}

	if data.NotificationID <= 0 {
		return Response{OK: false, Error: "notification_id is required"}
	}

	if data.Title == nil && data.Message == nil && data.ExpiresAt == nil {
		return Response{OK: false, Error: "nothing to update"}
	}

	if data.Title != nil {
		value := strings.TrimSpace(*data.Title)

		if value == "" {
			return Response{OK: false, Error: "title cannot be empty"}
		}

		if len(value) > 255 {
			return Response{OK: false, Error: "title is too long"}
		}

		data.Title = &value
	}

	if data.Message != nil {
		value := strings.TrimSpace(*data.Message)

		if value == "" {
			return Response{OK: false, Error: "message cannot be empty"}
		}

		if len(value) > 10000 {
			return Response{OK: false, Error: "message is too long"}
		}

		data.Message = &value
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	item, found, err := s.store.Notifications.AdminUpdate(
		ctx,
		data.NotificationID,
		data.Title,
		data.Message,
		data.ExpiresAt,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to update notification"}
	}

	if !found {
		return Response{OK: false, Error: "notification not found"}
	}

	return Response{OK: true, Result: item}
}

func (s *Service) admin_notifications_delete(
	session_token string,
	notification_id int64,
) Response {
	if _, auth := s.require_admin(session_token); !auth.OK {
		return auth
	}

	if notification_id <= 0 {
		return Response{OK: false, Error: "notification_id is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	deleted, err := s.store.Notifications.AdminDelete(ctx, notification_id)
	if err != nil {
		return Response{OK: false, Error: "failed to delete notification"}
	}

	if !deleted {
		return Response{OK: false, Error: "notification not found"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"notification_id": notification_id,
			"deleted":         true,
		},
	}
}

func notification_metadata(data map[string]any) json.RawMessage {
	metadata, err := json.Marshal(data)
	if err != nil {
		return json.RawMessage(`{}`)
	}

	return metadata
}

func (s *Service) create_grade_notification(
	ctx context.Context,
	grade db.Grade,
	item db.GradeItem,
	event_type string,
) error {
	if !valid_notification_event(notification_category_grades, event_type) {
		return errors.New("invalid grade notification event")
	}

	title := "Выставлена новая оценка"

	if event_type == "grade_updated" {
		title = "Оценка обновлена"
	}

	_, err := s.store.Notifications.Create(
		ctx,
		db.NotificationCreate{
			Category:  notification_category_grades,
			EventType: event_type,
			Title:     title,
			Message:   "Изменение по работе: " + item.Title,
			Metadata: notification_metadata(map[string]any{
				"grade_id":   grade.ID,
				"student_id": grade.StudentID,
				"item_id":    item.ID,
				"subject_id": item.SubjectID,
				"score":      grade.Score,
				"max_score":  item.MaxScore,
			}),
		},
		db.NotificationRecipients{
			Audience:   "students",
			StudentIDs: []int32{grade.StudentID},
		},
	)

	return err
}

func (s *Service) create_attendance_opened_notification(
	ctx context.Context,
	session db.AttendanceSession,
	group_ids []int32,
) error {
	_, err := s.store.Notifications.Create(
		ctx,
		db.NotificationCreate{
			Category:  notification_category_attendance,
			EventType: "attendance_opened",
			Title:     "Открыта отметка посещаемости",
			Message:   "Можно отметить присутствие на занятии.",
			Metadata: notification_metadata(map[string]any{
				"lesson_id":  session.ID,
				"subject_id": session.SubjectID,
				"expires_at": session.ExpiresAt,
			}),
			ExpiresAt: &session.ExpiresAt,
		},
		db.NotificationRecipients{
			Audience: "groups",
			GroupIDs: group_ids,
		},
	)

	return err
}

func (s *Service) create_attendance_result_notification(
	ctx context.Context,
	student_id int32,
	session db.AttendanceSession,
	accepted bool,
) error {
	event_type := "attendance_rejected"
	title := "Отметка посещаемости отклонена"
	message := "Не удалось подтвердить присутствие на занятии."

	if accepted {
		event_type = "attendance_marked"
		title = "Посещаемость отмечена"
		message = "Присутствие на занятии успешно подтверждено."
	}

	_, err := s.store.Notifications.Create(
		ctx,
		db.NotificationCreate{
			Category:  notification_category_attendance,
			EventType: event_type,
			Title:     title,
			Message:   message,
			Metadata: notification_metadata(map[string]any{
				"lesson_id":  session.ID,
				"student_id": student_id,
			}),
		},
		db.NotificationRecipients{
			Audience:   "students",
			StudentIDs: []int32{student_id},
		},
	)

	return err
}

func (s *Service) create_fraud_notification(
	ctx context.Context,
	student_id int32,
	session db.AttendanceSession,
	reason string,
) error {
	reason = strings.TrimSpace(reason)

	if reason == "" {
		reason = "подозрительная попытка отметки посещаемости"
	}

	_, err := s.store.Notifications.Create(
		ctx,
		db.NotificationCreate{
			Category:  notification_category_system,
			EventType: "fraud",
			Title:     "Обнаружена подозрительная отметка",
			Message:   "Система отклонила отметку посещаемости.",
			Metadata: notification_metadata(map[string]any{
				"lesson_id":    session.ID,
				"student_id":   student_id,
				"teacher_id":   session.TeacherID,
				"fraud_reason": reason,
			}),
		},
		db.NotificationRecipients{
			Audience:   "fraud",
			StudentIDs: []int32{student_id},
			TeacherID:  session.TeacherID,
		},
	)

	return err
}
