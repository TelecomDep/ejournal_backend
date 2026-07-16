package db

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

const queryLatencySampleLimit = 4096

type QueryTimingStats struct {
	Count      uint64  `json:"count"`
	Errors     uint64  `json:"errors"`
	SampleSize int     `json:"sample_size"`
	P50MS      float64 `json:"p50_ms"`
	P95MS      float64 `json:"p95_ms"`
	P99MS      float64 `json:"p99_ms"`
}

type queryTimingMetrics struct {
	mu      sync.Mutex
	count   uint64
	errors  uint64
	samples []int64
	next    int
}

type queryStartedAtKey struct{}

type queryTimingTracer struct {
	metrics *queryTimingMetrics
}

func newQueryTimingMetrics() *queryTimingMetrics {
	return &queryTimingMetrics{samples: make([]int64, 0, queryLatencySampleLimit)}
}

func (t *queryTimingTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryStartedAtKey{}, time.Now())
}

func (t *queryTimingTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	startedAt, ok := ctx.Value(queryStartedAtKey{}).(time.Time)
	if !ok {
		return
	}
	t.metrics.record(time.Since(startedAt), data.Err != nil)
}

func (m *queryTimingMetrics) record(elapsed time.Duration, failed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.count++
	if failed {
		m.errors++
	}
	latency := elapsed.Microseconds()
	if len(m.samples) < queryLatencySampleLimit {
		m.samples = append(m.samples, latency)
		return
	}
	m.samples[m.next] = latency
	m.next = (m.next + 1) % queryLatencySampleLimit
}

func (m *queryTimingMetrics) snapshot() QueryTimingStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	values := append([]int64(nil), m.samples...)
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	percentile := func(p float64) float64 {
		if len(values) == 0 {
			return 0
		}
		index := int(float64(len(values)-1) * p)
		return float64(values[index]) / 1000
	}
	return QueryTimingStats{
		Count:      m.count,
		Errors:     m.errors,
		SampleSize: len(values),
		P50MS:      percentile(0.50),
		P95MS:      percentile(0.95),
		P99MS:      percentile(0.99),
	}
}
