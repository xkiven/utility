package storage

import (
	"clipboard/config"
	"clipboard/storage/driver"
	"fmt"
)

// NewStorage 根据配置创建存储实例
func NewStorage(cfg *config.StorageConfig) (Storage, error) {
	switch cfg.Type {
	case config.StorageTypeJSON:
		return driver.NewJSONStorage(cfg)
	case config.StorageTypeMySQL:
		return driver.NewMySQLStorage(cfg)
	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", cfg.Type)
	}
}
