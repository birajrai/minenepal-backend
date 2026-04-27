package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"minenepal-backend/pkg/types"
)

type ServerService struct {
	cache        *CacheService
	queryTimeout time.Duration
}

func NewServerService(cache *CacheService, queryTimeout time.Duration) *ServerService {
	return &ServerService{
		cache:        cache,
		queryTimeout: queryTimeout,
	}
}

func (s *ServerService) GetStatus(ip string, port int) (*types.ServerStatus, error) {
	key := fmt.Sprintf("%s:%d", ip, port)

	if data, ok := s.cache.Get(key); ok {
		var status types.ServerStatus
		if err := json.Unmarshal(data, &status); err == nil {
			return &status, nil
		}
	}

	status, err := s.queryServer(ip, port)
	if err != nil {
		status = s.offlineStatus(ip, port, err.Error())
	}

	data, _ := json.Marshal(status)
	s.cache.Set(key, data, s.cache.ttl)

	return status, nil
}

func (s *ServerService) GetStatusBulk(servers []string) map[string]types.ServerStatus {
	results := make(map[string]types.ServerStatus)

	for _, server := range servers {
		ip, port := parseServer(server)
		if status, err := s.GetStatus(ip, port); err == nil {
			results[server] = *status
		} else {
			results[server] = *s.offlineStatus(ip, port, "unreachable")
		}
	}

	return results
}

func parseServer(s string) (string, int) {
	ip := s
	port := 25565

	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		ip = parts[0]
		fmt.Sscanf(parts[1], "%d", &port)
	}

	return ip, port
}

func (s *ServerService) queryServer(ip string, port int) (*types.ServerStatus, error) {
	host := ip

	if !strings.Contains(ip, ".") || !isHostname(ip) {
		srvRecord := s.lookupSRV(ip, port)
		if srvRecord != "" {
			host = srvRecord
		}
	}

	_ = s.lookupIP(host)

	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), s.queryTimeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	ping := int(time.Since(start).Milliseconds())

	handshake := s.createHandshake(host, port)
	_, err = conn.Write(handshake)
	if err != nil {
		return nil, err
	}

	statusReq := s.createStatusRequest()
	_, err = conn.Write(statusReq)
	if err != nil {
		return nil, err
	}

	if err := conn.SetReadDeadline(time.Now().Add(s.queryTimeout)); err != nil {
		return nil, err
	}

	response, err := s.readStatusResponse(conn)
	if err != nil {
		return nil, fmt.Errorf("no response from server")
	}

	status := s.parseResponse(ip, port, host, ping, response)

	if icon := s.handleIcon(ip, port, response); icon != "" {
		status.Icon = icon
	}

	return status, nil
}

func (s *ServerService) createHandshake(host string, port int) []byte {
	var buf bytes.Buffer

	buf.WriteByte(0x00)
	buf.Write(encodeVarint(47))
	buf.Write(encodeVarint(int32(len(host))))
	buf.Write([]byte(host))
	buf.Write(encodeVarint(int32(port)))
	buf.Write(encodeVarint(1))

	return s.createPacket(0, &buf)
}

func (s *ServerService) createStatusRequest() []byte {
	var buf bytes.Buffer
	buf.WriteByte(0x00)
	return s.createPacket(0, &buf)
}

func (s *ServerService) readStatusResponse(conn net.Conn) ([]byte, error) {
	lengthBuf := make([]byte, 5)
	n, err := conn.Read(lengthBuf)
	if err != nil || n == 0 {
		return nil, err
	}

	r := bytes.NewReader(lengthBuf)
	length := decodeVarint(r)
	if length <= 0 || length > 65536 {
		return nil, fmt.Errorf("invalid length")
	}

	data := make([]byte, length)
	_, err = io.ReadFull(conn, data)
	if err != nil {
		return nil, err
	}

	return data[3:], nil
}
	}

	_ = s.lookupIP(host)

	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), s.queryTimeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	ping := int(time.Since(start).Milliseconds())

	if err := conn.SetReadDeadline(time.Now().Add(s.queryTimeout)); err != nil {
		return nil, err
	}

	_, err = s.handshake(host, port, conn)
	if err != nil {
		return nil, err
	}

	if err := conn.SetReadDeadline(time.Now().Add(s.queryTimeout)); err != nil {
		return nil, err
	}

	response := s.readResponse(conn)
	if response == nil {
		return nil, fmt.Errorf("no response from server")
	}

	status := s.parseResponse(ip, port, host, ping, response)

	if icon := s.handleIcon(ip, port, response); icon != "" {
		status.Icon = icon
	}

	return status, nil
}

