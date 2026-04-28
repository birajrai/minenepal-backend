package handler

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"minenepal-backend/internal/service"
)

type StatusHandler struct {
	serverService *service.ServerService
}

func NewStatusHandler(serverService *service.ServerService) *StatusHandler {
	return &StatusHandler{
		serverService: serverService,
	}
}

func (h *StatusHandler) GetStatus(c *fiber.Ctx) error {
	hostParam := c.Params("ip")
	if hostParam == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "IP address is required",
		})
	}

	ip, port := parseHostPort(hostParam)

	status, err := h.serverService.GetStatus(ip, port)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if !status.Online {
		return c.Status(fiber.StatusNotFound).JSON(status)
	}

	return c.JSON(status)
}

func parseHostPort(s string) (string, int) {
	ip := s
	port := 25565

	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		ip = parts[0]
		if len(parts) > 1 {
			fmt.Sscanf(parts[1], "%d", &port)
		}
	}

	return ip, port
}

func (h *StatusHandler) GetStatusBulk(c *fiber.Ctx) error {
	serversParam := c.Query("servers")
	if serversParam == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No servers provided",
		})
	}

	servers := strings.Split(serversParam, ",")
	cleanedServers := make([]string, 0, len(servers))
	for _, s := range servers {
		s = strings.TrimSpace(s)
		if s != "" {
			cleanedServers = append(cleanedServers, s)
		}
	}

	if len(cleanedServers) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No valid servers provided",
		})
	}

	results := h.serverService.GetStatusBulk(cleanedServers)

	sortedResults := make(map[string]interface{})
	for _, key := range cleanedServers {
		if val, ok := results[key]; ok {
			sortedResults[key] = val
		}
	}

	return c.JSON(sortedResults)
}
