package app

type AdminServiceItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Path        string `json:"path"`
	ExternalURL string `json:"external_url"`
	Icon        string `json:"icon"`
	Status      string `json:"status"` // healthy, warning, unknown
}

func (s *Service) admin_services_list(token string) Response {
	actor, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: "unauthorized"}
	}
	if actor.Role != RoleAdmin && actor.Role != RoleMinister {
		return Response{OK: false, Error: "forbidden: admin role required"}
	}

	baseURL := s.siteBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:9001"
	}

	services := []AdminServiceItem{
		{
			ID:          "grafana",
			Name:        "Grafana Monitoring",
			Category:    "Monitoring & Telemetry",
			Description: "LMS Production Overview dashboard: HTTP RPS, p95/p99 latency, Go memory, and PostgreSQL connections.",
			Path:        "/grafana/",
			ExternalURL: baseURL + "/grafana/",
			Icon:        "dashboard",
			Status:      "healthy",
		},
		{
			ID:          "prometheus",
			Name:        "Prometheus Metrics",
			Category:    "Monitoring & Telemetry",
			Description: "Raw metrics collector, targets status, and active alerting rules.",
			Path:        "/prometheus/",
			ExternalURL: baseURL + "/prometheus/",
			Icon:        "activity",
			Status:      "healthy",
		},
		{
			ID:          "azimutt",
			Name:        "Azimutt ERD Explorer",
			Category:    "Database Architecture",
			Description: "Visual PostgreSQL database schema explorer with enterprise relationship mapping.",
			Path:        "/azimutt/",
			ExternalURL: baseURL + "/azimutt/",
			Icon:        "database",
			Status:      "healthy",
		},
		{
			ID:          "webmail",
			Name:        "Mailserver Webmail",
			Category:    "Communication",
			Description: "Internal SMTP mailserver administration and system email delivery inspection.",
			Path:        "/webmail/",
			ExternalURL: baseURL + "/webmail/",
			Icon:        "mail",
			Status:      "healthy",
		},
	}

	return Response{
		OK:     true,
		Result: services,
	}
}
