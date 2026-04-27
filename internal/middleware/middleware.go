package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"minenepal-backend/internal/stats"
	"time"
)

func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()
		if err != nil {
			return err
		}

		duration := time.Since(start)
		durationMs := float64(duration.Milliseconds())
		stats.RecordResponseTime(durationMs)

		log.Debug().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("duration", duration).
			Msg("request")

		return nil
	}
}

func CORS() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}