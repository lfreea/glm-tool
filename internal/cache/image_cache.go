package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/gophertool/tool/db/cache/config"
	_interface "github.com/gophertool/tool/db/cache/interface"
	"github.com/gophertool/tool/log"

	// 导入 buntdb 驱动以注册驱动
	_ "github.com/gophertool/tool/db/cache/buntdb"
)

// ImageCache 图片识别结果缓存（使用 gophertool/tool 实现）
var ImageCache _interface.Cache

func init() {
	// 初始化 BuntDB 缓存
	cfg := config.Cache{
		Driver: config.CacheDriverBuntdb,
		Path:   "image_cache.db",
	}

	var err error
	ImageCache, err = _interface.New(cfg)
	if err != nil {
		log.Errorf("初始化图片缓存失败: %v", err)
	}
}

// ComputeHash 计算图片的哈希值
func ComputeHash(imageData string) string {
	hash := sha256.Sum256([]byte(imageData))
	return hex.EncodeToString(hash[:])
}

// GetImageResult 从缓存获取图片识别结果
func GetImageResult(imageHash string) (string, bool) {
	result, err := ImageCache.Get(imageHash)
	if err != nil {
		// key not found 或其他错误
		return "", false
	}
	return result, true
}

// SetImageResult 保存图片识别结果到缓存（24小时过期）
func SetImageResult(imageHash string, result string) {
	err := ImageCache.Set(imageHash, result, 24*time.Hour)
	if err != nil {
		log.Warnf("保存图片缓存失败: %v", err)
	}
}
