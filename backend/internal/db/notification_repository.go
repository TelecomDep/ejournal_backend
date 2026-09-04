package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	pool *pgxpool.Pool
}

type Notification struct {
	NotificationID int64           `json:"notification_id"`
	Category       string          `json:"category"`
	EventType      string          `json:"event_type"`
	Title          string          `json:"title"`
	Message        string          `json:"message"`
	CreatedBy      *int32          `json:"created_by_user_id,omitempty"`
	Metadata       json.RawMessage `json:"metadata"`
	ExpiresAt      *time.Time      `json:"expires_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	ReadAt         *time.Time      `json:"read_at,omitempty"`
	IsRead         bool            `json:"is_read"`
	RecipientCount int32           `json:"recipient_count,omitempty"`
}

type NotificationCreate struct {
	Category  string
	EventType string
	Title     string
	Message   string
	CreatedBy *int32
	Metadata  json.RawMessage
	ExpiresAt *time.Time
}

type NotificationRecipients struct {
	Audience   string
	Role       string
	UserIDs    []int32
	StudentIDs []int32
	GroupIDs   []int32
	TeacherID  int32
}

type NotificationListFilter struct {
	Page       int32
	PageSize   int32
	Category   string
	UnreadOnly bool
}

type NotificationListResult struct {
	Items       []Notification
	Total       int32
	UnreadCount int32
}

type NotificationSettings struct {
	UserID     int32     `json:"user_id"`
	Grades     bool      `json:"grades"`
	Schedule   bool      `json:"schedule"`
	Attendance bool      `json:"attendance"`
	System     bool      `json:"system"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{
		pool: pool,
	}
}

