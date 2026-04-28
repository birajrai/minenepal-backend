package types

import "bytes"
import "encoding/json"

type HTMLString string

func (s HTMLString) MarshalJSON() ([]byte, error) {
	// Manually build JSON string without HTML escaping
	var buf bytes.Buffer
	buf.WriteByte('"')
	for _, r := range string(s) {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7f {
				// Escape control characters as \uXXXX
				fmt.Fprintf(&buf, `\u%04x`, r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
	return buf.Bytes(), nil
}

type Players struct {
	Online int `json:"online"`
	Max    int `json:"max"`
}

type MOTD struct {
	Clean string      `json:"clean"`
	Raw   string      `json:"raw"`
	HTML  HTMLString  `json:"html"`
}

type ServerStatus struct {
	Online   bool    `json:"online"`
	Host     string  `json:"host"`
	IP       string  `json:"ip"`
	Port     int     `json:"port"`
	RawIP    string  `json:"raw_ip,omitempty"`
	Ping     int     `json:"ping,omitempty"`
	Version  string  `json:"version"`
	Players  Players `json:"players"`
	MOTD     MOTD    `json:"motd"`
	Icon     string  `json:"icon,omitempty"`
	CachedAt int64   `json:"cached_at,omitempty"`
	Error    string  `json:"error,omitempty"`
}

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

type VoteResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type HealthResponse struct {
	Status       string        `json:"status"`
	Uptime       string        `json:"uptime"`
	ResponseTime string        `json:"responseTime"`
	Requests     int64         `json:"requests"`
	Memory       MemoryMetrics `json:"memory"`
	CPU          CPUMetrics    `json:"cpu"`
	Storage      string        `json:"storage"`
}

type MemoryMetrics struct {
	RSS       string `json:"rss"`
	HeapUsed  string `json:"heapUsed"`
	HeapTotal string `json:"heapTotal"`
	External  string `json:"external"`
}

type CPUMetrics struct {
	Usage       string   `json:"usage"`
	LoadAverage []string `json:"loadAverage"`
}

type CachedEntry struct {
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type WSMessage struct {
	Type   string      `json:"type"`
	Server string      `json:"server,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

type WSSubscription struct {
	Action  string   `json:"action"`
	Servers []string `json:"servers,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
