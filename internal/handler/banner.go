package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"minenepal-backend/internal/service"
)

type BannerHandler struct {
	serverService *service.ServerService
	bannerService *service.BannerService
}

func NewBannerHandler(serverService *service.ServerService, bannerService *service.BannerService) *BannerHandler {
	return &BannerHandler{
		serverService: serverService,
		bannerService: bannerService,
	}
}

func (h *BannerHandler) GetBanner(c *fiber.Ctx) error {
	ip := c.Params("ip")
	if ip == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "IP address is required",
		})
	}

	port := 25565
	if c.Params("port") != "" {
		if p, err := strconv.Atoi(c.Params("port")); err == nil {
			port = p
		}
	}

	status, err := h.serverService.GetStatus(ip, port)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	bannerData, err := h.bannerService.GenerateBanner(status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.Set("Content-Type", "image/svg+xml")
	c.Set("Cache-Control", "public, max-age=86400")

	return c.Send(bannerData)
}

func (h *BannerHandler) GetBannerSVG(c *fiber.Ctx) error {
	ip := c.Params("ip")
	if ip == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "IP address is required",
		})
	}

	port := 25565
	if c.Params("port") != "" {
		if p, err := strconv.Atoi(c.Params("port")); err == nil {
			port = p
		}
	}

	status, err := h.serverService.GetStatus(ip, port)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	bannerData, err := h.bannerService.GenerateBanner(status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.Set("Content-Type", "image/svg+xml")
	c.Set("Cache-Control", "public, max-age=86400")

	return c.Send(bannerData)
}

func (h *BannerHandler) GetBannerByQuery(c *fiber.Ctx) error {
	ip := c.Query("ip")
	if ip == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "IP address is required",
		})
	}

	port := 25565
	if portStr := c.Query("port"); portStr != "" {
		if strings.Contains(portStr, ":") {
			parts := strings.Split(portStr, ":")
			ip = parts[0]
		}
	}

	status, err := h.serverService.GetStatus(ip, port)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	bannerData, err := h.bannerService.GenerateBanner(status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.Set("Content-Type", "image/svg+xml")
	c.Set("Cache-Control", "public, max-age=86400")

	return c.Send(bannerData)
}
