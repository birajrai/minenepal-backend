package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CacheEntry struct {
	Timestamp int64  `json:"timestamp"`
	Data      []byte `json:"data"`
}

type CacheService struct {
	mu       sync.RWMutex
	memory   map[string]*CacheEntry
	cacheDir string
	ttl      time.Duration
}

func NewCacheService(cacheDir string, ttl time.Duration) *CacheService {
	c := &CacheService{
		memory:   make(map[string]*CacheEntry),
		cacheDir: cacheDir,
		ttl:      ttl,
	}
	c.ensureDir(cacheDir)
	c.ensureDir(filepath.Join(cacheDir, "icons"))
	c.ensureDir(filepath.Join(cacheDir, "banners"))
	c.ensureDir(filepath.Join(cacheDir, "status"))
	return c
}

func (c *CacheService) ensureDir(path string) {
	os.MkdirAll(path, 0755)
}

func (c *CacheService) keyToFilename(key string) string {
	hash := sha256.Sum256([]byte(key))
	hashStr := hex.EncodeToString(hash[:])
	return filepath.Join(c.cacheDir, "status", hashStr+".json")
}

func (c *CacheService) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	entry, exists := c.memory[key]
	c.mu.RUnlock()

	if !exists {
		filename := c.keyToFilename(key)
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, false
		}

		entry = &CacheEntry{}
		if err := json.Unmarshal(data, entry); err != nil {
			return nil, false
		}

		c.mu.Lock()
		c.memory[key] = entry
		c.mu.Unlock()
	}

	if time.Now().UnixMilli()-entry.Timestamp > c.ttl.Milliseconds() {
		return nil, false
	}

	return entry.Data, true
}

func (c *CacheService) Set(key string, data []byte, ttl time.Duration) {
	entry := &CacheEntry{
		Timestamp: time.Now().UnixMilli(),
		Data:      data,
	}

	c.mu.Lock()
	c.memory[key] = entry
	c.mu.Unlock()

	filename := c.keyToFilename(key)
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return
	}
	os.WriteFile(filename, jsonData, 0644)
}

func (c *CacheService) Delete(key string) {
	c.mu.Lock()
	delete(c.memory, key)
	c.mu.Unlock()

	filename := c.keyToFilename(key)
	os.Remove(filename)
}

func (c *CacheService) GetIconPath(ip string, port int) string {
	filename := sanitizeFilename(ip) + "_" + fmt.Sprintf("%d", port) + ".png"
	return filepath.Join(c.cacheDir, "icons", filename)
}

func (c *CacheService) GetIconURL(ip string, port int) string {
	filename := sanitizeFilename(ip) + "_" + fmt.Sprintf("%d", port) + ".png"
	return "/icons/" + filename
}

func (c *CacheService) GetBannerPath(ip string, port int) string {
	filename := sanitizeFilename(ip) + "_" + fmt.Sprintf("%d", port) + ".webp"
	return filepath.Join(c.cacheDir, "banners", filename)
}

func (c *CacheService) GetBannerURL(ip string, port int) string {
	filename := sanitizeFilename(ip) + "_" + fmt.Sprintf("%d", port) + ".webp"
	return "/banners/" + filename
}

func (c *CacheService) IconExists(ip string, port int, maxAge time.Duration) bool {
	path := c.GetIconPath(ip, port)
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(stat.ModTime()) < maxAge
}

func (c *CacheService) BannerExists(ip string, port int, maxAge time.Duration) bool {
	path := c.GetBannerPath(ip, port)
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(stat.ModTime()) < maxAge
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return s
}

func (c *CacheService) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixMilli()
	for key, entry := range c.memory {
		if now-entry.Timestamp > c.ttl.Milliseconds() {
			delete(c.memory, key)
		}
	}
}