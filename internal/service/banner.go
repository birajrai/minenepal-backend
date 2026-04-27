package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"minenepal-backend/pkg/types"
)

type BannerService struct {
	cache          *CacheService
	bannerCacheTTL time.Duration
}

func NewBannerService(cache *CacheService, bannerCacheTTL time.Duration) *BannerService {
	return &BannerService{
		cache:          cache,
		bannerCacheTTL: bannerCacheTTL,
	}
}

func (b *BannerService) GenerateBanner(status *types.ServerStatus) ([]byte, error) {
	width := 800
	height := 120
	iconSize := 64
	padding := 10

	var svg strings.Builder

	svg.WriteString(fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">`, width, height))

	svg.WriteString(`<defs><linearGradient id="grad" x1="0%" y1="0%" x2="0%" y2="100%">`)
	svg.WriteString(`<stop offset="0%" style="stop-color:#8B4513;stop-opacity:1" />`)
	svg.WriteString(`<stop offset="100%" style="stop-color:#654321;stop-opacity:1" />`)
	svg.WriteString(`</linearGradient></defs>`)
	svg.WriteString(`<rect width="100%" height="100%" fill="url(#grad)"/>`)

	if status.Icon != "" {
		svg.WriteString(fmt.Sprintf(`<image href="%s" x="%d" y="%d" width="%d" height="%d"/>`, status.Icon, padding, padding, iconSize, iconSize))
	}

	motdText := status.MOTD.Clean
	if motdText == "" {
		motdText = status.Host
	}

	textX := iconSize + padding*2

	svg.WriteString(fmt.Sprintf(`<text x="%d" y="50" font-family="monospace" font-size="12" text-anchor="start">`, textX))
	svg.WriteString(escapeXML(motdText))
	svg.WriteString(`</text>`)

	if status.Ping > 0 {
		pingColor := "#00ff00"
		if status.Ping > 200 {
			pingColor = "#ffff00"
		} else if status.Ping > 500 {
			pingColor = "#ff0000"
		}
		svg.WriteString(fmt.Sprintf(`<text x="%d" y="25" fill="%s" font-family="Arial, sans-serif" font-size="12" text-anchor="end">%dms</text>`, width-padding, pingColor, status.Ping))
	}

	playersText := fmt.Sprintf("%d/%d players", status.Players.Online, status.Players.Max)
	svg.WriteString(fmt.Sprintf(`<text x="%d" y="80" fill="#cccccc" font-family="Arial, sans-serif" font-size="12" text-anchor="start">%s</text>`, textX, escapeXML(playersText)))

	if status.Version != "" {
		versionText := truncateString(status.Version, 30)
		svg.WriteString(fmt.Sprintf(`<text x="%d" y="100" fill="#aaaaaa" font-family="Arial, sans-serif" font-size="10" text-anchor="start">%s</text>`, textX, escapeXML(versionText)))
	}

	svg.WriteString(`</svg>`)

	return []byte(svg.String()), nil
}

func (b *BannerService) GetBanner(ip string, port int, status *types.ServerStatus) ([]byte, error) {
	if b.cache.BannerExists(ip, port, b.bannerCacheTTL) {
		bannerPath := b.cache.GetBannerPath(ip, port)
		return os.ReadFile(bannerPath)
	}

	bannerData, err := b.GenerateBanner(status)
	if err != nil {
		return nil, err
	}

	bannerPath := b.cache.GetBannerPath(ip, port)
	if err := os.WriteFile(bannerPath, bannerData, 0644); err != nil {
		return nil, err
	}

	return bannerData, nil
}

func (b *BannerService) GetBannerPath() string {
	return filepath.Join(b.cache.cacheDir, "banners")
}

func escapeXML(s string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	s = re.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}