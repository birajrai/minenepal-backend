# MineNepal Server Status API Documentation

## Overview

The MineNepal Server Status API provides real-time information about Minecraft servers, including online status, player counts, MOTD, ping, and server icons. It supports both individual server queries and bulk requests, with automatic caching and banner generation.

**Base URLs:**
- Production Server 1: `https://mnsc1.bishestamedia.com.np/`
- Production Server 2: `https://mnsc2.bishestamedia.com.np/`
- Local Development: `http://localhost:10000` (default port)

**Version:** 1.0.0

**Content Type:** JSON (except for banner endpoints which return WebP images)

## Endpoints

### Health Check

#### GET /

Returns system health metrics and API status.

**Response:**
```json
{
  "status": "healthy",
  "uptime": "1234.56 seconds",
  "responseTime": "15.23 ms",
  "requests": 42,
  "memory": {
    "rss": "45.67 MB",
    "heapUsed": "23.45 MB",
    "heapTotal": "67.89 MB",
    "external": "1.23 MB"
  },
  "cpu": {
    "usage": "15.67%",
    "loadAverage": ["1.23", "1.45", "1.67"]
  },
  "storage": "12.34 MB"
}
```

### Single Server Status

#### GET /api/server/status/{ip}

Get the status of a Minecraft server at the default port (25565).

**Parameters:**
- `ip` (path): Server IP address or hostname

**Response (Online Server):**
```json
{
  "online": true,
  "host": "play.hypixel.net",
  "ip": "play.hypixel.net",
  "port": 25565,
  "raw_ip": "8.8.8.8",
  "ping": 45,
  "version": "Requires MC 1.8 / 1.18",
  "players": {
    "online": 12345,
    "max": 200000
  },
  "motd": {
    "clean": "Hypixel Network",
    "raw": "§cHypixel Network",
    "html": "<span style=\"color: #ff5555\">Hypixel Network</span>"
  },
  "icon": "/icons/play.hypixel.net_25565.png"
}
```

**Response (Offline Server):**
```json
{
  "online": false,
  "host": "offline.example.com",
  "ip": "offline.example.com",
  "port": 25565,
  "raw_ip": null,
  "ping": null,
  "error": "Server offline or unreachable",
  "icon": null
}
```

#### GET /api/server/status/{ip}/{port}

Get the status of a Minecraft server at a specific port.

**Parameters:**
- `ip` (path): Server IP address or hostname
- `port` (path): Server port number

**Response:** Same as above, with the specified port.

### Bulk Server Status

#### GET /api/server/status/bulk

Get the status of multiple Minecraft servers in a single request.

**Query Parameters:**
- `servers` (required): Comma-separated list of servers in format `ip:port` or `ip` (default port 25565)

**Example Request:**
```
GET /api/server/status/bulk?servers=play.hypixel.net,mc.hypixel.net:25566,play.craftnepal.com
```

**Response:**
```json
{
  "play.hypixel.net:25565": {
    "online": true,
    "host": "play.hypixel.net",
    "ip": "play.hypixel.net",
    "port": 25565,
    "raw_ip": "8.8.8.8",
    "ping": 45,
    "version": "Requires MC 1.8 / 1.18",
    "players": {
      "online": 12345,
      "max": 200000
    },
    "motd": {
      "clean": "Hypixel Network",
      "raw": "§cHypixel Network",
      "html": "<span style=\"color: #ff5555\">Hypixel Network</span>"
    },
    "icon": "/icons/play.hypixel.net_25565.png"
  },
  "mc.hypixel.net:25566": {
    "online": false,
    "host": "mc.hypixel.net",
    "ip": "mc.hypixel.net",
    "port": 25566,
    "raw_ip": null,
    "ping": null,
    "error": "Server offline or unreachable",
    "icon": null
  },
  "play.craftnepal.com:25565": {
    "online": true,
    ...
  }
}
```

### Server Banner

#### GET /api/server/banner/{ip}

Generate a visual banner image for a Minecraft server at default port (25565).

**Parameters:**
- `ip` (path): Server IP address or hostname

**Response:** WebP image file

**Banner Features:**
- Server icon (32x32 resized to 64x64)
- MOTD text with colors
- Ping in top-right corner
- Player count (online/max)
- Minecraft version
- Dirt texture background

**Error Response (JSON):**
```json
{
  "error": "Server offline"
}
```

#### GET /api/server/banner/{ip}/{port}

Generate a visual banner for a server at a specific port.

**Parameters:**
- `ip` (path): Server IP address or hostname
- `port` (path): Server port number

**Response:** Same as above.

### Server Icons

#### GET /icons/{filename}

Serve cached server favicon images.

**Parameters:**
- `filename` (path): Icon filename (e.g., `play.hypixel.net_25565.png`)

**Response:** PNG image file

## Response Fields

### Common Fields
- `online` (boolean): Whether the server is reachable
- `host` (string): Original hostname provided
- `ip` (string): Resolved IP address
- `port` (number): Server port
- `raw_ip` (string|null): Raw IP address from DNS lookup

### Online Server Fields
- `ping` (number): Round-trip latency in milliseconds
- `version` (string): Minecraft version string
- `players` (object):
  - `online` (number): Current player count
  - `max` (number): Maximum player capacity
- `motd` (object):
  - `clean` (string): Plain text MOTD
  - `raw` (string): Raw MOTD with color codes
  - `html` (string): HTML-formatted MOTD
- `icon` (string|null): Path to server favicon

### Offline Server Fields
- `ping` (null)
- `error` (string): Error message
- `icon` (null)

## Caching

The API implements multiple caching layers for performance:

- **Memory Cache:** 15-second TTL for recent requests
- **File Cache:** Persisted JSON data for server status
- **Icon Cache:** Server favicons stored as PNG files
- **Banner Cache:** Generated banners stored as WebP files

## Rate Limiting

No explicit rate limiting is implemented. However, responses are cached for 15 seconds to reduce load on both the API and Minecraft servers.

## Error Handling

All endpoints return appropriate HTTP status codes:

- `200`: Success
- `404`: Server offline (for status endpoints) or not found
- `400`: Bad request (missing parameters)
- `500`: Internal server error

Error responses include a JSON object with an `error` field containing a descriptive message.

## Dependencies

- Node.js >= 20.0.0
- Express.js
- Minecraft Server Util
- Sharp (image processing)
- Compression, CORS, HTML entities encoding

## Running the API

```bash
npm install
node minenepal.js
```

The server will start on port 10000 by default.

## Auto-Refresh

The API automatically refreshes status for predefined popular servers:
- play.hypixel.net
- play.craftnepal.com
- mcnpnetwork.com

This ensures these servers' data is always fresh in the cache.</content>
<parameter name="filePath">/home/biraj/Projects/minenepal_codebase/minenepal-server-status/API_DOCS.md