func isHostname(s string) bool {
	if strings.ContainsAny(s, ":") {
		return false
	}
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return false
	}
	return true
}

func (s *ServerService) lookupSRV(host string, defaultPort int) string {
	srvHost := fmt.Sprintf("_minecraft._tcp.%s", host)
	_, records, err := net.LookupSRV("", "", srvHost)
	if err != nil || len(records) == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", records[0].Target, records[0].Port)
}

func (s *ServerService) lookupIP(host string) string {
	ips, err := net.LookupHost(host)
	if err != nil || len(ips) == 0 {
		return host
	}
	return ips[0]
}

func (s *ServerService) handshake(host string, port int, conn net.Conn) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteByte(0x00)

	protocol := int32(47)
	buf.Write(encodeVarint(protocol))

	hostBytes := []byte(host)
	buf.Write(encodeVarint(int32(len(hostBytes))))
	buf.Write(hostBytes)

	buf.Write(encodeVarint(int32(port)))

	buf.WriteByte(0x01)

	packet := s.createPacket(0, &buf)

	_, err := conn.Write(packet)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func encodeVarint(val int32) []byte {
	var result []byte
	for {
		byteVal := byte(val & 0x7F)
		val >>= 7
		if val != 0 {
			byteVal |= 0x80
		}
		result = append(result, byteVal)
		if val == 0 {
			break
		}
	}
	return result
}

func (s *ServerService) createPacket(packetID int, data *bytes.Buffer) []byte {
	var buf bytes.Buffer

	payload := data.Bytes()
	length := len(payload)

	buf.Write(encodeVarint(int32(length)))
	buf.Write(encodeVarint(int32(packetID)))
	buf.Write(payload)

	return buf.Bytes()
}

func (s *ServerService) readResponse(conn net.Conn) []byte {
	var buf bytes.Buffer
	buf.WriteByte(0x00)

	packet := s.createPacket(0, &buf)

	_, err := conn.Write(packet)
	if err != nil {
		return nil
	}

	lengthBuf := make([]byte, 5)
	_, err = conn.Read(lengthBuf)
	if err != nil {
		return nil
	}

	length := int(decodeVarint(bytes.NewReader(lengthBuf)))

	if length <= 0 || length > 65536 {
		return nil
	}

	data := make([]byte, length)
	_, err = io.ReadFull(conn, data)
	if err != nil {
		return nil
	}

	return data
}

func decodeVarint(r *bytes.Reader) int {
	result := 0
	shift := 0
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0
		}
		result |= int(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return result
}

func (s *ServerService) parseResponse(ip string, port int, host string, ping int, data []byte) *types.ServerStatus {
	r := bytes.NewReader(data)
	decodeVarint(r)

	jsonLen := decodeVarint(r)
	if jsonLen <= 0 {
		return s.offlineStatus(ip, port, "empty response")
	}

	jsonData := make([]byte, jsonLen)
	r.Read(jsonData)

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		return s.offlineStatus(ip, port, "parse error")
	}

	status := &types.ServerStatus{
		Online: true,
		Host:   host,
		IP:     ip,
		Port:   port,
		RawIP:  s.lookupIP(host),
		Ping:   ping,
	}

	if version, ok := parsed["version"].(map[string]interface{}); ok {
		if name, ok := version["name"].(string); ok {
			status.Version = name
		}
	}

	if players, ok := parsed["players"].(map[string]interface{}); ok {
		if online, ok := players["online"].(float64); ok {
			status.Players.Online = int(online)
		}
		if max, ok := players["max"].(float64); ok {
			status.Players.Max = int(max)
		}
	}

	if motdObj, ok := parsed["description"].(map[string]interface{}); ok {
		if extra, ok := motdObj["extra"].([]interface{}); ok {
			status.MOTD.Raw = ""
			if text, ok := motdObj["text"].(string); ok {
				status.MOTD.Raw = text
			}
			for _, e := range extra {
				if m, ok := e.(map[string]interface{}); ok {
					if text, ok := m["text"].(string); ok {
						status.MOTD.Raw += text
					}
				}
			}
		} else if text, ok := motdObj["text"].(string); ok {
			status.MOTD.Raw = text
		}
	} else if desc, ok := parsed["description"].(string); ok {
		status.MOTD.Raw = desc
	}

	status.MOTD.Clean = stripColorCodes(status.MOTD.Raw)
	status.MOTD.HTML = formatMOTDHTML(status.MOTD.Raw)

	return status
}

