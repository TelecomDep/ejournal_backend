package app

import (
	"context"
	"encoding/json"
	"time"
)

type AuditLogItem struct {
	LogID        int64           `json:"log_id"`
	ActorID      int32           `json:"actor_id"`
	ActorName    string          `json:"actor_name"`
	ActorRole    string          `json:"actor_role"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	Details      json.RawMessage `json:"details"`
	IPAddress    string          `json:"ip_address,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type AuditLogsQuery struct {
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
	Action   string `json:"action,omitempty"`
}

func (s *Service) RecordAuditLog(
	ctx context.Context,
	actorID int32,
	actorName string,
	actorRole string,
	action string,
	resourceType string,
	resourceID string,
	details map[string]any,
	ipAddress string,
) error {
	if details == nil {
		details = make(map[string]any)
	}
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	_, err = s.store.Pool().Exec(
		ctx,
		`INSERT INTO audit_logs (
		     actor_id,
		     actor_name,
		     actor_role,
		     action,
		     resource_type,
		     resource_id,
		     details,
		     ip_address
		 ) VALUES (
		     NULLIF($1, 0),
		     $2,
		     $3,
		     $4,
		     $5,
		     $6,
		     $7,
		     $8
		 )`,
		actorID,
		actorName,
		actorRole,
		action,
		resourceType,
		resourceID,
		detailsJSON,
		ipAddress,
	)
	return err
}

func (s *Service) admin_audit_logs(token string, query AuditLogsQuery) Response {
	actor, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if actor.Role != RoleAdmin && actor.Role != RoleMinister {
		return Response{OK: false, Error: "forbidden: admin role required"}
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
		 FROM audit_logs
		 WHERE ($1 = '' OR action = $1)`,
		query.Action,
	).Scan(&total)
	if err != nil {
		return Response{OK: false, Error: "failed to count audit logs"}
	}

	rows, err := s.store.Pool().Query(
		ctx,
		`SELECT log_id,
		        COALESCE(actor_id, 0),
		        COALESCE(actor_name, ''),
		        COALESCE(actor_role, ''),
		        action,
		        resource_type,
		        COALESCE(resource_id, ''),
		        details,
		        COALESCE(ip_address, ''),
		        created_at
		 FROM audit_logs
		 WHERE ($1 = '' OR action = $1)
		 ORDER BY created_at DESC, log_id DESC
		 LIMIT $2 OFFSET $3`,
		query.Action,
		query.PageSize,
		offset,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to fetch audit logs"}
	}
	defer rows.Close()

	items := make([]AuditLogItem, 0)
	for rows.Next() {
		var item AuditLogItem
		var detailsRaw []byte
		if scanErr := rows.Scan(
			&item.LogID,
			&item.ActorID,
			&item.ActorName,
			&item.ActorRole,
			&item.Action,
			&item.ResourceType,
			&item.ResourceID,
			&detailsRaw,
			&item.IPAddress,
			&item.CreatedAt,
		); scanErr == nil {
			item.Details = json.RawMessage(detailsRaw)
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
