package handler

import (
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"

	ws "minenepal-backend/internal/websocket"
	"minenepal-backend/pkg/types"
)

type WebSocketHandler struct {
	hub *ws.Hub
}

func NewWebSocketHandler(hub *ws.Hub) *WebSocketHandler {
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
	client := &ws.Client{
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

func (h *WebSocketHandler) handleMessage(client *ws.Client, msg []byte) {
	var sub types.WSSubscription
	if err := json.Unmarshal(msg, &sub); err != nil {
		h.sendError(client.Conn, "Invalid message format")
		return
	}

	switch sub.Action {
	case "subscribe":
		if len(sub.Servers) > 0 {
			h.hub.Subscribe(client, sub.Servers)
			h.sendAck(client.Conn, "subscribed", sub.Servers)
		} else {
			h.sendError(client.Conn, "No servers specified")
		}

	case "unsubscribe":
		if len(sub.Servers) > 0 {
			h.hub.Unsubscribe(client, sub.Servers)
			h.sendAck(client.Conn, "unsubscribed", sub.Servers)
		} else {
			h.hub.Unsubscribe(client, h.hub.GetSubscriptions(client))
			h.sendAck(client.Conn, "unsubscribed all", nil)
		}

	default:
		h.sendError(client.Conn, "Unknown action: "+sub.Action)
	}
}

func (h *WebSocketHandler) sendError(conn *websocket.Conn, message string) {
	msg := types.WSMessage{
		Type: "error",
		Data: map[string]string{
			"message": message,
		},
	}
	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)
}

func (h *WebSocketHandler) sendAck(conn *websocket.Conn, action string, servers []string) {
	msg := types.WSMessage{
		Type: "ack",
		Data: map[string]interface{}{
			"action":  action,
			"servers": servers,
		},
	}
	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)
}