func (r *NotificationRepository) Create(
	ctx context.Context,
	data NotificationCreate,
	recipients NotificationRecipients,
) (Notification, error) {
	data.Category = strings.TrimSpace(data.Category)
	data.EventType = strings.TrimSpace(data.EventType)
	data.Title = strings.TrimSpace(data.Title)
	data.Message = strings.TrimSpace(data.Message)

	if data.Category == "" || data.EventType == "" {
		return Notification{}, fmt.Errorf("notification category and event_type are required")
	}

	if data.Title == "" || data.Message == "" {
		return Notification{}, fmt.Errorf("notification title and message are required")
	}

	if len(data.Metadata) == 0 {
		data.Metadata = json.RawMessage(`{}`)
	}

	if !json.Valid(data.Metadata) {
		return Notification{}, fmt.Errorf("notification metadata is invalid")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Notification{}, fmt.Errorf("begin notification transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var item Notification
	var metadata []byte

	err = tx.QueryRow(
		ctx,
		`INSERT INTO notifications (
		     category,
		     event_type,
		     title,
		     message,
		     created_by_user_id,
		     metadata,
		     expires_at
		 )
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING notification_id,
		           category,
		           event_type,
		           title,
		           message,
		           created_by_user_id,
		           metadata,
		           expires_at,
		           created_at`,
		data.Category,
		data.EventType,
		data.Title,
		data.Message,
		data.CreatedBy,
		data.Metadata,
		data.ExpiresAt,
	).Scan(
		&item.NotificationID,
		&item.Category,
		&item.EventType,
		&item.Title,
		&item.Message,
		&item.CreatedBy,
		&metadata,
		&item.ExpiresAt,
		&item.CreatedAt,
	)
	if err != nil {
		return Notification{}, fmt.Errorf("create notification: %w", err)
	}

	item.Metadata = json.RawMessage(metadata)

	var recipient_count int64

	switch recipients.Audience {
	case "all":
		tag, exec_err := tx.Exec(
			ctx,
			`INSERT INTO notification_recipients (notification_id, user_id)
			 SELECT $1, id
			 FROM users
			 WHERE status = 'active'
			 ON CONFLICT DO NOTHING`,
			item.NotificationID,
		)

		err = exec_err
		recipient_count = tag.RowsAffected()

	case "role":
		tag, exec_err := tx.Exec(
			ctx,
			`INSERT INTO notification_recipients (notification_id, user_id)
			 SELECT DISTINCT $1, u.id
			 FROM users u
			 JOIN user_roles ur ON ur.user_id = u.id
			 WHERE ur.role::text = $2
			   AND u.status = 'active'
			 ON CONFLICT DO NOTHING`,
			item.NotificationID,
			recipients.Role,
		)

		err = exec_err
		recipient_count = tag.RowsAffected()

	case "users":
		tag, exec_err := tx.Exec(
			ctx,
			`INSERT INTO notification_recipients (notification_id, user_id)
			 SELECT $1, id
			 FROM users
			 WHERE id = ANY($2::integer[])
			   AND status = 'active'
			 ON CONFLICT DO NOTHING`,
			item.NotificationID,
			recipients.UserIDs,
		)

		err = exec_err
		recipient_count = tag.RowsAffected()

	case "students":
		tag, exec_err := tx.Exec(
			ctx,
			`INSERT INTO notification_recipients (notification_id, user_id)
			 SELECT DISTINCT $1, COALESCE(st.user_id, st.student_id)
			 FROM students st
			 JOIN users u ON u.id = COALESCE(st.user_id, st.student_id)
			 WHERE st.student_id = ANY($2::integer[])
			   AND u.status = 'active'
			 ON CONFLICT DO NOTHING`,
			item.NotificationID,
			recipients.StudentIDs,
		)

		err = exec_err
		recipient_count = tag.RowsAffected()

	case "groups":
		tag, exec_err := tx.Exec(
			ctx,
			`INSERT INTO notification_recipients (notification_id, user_id)
			 SELECT DISTINCT $1, COALESCE(st.user_id, st.student_id)
			 FROM students st
			 JOIN users u ON u.id = COALESCE(st.user_id, st.student_id)
			 WHERE st.group_id = ANY($2::integer[])
			   AND u.status = 'active'
			 ON CONFLICT DO NOTHING`,
			item.NotificationID,
			recipients.GroupIDs,
		)

		err = exec_err
		recipient_count = tag.RowsAffected()

	case "fraud":
		tag, exec_err := tx.Exec(
			ctx,
			`WITH recipient_users AS (
			     SELECT COALESCE(st.user_id, st.student_id) AS user_id
			     FROM students st
			     WHERE st.student_id = ANY($2::integer[])

			     UNION

			     SELECT COALESCE(t.user_id, t.teacher_id) AS user_id
			     FROM teachers t
			     WHERE t.teacher_id = $3

			     UNION

			 SELECT DISTINCT u.id AS user_id
			 FROM users u
			 JOIN user_roles ur ON ur.user_id = u.id AND ur.role = 'admin'
			 WHERE u.status = 'active'
			 )
			 INSERT INTO notification_recipients (notification_id, user_id)
			 SELECT $1, ru.user_id
			 FROM recipient_users ru
			 JOIN users u ON u.id = ru.user_id
			 WHERE u.status = 'active'
			 ON CONFLICT DO NOTHING`,
			item.NotificationID,
			recipients.StudentIDs,
			recipients.TeacherID,
		)

		err = exec_err
		recipient_count = tag.RowsAffected()

	default:
		return Notification{}, fmt.Errorf("invalid notification audience")
	}

	if err != nil {
		return Notification{}, fmt.Errorf("create notification recipients: %w", err)
	}

	if recipient_count == 0 {
		return Notification{}, fmt.Errorf("notification has no recipients")
	}

	item.RecipientCount = int32(recipient_count)

	if err := tx.Commit(ctx); err != nil {
		return Notification{}, fmt.Errorf("commit notification transaction: %w", err)
	}

	return item, nil
}

func (r *NotificationRepository) ListForUser(
	ctx context.Context,
	user_id int32,
	filter NotificationListFilter,
) (NotificationListResult, error) {
	if user_id <= 0 {
		return NotificationListResult{}, fmt.Errorf("user_id is required")
	}

	offset := (filter.Page - 1) * filter.PageSize

	var result NotificationListResult

	err := r.pool.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM notification_recipients nr
		 JOIN notifications n ON n.notification_id = nr.notification_id
		 WHERE nr.user_id = $1
		   AND (n.expires_at IS NULL OR n.expires_at > NOW())
		   AND ($2 = '' OR n.category = $2)
		   AND (NOT $3 OR nr.read_at IS NULL)`,
		user_id,
		filter.Category,
		filter.UnreadOnly,
	).Scan(&result.Total)
	if err != nil {
		return NotificationListResult{}, fmt.Errorf("count user notifications: %w", err)
	}

	err = r.pool.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM notification_recipients nr
		 JOIN notifications n ON n.notification_id = nr.notification_id
		 WHERE nr.user_id = $1
		   AND nr.read_at IS NULL
		   AND (n.expires_at IS NULL OR n.expires_at > NOW())`,
		user_id,
	).Scan(&result.UnreadCount)
	if err != nil {
		return NotificationListResult{}, fmt.Errorf("count unread notifications: %w", err)
	}

	rows, err := r.pool.Query(
		ctx,
		`SELECT n.notification_id,
		        n.category,
		        n.event_type,
		        n.title,
		        n.message,
		        n.created_by_user_id,
		        n.metadata,
		        n.expires_at,
		        n.created_at,
		        nr.read_at,
		        nr.read_at IS NOT NULL
		 FROM notification_recipients nr
		 JOIN notifications n ON n.notification_id = nr.notification_id
		 WHERE nr.user_id = $1
		   AND (n.expires_at IS NULL OR n.expires_at > NOW())
		   AND ($2 = '' OR n.category = $2)
		   AND (NOT $3 OR nr.read_at IS NULL)
		 ORDER BY n.created_at DESC, n.notification_id DESC
		 LIMIT $4 OFFSET $5`,
		user_id,
		filter.Category,
		filter.UnreadOnly,
		filter.PageSize,
		offset,
	)
	if err != nil {
		return NotificationListResult{}, fmt.Errorf("list user notifications: %w", err)
	}

	defer rows.Close()

	result.Items = make([]Notification, 0)

	for rows.Next() {
		var item Notification
		var metadata []byte

		if err := rows.Scan(
			&item.NotificationID,
			&item.Category,
			&item.EventType,
			&item.Title,
			&item.Message,
			&item.CreatedBy,
			&metadata,
			&item.ExpiresAt,
			&item.CreatedAt,
			&item.ReadAt,
			&item.IsRead,
		); err != nil {
			return NotificationListResult{}, fmt.Errorf("scan user notification: %w", err)
		}

		item.Metadata = json.RawMessage(metadata)
		result.Items = append(result.Items, item)
	}

	if err := rows.Err(); err != nil {
		return NotificationListResult{}, fmt.Errorf("iterate user notifications: %w", err)
	}

	return result, nil
}

func (r *NotificationRepository) MarkRead(
	ctx context.Context,
	user_id int32,
	notification_id int64,
) (bool, error) {
	tag, err := r.pool.Exec(
		ctx,
		`UPDATE notification_recipients
		 SET read_at = COALESCE(read_at, NOW())
		 WHERE notification_id = $1
		   AND user_id = $2`,
		notification_id,
		user_id,
	)
	if err != nil {
		return false, fmt.Errorf("mark notification as read: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}

func (r *NotificationRepository) MarkAllRead(
	ctx context.Context,
	user_id int32,
) (int64, error) {
	tag, err := r.pool.Exec(
		ctx,
		`UPDATE notification_recipients
		 SET read_at = NOW()
		 WHERE user_id = $1
		   AND read_at IS NULL`,
		user_id,
	)
	if err != nil {
		return 0, fmt.Errorf("mark all notifications as read: %w", err)
	}

	return tag.RowsAffected(), nil
}

func (r *NotificationRepository) UnreadCount(
	ctx context.Context,
	user_id int32,
) (int32, error) {
	var count int32

	err := r.pool.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM notification_recipients nr
		 JOIN notifications n ON n.notification_id = nr.notification_id
		 WHERE nr.user_id = $1
		   AND nr.read_at IS NULL
		   AND (n.expires_at IS NULL OR n.expires_at > NOW())`,
		user_id,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}

	return count, nil
}

func (r *NotificationRepository) GetSettings(
	ctx context.Context,
	user_id int32,
) (NotificationSettings, error) {
	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO notification_settings (user_id)
		 VALUES ($1)
		 ON CONFLICT (user_id) DO NOTHING`,
		user_id,
	)
	if err != nil {
		return NotificationSettings{}, fmt.Errorf("create default notification settings: %w", err)
	}

	var settings NotificationSettings

	err = r.pool.QueryRow(
		ctx,
		`SELECT user_id,
		        grades,
		        schedule,
		        attendance,
		        system,
		        updated_at
		 FROM notification_settings
		 WHERE user_id = $1`,
		user_id,
	).Scan(
		&settings.UserID,
		&settings.Grades,
		&settings.Schedule,
		&settings.Attendance,
		&settings.System,
		&settings.UpdatedAt,
	)
	if err != nil {
		return NotificationSettings{}, fmt.Errorf("get notification settings: %w", err)
	}

	return settings, nil
}

func (r *NotificationRepository) UpdateSettings(
	ctx context.Context,
	settings NotificationSettings,
) (NotificationSettings, error) {
	var out NotificationSettings

	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO notification_settings (
		     user_id,
		     grades,
		     schedule,
		     attendance,
		     system,
		     updated_at
		 )
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (user_id)
		 DO UPDATE SET
		     grades = EXCLUDED.grades,
		     schedule = EXCLUDED.schedule,
		     attendance = EXCLUDED.attendance,
		     system = EXCLUDED.system,
		     updated_at = NOW()
		 RETURNING user_id,
		           grades,
		           schedule,
		           attendance,
		           system,
		           updated_at`,
		settings.UserID,
		settings.Grades,
		settings.Schedule,
		settings.Attendance,
		settings.System,
	).Scan(
		&out.UserID,
		&out.Grades,
		&out.Schedule,
		&out.Attendance,
		&out.System,
		&out.UpdatedAt,
	)
	if err != nil {
		return NotificationSettings{}, fmt.Errorf("update notification settings: %w", err)
	}

	return out, nil
}

func (r *NotificationRepository) AdminList(
	ctx context.Context,
	filter NotificationListFilter,
) ([]Notification, int32, error) {
	offset := (filter.Page - 1) * filter.PageSize

	var total int32

	err := r.pool.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM notifications
		 WHERE ($1 = '' OR category = $1)`,
		filter.Category,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count admin notifications: %w", err)
	}

	rows, err := r.pool.Query(
		ctx,
		`SELECT n.notification_id,
		        n.category,
		        n.event_type,
		        n.title,
		        n.message,
		        n.created_by_user_id,
		        n.metadata,
		        n.expires_at,
		        n.created_at,
		        COUNT(nr.user_id)
		 FROM notifications n
		 LEFT JOIN notification_recipients nr
		        ON nr.notification_id = n.notification_id
		 WHERE ($1 = '' OR n.category = $1)
		 GROUP BY n.notification_id
		 ORDER BY n.created_at DESC, n.notification_id DESC
		 LIMIT $2 OFFSET $3`,
		filter.Category,
		filter.PageSize,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin notifications: %w", err)
	}

	defer rows.Close()

	items := make([]Notification, 0)

	for rows.Next() {
		var item Notification
		var metadata []byte

		if err := rows.Scan(
			&item.NotificationID,
			&item.Category,
			&item.EventType,
			&item.Title,
			&item.Message,
			&item.CreatedBy,
			&metadata,
			&item.ExpiresAt,
			&item.CreatedAt,
			&item.RecipientCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan admin notification: %w", err)
		}

		item.Metadata = json.RawMessage(metadata)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate admin notifications: %w", err)
	}

	return items, total, nil
}

func (r *NotificationRepository) AdminUpdate(
	ctx context.Context,
	notification_id int64,
	title *string,
	message *string,
	expires_at *time.Time,
) (Notification, bool, error) {
	var item Notification
	var metadata []byte

	err := r.pool.QueryRow(
		ctx,
		`UPDATE notifications
		 SET title = COALESCE($2, title),
		     message = COALESCE($3, message),
		     expires_at = COALESCE($4, expires_at)
		 WHERE notification_id = $1
		   AND created_by_user_id IS NOT NULL
		 RETURNING notification_id,
		           category,
		           event_type,
		           title,
		           message,
		           created_by_user_id,
		           metadata,
		           expires_at,
		           created_at`,
		notification_id,
		title,
		message,
		expires_at,
	).Scan(
		&item.NotificationID,
		&item.Category,
		&item.EventType,
		&item.Title,
		&item.Message,
		&item.CreatedBy,
		&metadata,
		&item.ExpiresAt,
		&item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, false, nil
	}
	if err != nil {
		return Notification{}, false, fmt.Errorf("update admin notification: %w", err)
	}

	item.Metadata = json.RawMessage(metadata)

	return item, true, nil
}

func (r *NotificationRepository) AdminDelete(
	ctx context.Context,
	notification_id int64,
) (bool, error) {
	tag, err := r.pool.Exec(
		ctx,
		`DELETE FROM notifications
		 WHERE notification_id = $1
		   AND created_by_user_id IS NOT NULL`,
		notification_id,
	)
	if err != nil {
		return false, fmt.Errorf("delete admin notification: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}

func (r *NotificationRepository) DeleteForUser(
	ctx context.Context,
	user_id int32,
	notification_id int64,
) (bool, error) {
	tag, err := r.pool.Exec(
		ctx,
		`DELETE FROM notification_recipients
		 WHERE notification_id = $1
		   AND user_id = $2`,
		notification_id,
		user_id,
	)
	if err != nil {
		return false, fmt.Errorf("delete user notification: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}
