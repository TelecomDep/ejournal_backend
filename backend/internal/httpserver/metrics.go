package httpserver

import (
	"crypto/subtle"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

const latencySampleLimit = 4096

type httpMetrics struct {
	mu       sync.Mutex
	total    uint64
	byStatus map[int]uint64
	samples  []int64
	next     int
}

func newHTTPMetrics() *httpMetrics {
	return &httpMetrics{byStatus: make(map[int]uint64), samples: make([]int64, 0, latencySampleLimit)}
}

func (m *httpMetrics) record(status int, elapsed time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.total++
	m.byStatus[status]++
	latency := elapsed.Microseconds()
	if len(m.samples) < latencySampleLimit {
		m.samples = append(m.samples, latency)
		return
	}
	m.samples[m.next] = latency
	m.next = (m.next + 1) % latencySampleLimit
}

func (m *httpMetrics) snapshot() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	statuses := make(map[string]uint64, len(m.byStatus))
	for status, count := range m.byStatus {
		statuses[strconv.Itoa(status)] = count
	}
	latencies := append([]int64(nil), m.samples...)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	percentile := func(p float64) float64 {
		if len(latencies) == 0 {
			return 0
		}
		index := int(float64(len(latencies)-1) * p)
		return float64(latencies[index]) / 1000
	}
	return map[string]any{
		"requests_total":      m.total,
		"responses_by_status": statuses,
		"latency_sample_size": len(latencies),
		"latency_ms":          fiber.Map{"p50": percentile(0.50), "p95": percentile(0.95), "p99": percentile(0.99)},
	}
}

func (s *Server) metricsMiddleware(c *fiber.Ctx) error {
	startedAt := time.Now()
	err := c.Next()
	s.metrics.record(c.Response().StatusCode(), time.Since(startedAt))
	return err
}

func (s *Server) healthHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) internalMetricsHandler(c *fiber.Ctx) error {
	expectedToken := s.cfg.MetricsToken
	providedToken := c.Get("X-Metrics-Token")
	if expectedToken == "" || subtle.ConstantTimeCompare([]byte(expectedToken), []byte(providedToken)) != 1 {
		return c.SendStatus(fiber.StatusNotFound)
	}
	return c.JSON(fiber.Map{
		"http":         s.metrics.snapshot(),
		"runtime":      s.svc.RuntimeStats(),
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}
