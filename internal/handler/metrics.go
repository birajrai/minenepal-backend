package handler

import (
	"fmt"
	"runtime"

	"github.com/gofiber/fiber/v2"
	"minenepal-backend/internal/stats"
)

type MetricsHandler struct{}

func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

func (h *MetricsHandler) Metrics() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		output := fmt.Sprintf(`# HELP minenepal_requests_total Total number of requests
# TYPE minenepal_requests_total counter
minenepal_requests_total %d

# HELP minenepal_cache_hits_total Cache hit count
# TYPE minenepal_cache_hits_total counter
minenepal_cache_hits_total %d

# HELP minenepal_cache_misses_total Cache miss count
# TYPE minenepal_cache_misses_total counter
minenepal_cache_misses_total %d

# HELP process_memory_bytes Process memory usage
# TYPE process_memory_bytes gauge
process_memory_bytes{type="rss"} %d
process_memory_bytes{type="heap_used"} %d
process_memory_bytes{type="heap_total"} %d

# HELP minenepal_votes_total Total votes processed
# TYPE minenepal_votes_total counter
minenepal_votes_total %d
`,
			stats.GetRequestCount(),
			stats.GetCacheHits(),
			stats.GetCacheMisses(),
			memStats.Sys,
			memStats.HeapInuse,
			memStats.HeapSys,
			stats.GetVotesTotal(),
		)

		return c.SendString(output)
	}
}