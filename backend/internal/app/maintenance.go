package app

import (
	"context"
	"encoding/json"
	"time"
)

type MaintenanceStatus struct {
	Enabled   bool      `json:"enabled"`
	Message   string    `json:"message"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

func (s *Service) GetMaintenanceStatus(ctx context.Context) (MaintenanceStatus, error) {
	var valJSON []byte
	var updatedAt time.Time
	err := s.store.Pool().QueryRow(
		ctx,
		`SELECT setting_value, updated_at
		 FROM system_settings
		 WHERE setting_key = 'maintenance_mode'`,
	).Scan(&valJSON, &updatedAt)
	if err != nil {
		return MaintenanceStatus{Enabled: false, Message: "System operating normally"}, nil
	}

	var status MaintenanceStatus
	if err := json.Unmarshal(valJSON, &status); err != nil {
		return MaintenanceStatus{Enabled: false, Message: "System operating normally"}, nil
	}
	status.UpdatedAt = updatedAt
	return status, nil
}

func (s *Service) SetMaintenanceStatus(ctx context.Context, enabled bool, message string) error {
	status := MaintenanceStatus{
		Enabled: enabled,
		Message: message,
	}
	valJSON, err := json.Marshal(status)
	if err != nil {
		return err
	}

	_, err = s.store.Pool().Exec(
		ctx,
		`INSERT INTO system_settings (setting_key, setting_value, updated_at)
		 VALUES ('maintenance_mode', $1, NOW())
		 ON CONFLICT (setting_key)
		 DO UPDATE SET setting_value = EXCLUDED.setting_value, updated_at = NOW()`,
		valJSON,
	)
	return err
}

func (s *Service) admin_system_maintenance_get(token string) Response {
	ctx, cancel := s.dbContext()
	defer cancel()

	status, err := s.GetMaintenanceStatus(ctx)
	if err != nil {
		return Response{OK: false, Error: "failed to get maintenance status"}
	}

	return Response{
		OK:     true,
		Result: status,
	}
}

func (s *Service) admin_system_maintenance_set(token string, req MaintenanceStatus) Response {
	actor, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if actor.Role != RoleAdmin {
		return Response{OK: false, Error: "forbidden: admin role required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	if err := s.SetMaintenanceStatus(ctx, req.Enabled, req.Message); err != nil {
		return Response{OK: false, Error: "failed to update maintenance status"}
	}

	s.RecordAuditLog(
		ctx,
		actor.ID,
		actor.Login,
		actor.Role,
		"system_maintenance_toggle",
		"system",
		"maintenance_mode",
		map[string]any{
			"enabled": req.Enabled,
			"message": req.Message,
		},
		"",
	)

	return Response{
		OK: true,
		Result: map[string]any{
			"enabled": req.Enabled,
			"message": req.Message,
		},
	}
}
