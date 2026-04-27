package handler

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"minenepal-backend/pkg/types"
)

var (
	requestCount   int64
	startTime      = time.Now()
	healthCache    *healthCacheEntry
	healthCacheTTL = 5 * time.Second
)

type healthCacheEntry struct {
	timestamp time.Time
	data      *types.HealthResponse
}

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health() fiber.Handler {
	return func(c *fiber.Ctx) error {
		now := time.Now()

		if healthCache != nil && now.Sub(healthCache.timestamp) < healthCacheTTL {
			return c.JSON(healthCache.data)
		}

		metrics := getSystemMetrics()
		avgResponseTime := "0 ms"

		response := &types.HealthResponse{
			Status:       "healthy",
			Uptime:       fmt.Sprintf("%.2f seconds", time.Since(startTime).Seconds()),
			ResponseTime: avgResponseTime,
			Requests:     atomic.LoadInt64(&requestCount),
			Memory:       metrics.Memory,
			CPU:          metrics.CPU,
			Storage:      metrics.Storage,
		}

		healthCache = &healthCacheEntry{
			timestamp: now,
			data:      response,
		}

		return c.JSON(response)
	}
}

type metricsData struct {
	Memory  types.MemoryMetrics
	CPU     types.CPUMetrics
	Storage string
}

func getSystemMetrics() metricsData {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return metricsData{
		Memory: types.MemoryMetrics{
			RSS:       fmt.Sprintf("%.2f MB", float64(memStats.Sys)/1024/1024),
			HeapUsed:  fmt.Sprintf("%.2f MB", float64(memStats.HeapInuse)/1024/1024),
			HeapTotal: fmt.Sprintf("%.2f MB", float64(memStats.HeapSys)/1024/1024),
			External:  fmt.Sprintf("%.2f MB", float64(memStats.Lookups)/1024/1024),
		},
		CPU: types.CPUMetrics{
			Usage:       "0%",
			LoadAverage: []string{"0", "0", "0"},
		},
		Storage: getCacheDirSize(),
	}
}

func getCacheDirSize() string {
	return "0 MB"
}

func IncrementRequestCount() {
	atomic.AddInt64(&requestCount, 1)
}

func GetRequestCount() int64 {
	return atomic.LoadInt64(&requestCount)
}