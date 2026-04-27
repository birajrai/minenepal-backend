package handler

import (
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"minenepal-backend/internal/websocket"
	"minenepal-backend/pkg/types"
)

type WebSocketHandler struct {
	hub *websocket.Hub
}

func NewWebSocketHandler(hub *websocket.Hub) *WebSocketHandler {
	return &WebSocketHandler{
		hub: hub,
	}
}

func (h *WebSocketHandler) Upgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

func (h *WebSocketHandler) Handle(c *websocket.Conn) {
	client := &websocket.Client{
		Conn:          c,
		Subscriptions: make(map[string]bool),
	}

	h.hub.Register(client)
	defer h.hub.Unregister(client)

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}

		h.handleMessage(client, msg)
	}
}

func (h *WebSocketHandler) handleMessage(client *websocket.Client, msg []byte) {
	var sub types.WSSubscription
	if err := json.Unmarshal(msg, &sub); err != nil {
		h.sendError(client, "Invalid message format")
		return
	}

	switch sub.Action {
	case "subscribe":
		if len(sub.Servers) > 0 {
			h.hub.Subscribe(client, sub.Servers)
			h.sendAck(client, "subscribed", sub.Servers)
		} else {
			h.sendError(client, "No servers specified")
		}

	case "unsubscribe":
		if len(sub.Servers) > 0 {
			h.hub.Unsubscribe(client, sub.Servers)
			h.sendAck(client, "unsubscribed", sub.Servers)
		} else {
			h.hub.Unsubscribe(client, h.hub.GetSubscriptions(client))
			h.sendAck(client, "unsubscribed all", nil)
		}

	default:
		h.sendError(client, "Unknown action: "+sub.Action)
	}
}

func (h *WebSocketHandler) sendError(client *websocket.Client, message string) {
	msg := types.WSMessage{
		Type: "error",
		Data: map[string]string{
			"message": message,
		},
	}
	data, _ := json.Marshal(msg)
	client.Conn.WriteMessage(websocket.TextMessage, data)
}

func (h *WebSocketHandler) sendAck(client *websocket.Client, action string, servers []string) {
	msg := types.WSMessage{
		Type: "ack",
		Data: map[string]interface{}{
			"action":  action,
			"servers": servers,
		},
	}
	data, _ := json.Marshal(msg)
	client.Conn.WriteMessage(websocket.TextMessage, data)
}