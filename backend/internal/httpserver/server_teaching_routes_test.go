package httpserver

import (
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestTeachingRoutesKeepLegacyAliases(t *testing.T) {
	server := &Server{}
	app := fiber.New()
	server.registerTeachingRoutes(app.Group("/api/teaching"))
	server.registerTeachingRoutes(app.Group("/api/teacher"))

	routes := make(map[string]struct{})
	for _, route := range app.GetRoutes(true) {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	for route := range routes {
		if !strings.Contains(route, " /api/teaching/") {
			continue
		}
		legacyRoute := strings.Replace(route, " /api/teaching/", " /api/teacher/", 1)
		if _, ok := routes[legacyRoute]; !ok {
			t.Errorf("missing legacy alias for %s", route)
		}
	}
}
