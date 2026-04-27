package handler

import (
	"fmt"
	"runtime"
	"sync/atomic"

	"github.com/gofiber/fiber/v2"
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

		var memStats2 runtime.MemStats
		runtime.ReadMemStats(&memStats2)

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
			atomic.LoadInt64(&requestCount),
			atomic.LoadInt64(&cacheHits),
			atomic.LoadInt64(&cacheMisses),
			memStats.Sys,
			memStats.HeapInuse,
			memStats.HeapSys,
			atomic.LoadInt64(&votesTotal),
		)

		return c.SendString(output)
	}
}

var (
	requestCount  int64 = 0
	cacheHits     int64 = 0
	cacheMisses   int64 = 0
	votesTotal    int64 = 0
)

func IncrementRequestCount() {
	atomic.AddInt64(&requestCount, 1)
}

func IncrementCacheHit() {
	atomic.AddInt64(&cacheHits, 1)
}

func IncrementCacheMiss() {
	atomic.AddInt64(&cacheMisses, 1)
}

func IncrementVotes() {
	atomic.AddInt64(&votesTotal, 1)
}

func IncrementVotesByMethod(method string) {
	atomic.AddInt64(&votesTotal, 1)
}

func GetRequestCount() int64 {
	return atomic.LoadInt64(&requestCount)
}

func GetCacheHits() int64 {
	return atomic.LoadInt64(&cacheHits)
}

func GetCacheMisses() int64 {
	return atomic.LoadInt64(&cacheMisses)
}

func GetVotesTotal() int64 {
	return atomic.LoadInt64(&votesTotal)
}