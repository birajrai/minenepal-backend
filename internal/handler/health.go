package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
	"minenepal-backend/internal/stats"
	"minenepal-backend/pkg/types"
)

var (
	healthCache    *healthCacheEntry
	healthCacheTTL = 5 * time.Second
	startTime      = time.Now()
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

		avgResponseTime := stats.GetAverageResponseTime()
		responseTime := "0 ms"
		if avgResponseTime > 0 {
			responseTime = fmt.Sprintf("%.2f ms", avgResponseTime)
		}

		response := &types.HealthResponse{
			Status:       "healthy",
			Uptime:       fmt.Sprintf("%.2f seconds", time.Since(startTime).Seconds()),
			ResponseTime: responseTime,
			Requests:     atomic.LoadInt64(&stats.RequestCount),
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
		CPU:     getCPUMetrics(),
		Storage: getStorageMetrics(),
	}
}

func getCPUMetrics() types.CPUMetrics {
	return types.CPUMetrics{
		Usage:       "N/A",
		LoadAverage: []string{"N/A", "N/A", "N/A"},
	}
}

func getStorageMetrics() string {
	cacheDir := "./cache"
	var size int64

	filepath.Walk(cacheDir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return fmt.Sprintf("%.2f MB", float64(size)/1024/1024)
}