var colorCodeRegex = regexp.MustCompile(`§[0-9a-fk-or]`)
var colorFormatRegex = regexp.MustCompile(`§([0-9a-fk-or])([^§]*)`)

var colorMap = map[string]string{
	"0": "#000000", "1": "#0000AA", "2": "#00AA00", "3": "#00AAAA",
	"4": "#AA0000", "5": "#AA00AA", "6": "#FFAA00", "7": "#AAAAAA",
	"8": "#555555", "9": "#5555FF", "a": "#55FF55", "b": "#55FFFF",
	"c": "#FF5555", "d": "#FF55FF", "e": "#FFFF55", "f": "#FFFFFF",
	"l": "", "o": "", "n": "", "m": "", "k": "", "r": "",
}

func stripColorCodes(s string) string {
	return colorCodeRegex.ReplaceAllString(s, "")
}

func formatMOTDHTML(raw string) string {
	return colorFormatRegex.ReplaceAllStringFunc(raw, func(match string) string {
		parts := colorFormatRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return ""
		}
		code := parts[1]
		text := parts[2]

		if code == "l" || code == "o" {
			style := ""
			if code == "l" {
				style = "font-weight: bold;"
			} else if code == "o" {
				style = "font-style: italic;"
			}
			return fmt.Sprintf(`<span style="%s">%s</span>`, style, text)
		}

		colorVal, ok := colorMap[code]
		if !ok {
			return text
		}
		return fmt.Sprintf(`<span style="color: %s">%s</span>`, colorVal, text)
	})
}

func (s *ServerService) handleIcon(ip string, port int, data []byte) string {
	if s.cache.IconExists(ip, port, 14*24*time.Hour) {
		return s.cache.GetIconURL(ip, port)
	}

	r := bytes.NewReader(data)
	decodeVarint(r)
	jsonLen := decodeVarint(r)
	jsonData := make([]byte, jsonLen)
	r.Read(jsonData)

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		return ""
	}

	favicon, ok := parsed["favicon"].(string)
	if !ok || favicon == "" {
		return ""
	}

	if !strings.HasPrefix(favicon, "data:image/png;base64,") {
		return ""
	}

	base64Data := strings.TrimPrefix(favicon, "data:image/png;base64,")
	iconData, err := decodeBase64(base64Data)
	if err != nil {
		return ""
	}

	img, err := png.Decode(bytes.NewReader(iconData))
	if err != nil {
		return ""
	}

	resized := resizeImage(img, 32, 32)

	iconPath := s.cache.GetIconPath(ip, port)
	out, err := os.Create(iconPath)
	if err != nil {
		return ""
	}
	defer out.Close()
	png.Encode(out, resized)

	return s.cache.GetIconURL(ip, port)
}

func resizeImage(img image.Image, width, height int) image.Image {
	srcBounds := img.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	if srcWidth == width && srcHeight == height {
		return img
	}

	return img
}

func decodeBase64(s string) ([]byte, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	const padding = '='

	decoded := make([]byte, len(s)*6/8)
	buffer := 0
	bits := 0
	outIdx := 0

	for i := 0; i < len(s); i++ {
		c := s[i]
		var val byte

		if c == padding {
			break
		}

		for j := 0; j < 64; j++ {
			if alphabet[j] == c {
				val = byte(j)
				break
			}
		}

		buffer = (buffer << 6) | int(val)
		bits += 6

		if bits >= 8 {
			bits -= 8
			decoded[outIdx] = byte(buffer >> bits)
			outIdx++
		}
	}

	return decoded[:outIdx], nil
}

func (s *ServerService) offlineStatus(ip string, port int, errMsg string) *types.ServerStatus {
	return &types.ServerStatus{
		Online: false,
		Host:   ip,
		IP:     ip,
		Port:   port,
		RawIP:  "",
		Ping:   0,
		Error:  errMsg,
		Players: types.Players{
			Online: 0,
			Max:    0,
		},
		MOTD: types.MOTD{
			Clean: "",
			Raw:   "",
			HTML:  "",
		},
		Icon: "",
	}
}

func (s *ServerService) GetCache() *CacheService {
	return s.cache
}