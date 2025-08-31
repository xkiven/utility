package driver

import (
	"clipboard/config"
	"clipboard/model"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
	"path/filepath"
)

// MySQLStorage MySQL存储实现（使用GORM）
type MySQLStorage struct {
	config    *config.StorageConfig
	db        *gorm.DB
	imagePath string
}

// NewMySQLStorage 创建MySQL存储实例
func NewMySQLStorage(cfg *config.StorageConfig) (*MySQLStorage, error) {
	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQL.User,
		cfg.MySQL.Password,
		cfg.MySQL.Host,
		cfg.MySQL.Port,
		cfg.MySQL.Database,
	)

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("无法连接到MySQL数据库: %v", err)
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(&model.ClipboardItem{}); err != nil {
		return nil, fmt.Errorf("迁移表结构失败: %v", err)
	}

	// 创建图片存储目录
	appDataDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	imagePath := filepath.Join(appDataDir, "clipboard-manager", "mysql_images")
	if err := os.MkdirAll(imagePath, 0755); err != nil {
		return nil, err
	}

	return &MySQLStorage{
		config:    cfg,
		db:        db,
		imagePath: imagePath,
	}, nil
}

// SaveItems 保存所有历史项
func (s *MySQLStorage) SaveItems(items []*model.ClipboardItem) error {
	// 先清空旧数据
	if err := s.db.Where("1 = 1").Delete(&model.ClipboardItem{}).Error; err != nil {
		return err
	}

	// 批量插入新数据
	return s.db.Create(items).Error
}

// LoadItems 加载所有历史项
func (s *MySQLStorage) LoadItems() ([]*model.ClipboardItem, error) {
	var items []*model.ClipboardItem

	// 查询并按收藏状态和时间排序
	result := s.db.Order("is_favorite DESC, timestamp DESC").
		Limit(s.config.MaxItems).
		Find(&items)

	if result.Error != nil {
		return nil, result.Error
	}

	return items, nil
}

// AddItem 添加新项
func (s *MySQLStorage) AddItem(newItem *model.ClipboardItem) ([]*model.ClipboardItem, error) {
	// 检查是否已存在相同内容
	var existingItem model.ClipboardItem
	result := s.db.Where("content = ? AND type = ? AND image_path = ?",
		newItem.Content, newItem.Type, newItem.ImagePath).
		First(&existingItem)

	if result.Error == nil {
		// 已存在，更新时间戳
		if err := s.db.Model(&existingItem).Update("timestamp", newItem.Timestamp).Error; err != nil {
			return nil, err
		}
	} else if result.Error == gorm.ErrRecordNotFound {
		// 不存在，插入新记录
		if err := s.db.Create(newItem).Error; err != nil {
			return nil, err
		}
	} else {
		// 其他错误
		return nil, result.Error
	}

	// 获取超过最大数量的记录ID
	var oldItems []model.ClipboardItem
	if err := s.db.Order("is_favorite DESC, timestamp ASC").
		Offset(s.config.MaxItems).
		Find(&oldItems).Error; err != nil {
		return nil, err
	}

	// 删除超过最大数量的记录
	if len(oldItems) > 0 {
		var ids []string
		for _, item := range oldItems {
			ids = append(ids, item.ID)
		}

		// 删除前先获取图片路径
		var imageItems []model.ClipboardItem
		if err := s.db.Where("id IN ? AND type = ?", ids, model.TypeImage).
			Find(&imageItems).Error; err != nil {
			return nil, err
		}

		// 删除图片文件
		for _, item := range imageItems {
			if item.ImagePath != "" {
				os.Remove(item.ImagePath)
			}
		}

		// 从数据库删除
		if err := s.db.Where("id IN ?", ids).Delete(&model.ClipboardItem{}).Error; err != nil {
			return nil, err
		}
	}

	// 返回更新后的列表
	return s.LoadItems()
}

// DeleteItem 删除项
func (s *MySQLStorage) DeleteItem(id string) ([]*model.ClipboardItem, error) {
	// 先获取项信息
	var item model.ClipboardItem
	if err := s.db.First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// 如果是图片，删除文件
	if item.Type == model.TypeImage && item.ImagePath != "" {
		os.Remove(item.ImagePath)
	}

	// 从数据库删除
	if err := s.db.Delete(&model.ClipboardItem{}, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// 返回更新后的列表
	return s.LoadItems()
}

// ToggleFavorite 切换收藏状态
func (s *MySQLStorage) ToggleFavorite(id string) ([]*model.ClipboardItem, error) {
	// 使用GORM的更新功能切换收藏状态
	result := s.db.Model(&model.ClipboardItem{}).
		Where("id = ?", id).
		Update("is_favorite", gorm.Expr("NOT is_favorite"))

	if result.Error != nil {
		return nil, result.Error
	}

	// 返回更新后的列表
	return s.LoadItems()
}

// Search 搜索项
func (s *MySQLStorage) Search(keyword string) ([]*model.ClipboardItem, error) {
	if keyword == "" {
		return s.LoadItems()
	}

	var items []*model.ClipboardItem
	result := s.db.Where("content LIKE ?", "%"+keyword+"%").
		Order("is_favorite DESC, timestamp DESC").
		Find(&items)

	if result.Error != nil {
		return nil, result.Error
	}

	return items, nil
}

// GetImagePath 获取图片存储路径
func (s *MySQLStorage) GetImagePath() string {
	return s.imagePath
}

// Close 关闭存储
func (s *MySQLStorage) Close() error {
	// 获取底层sql.DB并关闭
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
