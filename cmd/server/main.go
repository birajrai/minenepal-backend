package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"minenepal-backend/internal/config"
	"minenepal-backend/internal/handler"
	"minenepal-backend/internal/middleware"
	"minenepal-backend/internal/service"
	"minenepal-backend/internal/stats"
	ws "minenepal-backend/internal/websocket"
)

var (
	buildVersion = "dev"
	buildDate    = "unknown"
)

func main() {
	version := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *version {
		fmt.Printf("MineNepal Backend %s (built %s)\n", buildVersion, buildDate)
		return
	}

	cfg := config.Load()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if cfg.LogPretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}
	zerolog.SetGlobalLevel(cfg.LogLevel)

	log.Info().
		Str("port", cfg.Port).
		Str("host", cfg.Host).
		Dur("cache_ttl", cfg.CacheTTL).
		Msg("starting minenepal-backend")

	cache := service.NewCacheService(cfg.CacheDir, cfg.CacheTTL)
	serverService := service.NewServerService(cache, cfg.ServerQueryTimeout)
	votifierService := service.NewVotifierService(cfg.VotifierTimeout)
	bannerService := service.NewBannerService(cache, cfg.BannerCacheTTL)

	hub := ws.NewHub(log.With().Str("component", "websocket").Logger())

	go hub.Run()

	healthHandler := handler.NewHealthHandler()
	statusHandler := handler.NewStatusHandler(serverService)
	voteHandler := handler.NewVoteHandler(votifierService, hub)
	bannerHandler := handler.NewBannerHandler(serverService, bannerService)
	wsHandler := handler.NewWebSocketHandler(hub)
	metricsHandler := handler.NewMetricsHandler()

	app := fiber.New(fiber.Config{
		AppName:               "MineNepal Backend",
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:          120 * time.Second,
		DisableStartupMessage: false,
	})

	app.Use(recover.New())
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(middleware.CORS())

	app.Use(func(c *fiber.Ctx) error {
		stats.IncrementRequestCount()
		return c.Next()
	})

	app.Get("/", healthHandler.Health())
	app.Get("/metrics", metricsHandler.Metrics())

	api := app.Group("/api")

	serverGroup := api.Group("/server")

	serverGroup.Get("/status/:ip", statusHandler.GetStatus)
	serverGroup.Get("/status/:ip/:port", statusHandler.GetStatus)
	serverGroup.Get("/status/bulk", statusHandler.GetStatusBulk)

	serverGroup.Get("/banner/:ip", bannerHandler.GetBanner)
	serverGroup.Get("/banner/:ip/:port", bannerHandler.GetBanner)

	api.Post("/vote", voteHandler.SendVote)

	app.Static("/icons", cache.GetCacheDir()+"/icons")
	app.Static("/banners", bannerService.GetBannerPath())

	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	wsHandlerFunc := func(c *websocket.Conn) {
		wsHandler.Handle(c)
	}
	app.Get("/ws", websocket.New(wsHandlerFunc))

	go func() {
		if err := app.Listen(cfg.Host + ":" + cfg.Port); err != nil {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	log.Info().Str("addr", cfg.Host+":"+cfg.Port).Msg("server started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")

	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("shutdown error")
	}

	log.Info().Msg("server stopped")
}