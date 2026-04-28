package service

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
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
	padding := 16

	var svg strings.Builder

	svg.WriteString(fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">`, width, height))

	svg.WriteString(`<defs>
		<linearGradient id="bg" x1="0%" y1="0%" x2="100%" y2="100%">
			<stop offset="0%" style="stop-color:#1a1a2e;stop-opacity:1" />
			<stop offset="100%" style="stop-color:#16213e;stop-opacity:1" />
		</linearGradient>
		<linearGradient id="accent" x1="0%" y1="0%" x2="0%" y2="100%">
			<stop offset="0%" style="stop-color:#e94560;stop-opacity:1" />
			<stop offset="100%" style="stop-color:#c73e54;stop-opacity:1" />
		</linearGradient>
		<filter id="shadow" x="-20%" y="-20%" width="140%" height="140%">
			<feDropShadow dx="1" dy="1" stdDeviation="1" flood-opacity="0.3"/>
		</filter>
	</defs>`)

	svg.WriteString(`<rect width="100%" height="100%" fill="url(#bg)"/>`)
	svg.WriteString(`<rect width="4" height="100%" fill="url(#accent)"/>`)

	iconY := (height - iconSize) / 2
	if status.Icon != "" {
		iconPath := strings.TrimPrefix(status.Icon, "/icons/")
		fullIconPath := filepath.Join(b.cache.cacheDir, "icons", iconPath)
		if data, err := os.ReadFile(fullIconPath); err == nil {
			encoded := base64.StdEncoding.EncodeToString(data)
			svg.WriteString(fmt.Sprintf(`<image x="%d" y="%d" width="%d" height="%d" href="data:image/png;base64,%s"/>`, padding+4, iconY-4, iconSize, iconSize, encoded))
		}
	}

	motdText := status.MOTD.Clean
	if motdText == "" {
		motdText = status.Host
	}
	motdText = strings.ReplaceAll(motdText, "\n", " ")
	motdText = escapeXML(motdText)

	minWidth := 60
	if len(motdText) < minWidth {
		motdText = motdText + strings.Repeat(" ", minWidth-len(motdText))
	}

	textX := iconSize + padding + 4
	centerY := height / 2

	svg.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Consolas, Monaco, monospace" font-size="15" fill="#ffffff" font-weight="bold" filter="url(#shadow)">%s</text>`, textX, centerY-25, motdText))

	pingY := padding + 12
	if status.Ping > 0 {
		pingColor := "#4ade80"
		pingText := fmt.Sprintf("%dms", status.Ping)
		if status.Ping > 300 {
			pingColor = "#fbbf24"
			pingText = fmt.Sprintf("%dms", status.Ping)
		}
		if status.Ping > 600 {
			pingColor = "#f87171"
		}
		svg.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Consolas, Monaco, monospace" font-size="11" fill="%s" text-anchor="end" font-weight="bold">%s</text>`, width-padding, pingY, pingColor, pingText))

		pingDotY := pingY - 4
		dotColor := pingColor
		if status.Ping > 0 {
			svg.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="3" fill="%s"/><circle cx="%d" cy="%d" r="3" fill="%s" opacity="0.4"><circle cx="%d" cy="%d" r="3" fill="%s" opacity="0.2"/></circle>`, width-padding-45, pingDotY, dotColor, width-padding-45, pingDotY, dotColor, width-padding-45, pingDotY, dotColor))
		}
	}

	playersText := fmt.Sprintf("Players: %d / %d", status.Players.Online, status.Players.Max)
	svg.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Consolas, Monaco, monospace" font-size="11" fill="#94a3b8">%s</text>`, textX, centerY+15, playersText))

	if status.Version != "" {
		versionText := status.Version
		if len(versionText) > 40 {
			versionText = versionText[:40] + "..."
		}
		svg.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Consolas, Monaco, monospace" font-size="10" fill="#64748b">%s</text>`, textX, height-padding, escapeXML(versionText)))
	}

	bgHost := fmt.Sprintf("bg: %s:%d", status.IP, status.Port)
	svg.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Consolas, Monaco, monospace" font-size="9" fill="#475569" text-anchor="end">%s</text>`, width-padding, height-4, bgHost))

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
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
