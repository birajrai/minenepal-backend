package service

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
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
	img := image.NewRGBA(image.Rect(0, 0, 800, 120))

	for y := 0; y < 120; y++ {
		for x := 0; x < 800; x++ {
			img.Set(x, y, color.RGBA{139, 69, 19, 255})
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)

	return buf.Bytes(), nil
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