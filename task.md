# MineNepal Backend - Complete Specification

## Overview

MineNepal Backend is a high-performance Go service that provides:
- Minecraft server status querying
- Dynamic banner generation
- Votifier vote forwarding (v1 & v2 protocols)
- Real-time WebSocket updates

**Tech Stack:** Go 1.21+, Fiber v2, Zerolog

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLIENTS                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐             │
│   │ Browser  │    │  Discord │    │   Bot    │    │ Laravel  │             │
│   │  (Web)   │    │   Bot    │    │          │    │   App    │             │
│   └────┬─────┘    └────┬─────┘    └────┬─────┘    └────┬─────┘             │
│        │                │                │                │                   │
│        │ HTTP/WebSocket │                │   HTTP POST   │   HTTP Request   │
│        │                │                │                │                   │
└────────┼────────────────┼────────────────┼────────────────┼───────────────────┘
         │                │                │                │
         ▼                ▼                ▼                ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         MINENEPAL BACKEND (Go)                                │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                         API LAYER (Fiber)                             │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │   │
│  │  │ Health  │ │ Status  │ │ Banner  │ │  Vote   │ │   WS    │       │   │
│  │  │ GET /   │ │ GET /s  │ │ GET /b  │ │POST /v  │ │ GET /ws │       │   │
│  │  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘       │   │
│  └───────┼───────────┼───────────┼───────────┼───────────┼───────────────┘   │
│          │           │           │           │           │                   │
│  ┌───────▼───────────▼───────────▼───────────▼───────────▼───────────────┐ │
│  │                         SERVICE LAYER                                   │ │
│  │                                                                          │ │
│  │  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐     │ │
│  │  │   CacheService  │    ���   ServerService │    │  VotifierService│     │ │
│  │  │  (In-Mem + Disk) │    │  (Query + Parse) │    │   (v1 + v2)     │     │ │
│  │  └────────┬────────┘    └────────┬────────┘    └────────┬────────┘     │ │
│  └───────────┼───────────────────────┼──────────────────────┼──────────────┘ │
│              │                       │                      │                  │
│  ┌───────────▼───────────────────────▼──────────────────────▼──────────────┐ │
│  │                         CACHE LAYER                                       │ │
│  │  ┌──────────────────────────────────────────────────────────────────────┐  │ │
│  │  │  In-Memory Cache (LRU)          │  Disk Cache (TTL-based)           │  │ │
│  │  │  • Server Status (15s TTL)       │  • icons/*.png                    │  │ │
│  │  │  • Banner Cache (24h TTL)        │  • banners/*.webp                 │  │ │
│  │  │  • DNS Lookup                    │  • status JSON                    │  │ │
│  │  └──────────────────────────────────────────────────────────────────────┘  │ │
│  └───────────────────────────────────────────────────────────────────────────┘ │
│                                                                              │
└───────────────────────────────────────────────────────────────────���──────────┘
         │                       │                      │
         ▼                       ▼                      ▼
┌─────────────────┐   ┌─────────────────┐      ┌─────────────────┐
│  DNS Server     │   │ Minecraft       │      │ Votifier Port   │
│  (SRV Lookup)   │   │ Server (25565)  │      │ (8192 default)  │
└─────────────────┘   └─────────────────┘      └─────────────────┘
```

---

## System Flow Diagrams

### 1. Server Status Query Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        SERVER STATUS QUERY FLOW                               │
└─────────────────────────────────────────────────────────────────────────────┘

  ┌─────────┐
  │  Client  │
  └────┬────┘
       │ GET /api/server/status/192.168.1.1:25565
       ▼
  ┌─────────────┐
  │  Validate   │ ─── Invalid IP/Port ───▶ Return 400 Error
  │   Input     │
  └──────┬──────┘
         │ Valid
         ▼
  ┌─────────────┐     ┌─────────────┐
  │   Check     │ ──▶ │   Memory    │ ─── Cache Hit ──▶ Return Cached Data
  │    Cache    │     │    Cache    │
  └──────┬──────┘     └─────────────┘
         │ Cache Miss
         ▼
  ┌─────────────┐     ┌─────────────┐
  │   Check     │ ──▶ │ File Cache  │ ─── Cache Hit ──▶ Return Cached Data
  │    Cache    │     │   (.json)   │
  └──────┬──────┘     └─────────────┘
         │ Cache Miss
         ▼
  ┌─────────────┐
  │ SRV Record  │ ─── _minecraft._tcp.host.com ──▶ Resolve
  │   Lookup     │
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │    DNS      │ ─── Resolve hostname to IP
  │   Lookup    │
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │   TCP       │
  │   Connect   │ ◀── Timeout (5s) ──▶ Return Offline
  └──────┬──────┘
         │ Connected
         ▼
  ┌─────────────┐
  │  Send Ping  │ ─── Minecraft Ping Packet
  │   Packet    │
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │  Read       │
  │   Response  │
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │   Parse     │ ─── Extract: version, players, MOTD, icon
  │   Response  │
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │   Save      │ ─── Save icon to ./cache/icons/
  │    Icon     │ ─── Save status to memory + file cache
  └──────┬──────┘
         │
         ▼
  ┌─────────────┐
  │   Return    │ ─── { online: true, version: "...", players: {...} }
  │   Response  │
  └─────────────┘
```

### 2. Vote Processing Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          VOTE PROCESSING FLOW                                │
└─────────────────────────────────────────────────────────────────────────���───┘

  ┌─────────────┐
  │  MineNepal  │     ┌─────────────┐
  │   (Laravel) │────▶│   POST      │
  └─────────────┘     │ /api/vote   │
                      └──────┬──────┘
                             │
                             ▼
                      ┌─────────────┐
                      │   Validate  │ ─── Invalid ───▶ Return 400
                      │   Request   │
                      └──────┬──────┘
                             │ Valid
                             ▼
                      ┌─────────────┐
                      │ Determine  │ ─── votifier_host set? ───▶ Use votifier_host
                      │   Endpoint  │     votifier_host empty? ───▶ Use server_ip
                      └──────┬──────┘
                             │
                             ▼
                      ┌─────────────┐
                      │   Detect    │
                      │   Protocol  │
                      └──────┬──────┘
                             │
              ┌──────────────┴──────────────┐
              │ v2                          │ v1
              ▼                              ▼
      ┌───────────────┐              ┌───────────────┐
      │    Votifier   │              │    Votifier   │
      │      v2       │              │      v1       │
      └───────┬───────┘              └───────┬───────┘
              │                               │
              ▼                               ▼
      ┌───────────────┐              ┌───────────────┐
      │     TCP       │              │     RSA       │
      │   Connect     │              │   Encrypt     │
      └───────┬───────┘              │   Payload     │
              │                               │
              ▼                               ▼
      ┌───────────────┐              ┌───────────────┐
      │ Read Header   │              │     TCP       │
      │ "VOTIFIER 2"  │              │   Connect     │
      └───────┬───────┘              └───────┬───────┘
              │                               │
              ▼                               ▼
      ┌───────────────┐              ┌───────────────┐
      │ Extract        │              │   Write       │
      │  Challenge     │              │   Encrypted   │
      └───────┬───────┘              │   Packet      │
              │                               │
              ▼                               │
      ┌───────────────┐                       │
      │   Build        │                       │
      │   Payload      │                       │
      │ + HMAC Sign     │                       │
      └───────┬───────┘                       │
              │                                │
              ▼                                │
      ┌───────────────┐                        │
      │   Write       │                        │
      │  JSON Packet  │                        │
      └───────┬───────┘                        │
              │                                │
              ▼                                │
      ┌───────────────┐                        │
      │   Read        │                        │
      │   Response    │                        │
      └───────┬───────┘                        │
              │                                │
              │                                │
              ▼                                ▼
      ┌───────────────���───────────────────────┐
      │            CHECK RESULT                │
      └──────────────────┬────────────────────┘
                         │
          ┌──────────────┴──────────────┐
          │ Success                    │ Failure
          ▼                             ▼
  ┌───────────────┐             ┌───────────────┐
  │ Return 200    │             │ Return 500    │
  │ { success }  │             │ { error }     │
  └───────────────┘             └───────────────┘
          │
          ▼
  ┌───────────────┐
  │   Broadcast   │ ─── WebSocket ──▶ Subscribed Clients
  │   to WS Hub   │     { type: "vote", server: "ip:port", ... }
  └───────────────┘
```

### 3. WebSocket Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       WEBSOCKET FLOW                                         │
└─────────────────────────────────────────────────────────────────────────────┘

  ┌─────────┐         ┌─────────┐         ┌─────────┐         ┌─────────┐
  │ Client  │         │ Client  │         │ Client  │         │ Client  │
  │   A    │         │   B    │         │   C    │         │   D    │
  └────┬────┘         └────┬────┘         └────┬────┘         └────┬────┘
       │                    │                    │                    │
       │ WS Connect         │                    │                    │
       │───────────────────▶│                    │                    │
       │                    │ WS Connect         │                    │
       │                    │────────────────────▶│                    │
       │                    │                    │ WS Connect         │
       │                    │                    │────────────────────▶│
       │                    │                    │                    │
       ▼                    ▼                    ▼                    ▼
  ┌─────────────────────────────────────────────────────────────────────────┐
  │                         WEBSOCKET HUB                                    │
  │                                                                         │
  │  ┌─────────────────────────────────────────────────────────────────┐   │
  │  │                     Client Registry                              │   │
  │  │  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐      │   │
  │  │  │ Client A  │  │ Client B  │  │ Client C  │  │ Client D  │      │   │
  │  │  │ Subs:     │  │ Subs:     │  │ Subs:     │  │ Subs:     │      │   │
  │  │  │ "ip1:port"│  │ "ip1:port"│  │ "ip2:port"│  │ "ip3:port"│      │   │
  │  │  │ "ip2:port"│  │           │  │           │  │ "ip4:port"│      │   │
  │  │  └───────────┘  └───────────┘  └───────────┘  └───────────┘      │   │
  │  └─────────────────────────────────────────────────────────────────┘   │
  │                                                                         │
  └─────────────────────────────────────────────────────────────────────────┘
       │                    │                    │                    │
       │ {"action":         │                    │                    │
       │  "subscribe",     │                    │                    │
       │  "servers":       │                    │                    │
       │  ["ip1:port",     │                    │                    │
       │   "ip2:port"]}    │                    │                    │
       │◀─────────────────▶│                    │                    │
       │                    │                    │                    │

       │                    │                    │                    │
       ▼                    ▼                    ▼                    ▼
  ┌─────────────────────────────────────────────────────────────────────────┐
  │                    SERVER STATUS UPDATE EVENT                             │
  │                                                                         │
  │  When server status changes or is refreshed:                             │
  │                                                                         │
  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                   │
  │  │ Server     │───▶│   Lookup    │───▶│  Broadcast  │                   │
  │  │ "ip1:port" │    │ Subscribers │    │ to Clients  │                   │
  │  │ changed    │    │ (A, B)      │    │ (A, B)      │                   │
  │  └─────────────┘    └─────────────┘    └─────────────┘                   │
  │                                                                         │
  │  Client A ◀── { type: "status_update", server: "ip1:port", data: {...} }
  │  Client B ◀── { type: "status_update", server: "ip1:port", data: {...} }
  │                                                                         │
  └─────────────────────────────────────────────────────────────────────────┘

       │                    │                    │                    │
       ▼                    ▼                    ▼                    ▼
  ┌─────────────────────────────────────────────────────────────────────────┐
  │                    VOTE NOTIFICATION EVENT                               │
  │                                                                         │
  │  When vote is processed:                                                 │
  │                                                                         │
  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                   │
  │  │   Vote      │───▶│   Lookup    │───▶│  Broadcast  │                   │
  │  │  Success    │    │ Subscribers │    │ to Clients  │                   │
  │  └─────────────┘    └─────────────┘    └─────────────┘                   │
  │                                                                         │
  │  All clients subscribed to "server_ip:port" receive:                    │
  │  ◀── { type: "vote", server: "ip:port", data: { username, success } }    │
  │                                                                         │
  └─────────────────────────────────────────────────────────────────────────┘
```

---

## API Endpoints Specification

### 1. Health Check

```
GET /
```

**Response (200 OK):**
```json
{
  "status": "healthy",
  "uptime": "12345.67 seconds",
  "responseTime": "1.23 ms",
  "requests": 12345,
  "memory": {
    "rss": "45.67 MB",
    "heapUsed": "12.34 MB",
    "heapTotal": "25.67 MB",
    "external": "3.45 MB"
  },
  "cpu": {
    "usage": "12.34%",
    "loadAverage": ["1.23", "0.45", "0.12"]
  },
  "storage": "45.67 MB"
}
```

---

### 2. Server Status (Single)

```
GET /api/server/status/:ip
GET /api/server/status/:ip/:port
```

**Parameters:**
- `ip` (required): Server IP or hostname
- `port` (optional): Server port (default: 25565)

**Response (200 OK - Online):**
```json
{
  "online": true,
  "host": "mc.example.com",
  "ip": "192.168.1.1",
  "port": 25565,
  "raw_ip": "192.168.1.1",
  "ping": 45,
  "version": "Paper 1.21.3",
  "players": {
    "online": 45,
    "max": 100
  },
  "motd": {
    "clean": "A Minecraft Server",
    "raw": "§aA Minecraft Server",
    "html": "A Minecraft Server"
  },
  "icon": "/icons/192.168.1.1_25565.png",
  "cached_at": 1234567890
}
```

**Response (404 - Offline):**
```json
{
  "online": false,
  "host": "192.168.1.1",
  "ip": "192.168.1.1",
  "port": 25565,
  "raw_ip": null,
  "ping": null,
  "error": "Server offline or unreachable",
  "icon": null
}
```

---

### 3. Server Status (Bulk)

```
GET /api/server/status/bulk?servers=ip:port,ip2:port2,ip3
```

**Parameters:**
- `servers` (required): Comma-separated list of servers (ip:port or ip)

**Response (200 OK):**
```json
{
  "192.168.1.1:25565": { ... server data ... },
  "192.168.1.2:25565": { ... server data ... },
  "192.168.1.3:25565": { ... server data ... }
}
```

**Notes:**
- Results are sorted by key
- Each server returns online/offline status

---

### 4. Server Banner

```
GET /api/server/banner/:ip
GET /api/server/banner/:ip/:port
```

**Parameters:**
- `ip` (required): Server IP or hostname
- `port` (optional): Server port (default: 25565)

**Response:**
- Content-Type: `image/svg+xml`
- Cached for 24 hours

**Banner Design:**
```
┌─────────────────────────────────────────────────────────────────┐
│ [Icon]                                                          │
│  64x64      Minecraft Server                          45ms       │
│                 ████████████                                  │
│                 ██A Minecraft Server████                       │
│                 ████████████████████████                       │
│                                                                 │
│                      45/100 players                              │
│                   Paper 1.21.3                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

### 5. Vote (NEW)

```
POST /api/vote
Content-Type: application/json
```

**Request Body:**
```json
{
  "server_ip": "192.168.1.1",
  "server_port": 25565,
  "votifier_host": "optional-override.com",
  "votifier_port": 8192,
  "votifier_method": "v2",
  "votifier_token": "your-token-or-public-key",
  "username": "PlayerName",
  "ip_address": "client.ip.address",
  "service": "MineNepal"
}
```

**Field Specifications:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `server_ip` | string | Yes | Minecraft server IP |
| `server_port` | int | No | Server port (default: 25565) |
| `votifier_host` | string | No | Override Votifier host (default: server_ip) |
| `votifier_port` | int | No | Votifier port (default: 8192) |
| `votifier_method` | string | Yes | `v1` or `v2` |
| `votifier_token` | string | Yes | Token (v2) or Public Key (v1) |
| `username` | string | Yes | Minecraft username |
| `ip_address` | string | No | Voter IP address |
| `service` | string | No | Service name (default: "MineNepal") |

**Response (200 OK):**
```json
{
  "success": true
}
```

**Response (500 Error):**
```json
{
  "success": false,
  "error": "Server refused the vote packet"
}
```

---

### 6. WebSocket

```
GET /ws
Upgrade: websocket
```

**Connection Protocol:**

Client sends subscription message:
```json
{
  "action": "subscribe",
  "servers": ["192.168.1.1:25565", "mc.example.com:25565"]
}
```

**Server Messages:**

Status Update:
```json
{
  "type": "status_update",
  "server": "192.168.1.1:25565",
  "data": {
    "online": true,
    "players": { "online": 45, "max": 100 },
    "ping": 45
  }
}
```

Vote Notification:
```json
{
  "type": "vote",
  "server": "192.168.1.1:25565",
  "data": {
    "username": "PlayerName",
    "success": true,
    "timestamp": 1234567890
  }
}
```

Error:
```json
{
  "type": "error",
  "message": "Invalid subscription format"
}
```

---

### 7. Metrics

```
GET /metrics
```

**Response (Prometheus format):**
```
# HELP minenepal_requests_total Total number of requests
# TYPE minenepal_requests_total counter
minenepal_requests_total 12345

# HELP minenepal_response_time_seconds Response time in seconds
# TYPE minenepal_response_time_seconds histogram
minenepal_response_time_seconds_bucket{le="0.005"} 100
minenepal_response_time_seconds_bucket{le="0.01"} 500
minenepal_response_time_seconds_bucket{le="0.025"} 1000
minenepal_response_time_seconds_bucket{le="0.05"} 1500
minenepal_response_time_seconds_bucket{le="+Inf"} 2000

# HELP minenepal_cache_hits_total Cache hit count
# TYPE minenepal_cache_hits_total counter
minenepal_cache_hits_total 10000

# HELP minenepal_cache_misses_total Cache miss count
# TYPE minenepal_cache_misses_total counter
minenepal_cache_misses_total 500

# HELP process_memory_bytes Process memory usage
# TYPE process_memory_bytes gauge
process_memory_bytes{type="rss"} 45678912
process_memory_bytes{type="heap_used"} 12345678

# HELP minenepal_votes_total Total votes processed
# TYPE minenepal_votes_total counter
minenepal_votes_total 1234
minenepal_votes_total{method="v1"} 100
minenepal_votes_total{method="v2"} 1134
```

---

## Votifier Protocol Specifications

### Votifier v1 (RSA)

**Protocol:**
1. TCP connection to votifier port
2. Read one line (server banner, discard)
3. Send 256-byte RSA-encrypted payload
4. Payload format:
   ```
   VOTE
   {service_name}
   {username}
   {ip_address}
   {timestamp}
   ```

**Encryption:**
- Algorithm: RSA with PKCS1 padding
- Key: Base64-encoded public key (with headers stripped)

---

### Votifier v2 (NuVotifier)

**Protocol:**
1. TCP connection to votifier port
2. Server sends: `VOTIFIER 2 {challenge}`
3. Client builds payload:
   ```json
   {
     "username": "{username}",
     "serviceName": "{service}",
     "timestamp": {unix_ms},
     "address": "{ip}",
     "challenge": "{challenge}"
   }
   ```
4. Sign payload with HMAC-SHA256 using token
5. Send: `0x733A` (magic) + payload length (2 bytes) + JSON packet
6. Read response: `{"status": "ok"}`

---

## Data Structures

### ServerStatus
```go
type ServerStatus struct {
    Online     bool       `json:"online"`
    Host       string     `json:"host"`
    IP         string     `json:"ip"`
    Port       int        `json:"port"`
    RawIP      string     `json:"raw_ip,omitempty"`
    Ping       int        `json:"ping,omitempty"`
    Version    string     `json:"version"`
    Players    Players    `json:"players"`
    MOTD       MOTD       `json:"motd"`
    Icon       string     `json:"icon,omitempty"`
    CachedAt   int64      `json:"cached_at,omitempty"`
    Error      string     `json:"error,omitempty"`
}

type Players struct {
    Online int `json:"online"`
    Max    int `json:"max"`
}

type MOTD struct {
    Clean string `json:"clean"`
    Raw   string `json:"raw"`
    HTML  string `json:"html"`
}
```

### VoteRequest
```go
type VoteRequest struct {
    ServerIP       string `json:"server_ip"`
    ServerPort     int    `json:"server_port"`
    VotifierHost   string `json:"votifier_host,omitempty"`
    VotifierPort   int    `json:"votifier_port"`
    VotifierMethod string `json:"votifier_method"`
    VotifierToken  string `json:"votifier_token"`
    Username       string `json:"username"`
    IPAddress      string `json:"ip_address"`
    Service        string `json:"service"`
}
```

### VoteResponse
```go
type VoteResponse struct {
    Success bool   `json:"success"`
    Error   string `json:"error,omitempty"`
}
```

---

## Project Structure

```
minenepal-backend/
│
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
│
├── internal/
│   │
│   ├── config/
│   │   └── config.go               # Configuration loading from ENV
│   │
│   ├── handler/
│   │   ├── health.go               # GET /
│   │   ├── status.go               # GET /api/server/status/*
│   │   ├── banner.go              # GET /api/server/banner/*
│   │   ├── vote.go                # POST /api/vote
│   │   ├── websocket.go           # GET /ws
│   │   └── metrics.go             # GET /metrics
│   │
│   ├── middleware/
│   │   └── middleware.go           # CORS, compression, logging
│   │
│   ├── service/
│   │   ├── server.go              # Minecraft server querying
│   │   ├── votifier.go            # Votifier v1/v2 protocols
│   │   ├── cache.go               # In-memory + file cache
│   │   └── banner.go              # Banner generation
│   │
│   ├── websocket/
│   │   └── hub.go                 # WebSocket hub & client management
│   │
│   └── response/
│       └── response.go             # Standardized API responses
│
├── pkg/
│   └── types/
│       └── types.go                # Shared types & interfaces
│
├── cache/                          # Generated cache files
│   ├── icons/                      # Server icons
│   ├── banners/                   # Generated banners
│   └── status/                    # Server status JSON
│
├── go.mod                          # Go module definition
├── go.sum                          # Dependency checksums
├── .env.example                    # Environment variables example
├── Dockerfile                      # Docker container
└── README.md                       # Documentation
```

---

## Configuration (.env)

```env
# Server
PORT=10000
HOST=0.0.0.0

# Cache
CACHE_TTL=15s
BANNER_CACHE_TTL=24h
CACHE_DIR=./cache

# Timeouts
SERVER_QUERY_TIMEOUT=5s
VOTIFIER_TIMEOUT=5s

# Logging
LOG_LEVEL=info
LOG_PRETTY=true
```

---

## Implementation Tasks

### Phase 1: Setup & Foundation

- [ ] Initialize Go module (`go mod init`)
- [ ] Create project structure
- [ ] Add dependencies (Fiber, Zerolog, etc.)
- [ ] Implement configuration loading
- [ ] Set up basic Fiber server
- [ ] Configure logging middleware
- [ ] Create response helpers

### Phase 2: Core Services

- [ ] Implement CacheService (in-memory + file)
- [ ] Implement ServerService (Minecraft query)
- [ ] Implement BannerService (SVG → WebP)
- [ ] Implement VotifierService (v1 + v2)

### Phase 3: API Handlers

- [ ] Health endpoint (`GET /`)
- [ ] Status endpoint (`GET /api/server/status/:ip[:port]`)
- [ ] Bulk status endpoint (`GET /api/server/status/bulk`)
- [ ] Banner endpoint (`GET /api/server/banner/:ip[:port]`)
- [ ] Vote endpoint (`POST /api/vote`)
- [ ] WebSocket endpoint (`GET /ws`)
- [ ] Metrics endpoint (`GET /metrics`)

### Phase 4: WebSocket

- [ ] Implement Hub (client registry)
- [ ] Implement subscribe/unsubscribe
- [ ] Broadcast on server status change
- [ ] Broadcast on vote notification

### Phase 5: Testing & Polish

- [ ] Write unit tests (cache, votifier, server query)
- [ ] Add integration tests
- [ ] Performance optimization
- [ ] Docker setup
- [ ] Documentation

### Phase 6: Laravel Integration

- [ ] Update ProcessVotifierVote job
- [ ] Add backend URL configuration
- [ ] Test end-to-end vote flow

---

## Dependencies

```go
require (
    github.com/gofiber/fiber/v2           // Web framework
    github.com/gofiber/websocket/v2       // WebSocket support
    github.com/rs/zerolog                 // Structured logging
    github.com/disintegration/imaging     // Image processing
)
```

---

## Error Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 400 | Invalid request (bad ip/port, missing fields) |
| 404 | Server offline or not found |
| 500 | Internal error (votifier failed, query failed) |
| 500 | Vote failed (server refused packet) |

---

## Performance Targets

- Server query: < 100ms (excluding network)
- Cache hit: < 5ms response
- Banner generation: < 50ms
- WebSocket message: < 10ms broadcast
- Memory usage: < 100MB under normal load

---

## Security Considerations

1. **Input Validation** - Sanitize all IPs/ports
2. **Timeout Enforcement** - Prevent hanging connections
3. **Cache Limits** - Prevent memory exhaustion
4. **No Rate Limiting** - Handled by MineNepal
5. **Structured Logging** - Audit trail for votes

---

## Monitoring

- Health endpoint for load balancer checks
- Prometheus metrics for monitoring
- Structured JSON logs for debugging
- WebSocket client count tracking

---

## Backward Compatibility

All existing API endpoints maintain 100% compatibility:
- Same URL paths
- Same response formats
- Same HTTP status codes

New endpoints follow the same patterns for consistency.