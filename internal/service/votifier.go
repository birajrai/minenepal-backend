package service

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"strings"
	"time"

	"minenepal-backend/pkg/types"
)

type VotifierService struct {
	timeout time.Duration
}

func NewVotifierService(timeout time.Duration) *VotifierService {
	return &VotifierService{
		timeout: timeout,
	}
}

func (v *VotifierService) SendVote(req *types.VoteRequest) error {
	if req.VotifierMethod == "v2" {
		return v.sendV2(req)
	}
	return v.sendV1(req)
}

func (v *VotifierService) sendV1(req *types.VoteRequest) error {
	if req.VotifierToken == "" {
		return fmt.Errorf("votifier v1 requires public key")
	}

	publicKey, err := parsePublicKey(req.VotifierToken)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	host := req.VotifierHost
	if host == "" {
		host = req.ServerIP
	}
	port := req.VotifierPort
	if port == 0 {
		port = 8192
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, v.timeout)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	_, err = reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read server banner: %w", err)
	}

	serviceName := req.Service
	if serviceName == "" {
		serviceName = "MineNepal"
	}

	timestamp := time.Now().Unix()
	payload := fmt.Sprintf("VOTE\n%s\n%s\n%s\n%d\n", serviceName, req.Username, req.IPAddress, timestamp)

	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, []byte(payload))
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	_, err = conn.Write(encrypted)
	if err != nil {
		return fmt.Errorf("failed to send vote: %w", err)
	}

	return nil
}

func (v *VotifierService) sendV2(req *types.VoteRequest) error {
	if req.VotifierToken == "" {
		return fmt.Errorf("votifier v2 requires token")
	}

	host := req.VotifierHost
	if host == "" {
		host = req.ServerIP
	}
	port := req.VotifierPort
	if port == 0 {
		port = 8192
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, v.timeout)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	header, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, "VOTIFIER 2") {
		return fmt.Errorf("invalid votifier v2 header: %s", header)
	}

	parts := strings.Split(header, " ")
	if len(parts) < 3 {
		return fmt.Errorf("invalid challenge format")
	}
	challenge := parts[2]

	serviceName := req.Service
	if serviceName == "" {
		serviceName = "MineNepal"
	}

	timestamp := time.Now().UnixMilli()

	payloadData := map[string]interface{}{
		"username":     req.Username,
		"serviceName":  serviceName,
		"timestamp":    timestamp,
		"address":      req.IPAddress,
		"challenge":    challenge,
	}

	jsonPayload, err := json.Marshal(payloadData)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	signature := computeHMAC(jsonPayload, req.VotifierToken)

	packet := map[string]string{
		"payload":  base64.StdEncoding.EncodeToString(jsonPayload),
		"signature": signature,
	}

	jsonPacket, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("failed to marshal packet: %w", err)
	}

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint16(0x733A))
	binary.Write(&buf, binary.BigEndian, uint16(len(jsonPacket)))
	buf.Write(jsonPacket)

	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send vote: %w", err)
	}

	responseRaw, err := reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if status, ok := response["status"].(string); ok && status == "ok" {
		return nil
	}

	return fmt.Errorf("votifier server error: %s", responseRaw)
}

func parsePublicKey(keyData string) (*rsa.PublicKey, error) {
	keyData = strings.ReplaceAll(keyData, "\n", "")
	keyData = strings.ReplaceAll(keyData, "\r", "")
	keyData = strings.ReplaceAll(keyData, "-----BEGIN PUBLIC KEY-----", "")
	keyData = strings.ReplaceAll(keyData, "-----END PUBLIC KEY-----", "")

	keyData = strings.TrimSpace(keyData)

	block, _ := pem.Decode([]byte(keyData))
	if block != nil {
		keyData = string(block.Bytes)
	}

	keyData = strings.TrimSpace(keyData)

	paddedKey := "-----BEGIN PUBLIC KEY-----\n"
	paddedKey += insertNewlines(keyData, 64)
	paddedKey += "\n-----END PUBLIC KEY-----"

	block, err := pem.Decode([]byte(paddedKey))
	if err != nil {
		return nil, fmt.Errorf("failed to decode PEM: %w", err)
	}

	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	pubKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return pubKey, nil
}

func insertNewlines(s string, every int) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && i%every == 0 {
			result.WriteRune('\n')
		}
		result.WriteRune(r)
	}
	return result.String()
}

func computeHMAC(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (v *VotifierService) GetTimeout() time.Duration {
	return v.timeout
}