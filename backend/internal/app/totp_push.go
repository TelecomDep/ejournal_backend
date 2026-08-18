package app

import (
	"context"
	"strings"
	"time"
)

type DeviceTokenRegistration struct {
	DeviceToken string `json:"device_token"`
	DeviceName  string `json:"device_name,omitempty"`
	Platform    string `json:"platform,omitempty"`
}

type UserDeviceTokenItem struct {
	ID          int64     `json:"id"`
	UserID      int32     `json:"user_id"`
	DeviceToken string    `json:"device_token"`
	DeviceName  string    `json:"device_name"`
	Platform    string    `json:"platform"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (s *Service) RegisterDeviceToken(token string, data DeviceTokenRegistration) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}

	data.DeviceToken = strings.TrimSpace(data.DeviceToken)
	if data.DeviceToken == "" {
		return Response{OK: false, Error: "device_token is required"}
	}
	if data.Platform == "" {
		data.Platform = "android"
	}
	if data.DeviceName == "" {
		data.DeviceName = "Android Device"
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	_, err = s.store.Pool().Exec(
		ctx,
		`INSERT INTO user_device_tokens (user_id, device_token, device_name, platform, updated_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (user_id, device_token)
		 DO UPDATE SET device_name = EXCLUDED.device_name, platform = EXCLUDED.platform, updated_at = NOW()`,
		user.ID,
		data.DeviceToken,
		data.DeviceName,
		data.Platform,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to register device token"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"status":       "registered",
			"device_token": data.DeviceToken,
			"platform":     data.Platform,
		},
	}
}

func (s *Service) ListDeviceTokens(token string) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	rows, err := s.store.Pool().Query(
		ctx,
		`SELECT id, user_id, device_token, COALESCE(device_name, ''), COALESCE(platform, 'android'), updated_at
		 FROM user_device_tokens
		 WHERE user_id = $1
		 ORDER BY updated_at DESC`,
		user.ID,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to list device tokens"}
	}
	defer rows.Close()

	items := make([]UserDeviceTokenItem, 0)
	for rows.Next() {
		var item UserDeviceTokenItem
		if scanErr := rows.Scan(&item.ID, &item.UserID, &item.DeviceToken, &item.DeviceName, &item.Platform, &item.UpdatedAt); scanErr == nil {
			items = append(items, item)
		}
	}

	return Response{
		OK:     true,
		Result: items,
	}
}

func (s *Service) DeleteDeviceToken(token string, deviceToken string) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	_, err = s.store.Pool().Exec(
		ctx,
		`DELETE FROM user_device_tokens
		 WHERE user_id = $1 AND device_token = $2`,
		user.ID,
		strings.TrimSpace(deviceToken),
	)
	if err != nil {
		return Response{OK: false, Error: "failed to delete device token"}
	}

	return Response{
		OK:     true,
		Result: map[string]any{"status": "deleted"},
	}
}

func (s *Service) DispatchTOTPPushNotification(ctx context.Context, userID int32, totpSecret string, loginIP string) (bool, string) {
	// TOTP secrets must not pass through the notifications table.
	_ = ctx
	_ = userID
	_ = totpSecret
	_ = loginIP
	return false, ""
}
