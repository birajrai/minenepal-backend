package handler

import (
	"github.com/gofiber/fiber/v2"
	"minenepal-backend/internal/service"
	"minenepal-backend/internal/websocket"
	"minenepal-backend/pkg/types"
)

type VoteHandler struct {
	votifierService *service.VotifierService
	hub             *websocket.Hub
}

func NewVoteHandler(votifierService *service.VotifierService, hub *websocket.Hub) *VoteHandler {
	return &VoteHandler{
		votifierService: votifierService,
		hub:             hub,
	}
}

func (h *VoteHandler) SendVote(c *fiber.Ctx) error {
	var req types.VoteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.VoteResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if req.Username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.VoteResponse{
			Success: false,
			Error:   "Username is required",
		})
	}

	if req.ServerIP == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.VoteResponse{
			Success: false,
			Error:   "Server IP is required",
		})
	}

	if req.VotifierMethod == "" {
		req.VotifierMethod = "v2"
	}

	if req.VotifierToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.VoteResponse{
			Success: false,
			Error:   "Votifier token is required",
		})
	}

	if req.VotifierPort == 0 {
		req.VotifierPort = 8192
	}

	if req.Service == "" {
		req.Service = "MineNepal"
	}

	if err := h.votifierService.SendVote(&req); err != nil {
		serverKey := req.ServerIP
		if req.ServerPort != 0 {
			serverKey = req.ServerIP
		}
		h.hub.BroadcastVote(serverKey, req.Username, false)

		return c.Status(fiber.StatusInternalServerError).JSON(types.VoteResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	serverKey := req.ServerIP
	h.hub.BroadcastVote(serverKey, req.Username, true)

	return c.JSON(types.VoteResponse{
		Success: true,
	})
}