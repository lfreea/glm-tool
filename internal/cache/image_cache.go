package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"glm-tool/config"

	cacheconfig "github.com/gophertool/tool/db/cache/config"
	_interface "github.com/gophertool/tool/db/cache/interface"
	"github.com/gophertool/tool/log"

	// 导入 buntdb 驱动以注册驱动
	_ "github.com/gophertool/tool/db/cache/buntdb"
)

var (
	imageCache _interface.Cache
	once       sync.Once
)

// getCache 延迟初始化缓存
func getCache() _interface.Cache {
	once.Do(func() {
		cfg := cacheconfig.Cache{
			Driver: cacheconfig.CacheDriverBuntdb,
			Path:   config.AppConfig.CachePath,
		}

		var err error
		imageCache, err = _interface.New(cfg)
		if err != nil {
			log.Errorf("初始化图片缓存失败: %v", err)
		}
	})
	return imageCache
}

// ComputeHash 计算图片的哈希值
func ComputeHash(imageData string) string {
	hash := sha256.Sum256([]byte(imageData))
	return hex.EncodeToString(hash[:])
}

// GetImageResult 从缓存获取图片识别结果
func GetImageResult(imageHash string) (string, bool) {
	cache := getCache()
	if cache == nil {
		return "", false
	}
	result, err := cache.Get(imageHash)
	if err != nil {
		return "", false
	}
	return result, true
}

// SetImageResult 保存图片识别结果到缓存
func SetImageResult(imageHash string, result string) {
	cache := getCache()
	if cache == nil {
		return
	}
	ttl := time.Duration(config.AppConfig.CacheTTLHours) * time.Hour
	err := cache.Set(imageHash, result, ttl)
	if err != nil {
		log.Warnf("保存图片缓存失败: %v", err)
	}
}
