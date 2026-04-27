package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/rs/zerolog"
	"minenepal-backend/pkg/types"
)

type Client struct {
	Conn      *websocket.Conn
	Subscriptions map[string]bool
	mu         sync.RWMutex
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	serverHub  map[string][]*Client
	mu         sync.RWMutex
	logger     zerolog.Logger
}

func NewHub(logger zerolog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		serverHub:  make(map[string][]*Client),
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Debug().Int("clients", len(h.clients)).Msg("client connected")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Conn.Close()
				
				for server := range client.Subscriptions {
					h.removeSubscription(client, server)
				}
			}
			h.mu.Unlock()
			h.logger.Debug().Int("clients", len(h.clients)).Msg("client disconnected")

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
					h.logger.Err(err).Msg("error writing to client")
					h.mu.RUnlock()
					h.unregister <- client
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Subscribe(client *Client, servers []string) {
	client.mu.Lock()
	for _, server := range servers {
		client.Subscriptions[server] = true
		h.addSubscription(client, server)
	}
	client.mu.Unlock()
}

func (h *Hub) Unsubscribe(client *Client, servers []string) {
	client.mu.Lock()
	for _, server := range servers {
		delete(client.Subscriptions, server)
		h.removeSubscription(client, server)
	}
	client.mu.Unlock()
}

func (h *Hub) addSubscription(client *Client, server string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	clients := h.serverHub[server]
	for _, c := range clients {
		if c == client {
			return
		}
	}
	h.serverHub[server] = append(h.serverHub[server], client)
}

func (h *Hub) removeSubscription(client *Client, server string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	clients := h.serverHub[server]
	newClients := make([]*Client, 0, len(clients))
	for _, c := range clients {
		if c != client {
			newClients = append(newClients, c)
		}
	}
	if len(newClients) == 0 {
		delete(h.serverHub, server)
	} else {
		h.serverHub[server] = newClients
	}
}

func (h *Hub) BroadcastStatusUpdate(server string, data *types.ServerStatus) {
	msg := types.WSMessage{
		Type:   "status_update",
		Server: server,
		Data:   data,
	}
	h.broadcastToServer(server, msg)
}

func (h *Hub) BroadcastVote(server string, username string, success bool) {
	msg := types.WSMessage{
		Type:   "vote",
		Server: server,
		Data: map[string]interface{}{
			"username":  username,
			"success":   success,
			"timestamp": json.Number(""),
		},
	}
	h.broadcastToServer(server, msg)
}

func (h *Hub) broadcastToServer(server string, msg types.WSMessage) {
	h.mu.RLock()
	clients := h.serverHub[server]
	h.mu.RUnlock()

	if len(clients) == 0 {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Err(err).Msg("failed to marshal message")
		return
	}

	h.mu.RLock()
	for _, client := range clients {
		if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("error writing to client: %v", err)
			h.unregister <- client
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) BroadcastToAll(msg types.WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Err(err).Msg("failed to marshal message")
		return
	}
	h.broadcast <- data
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) GetSubscriptions(client *Client) []string {
	client.mu.RLock()
	defer client.mu.RUnlock()
	
	servers := make([]string, 0, len(client.Subscriptions))
	for server := range client.Subscriptions {
		servers = append(servers, server)
	}
	return servers
}