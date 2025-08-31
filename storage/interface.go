package storage

import "clipboard/model"

// Storage 存储接口定义
type Storage interface {
	// SaveItems 保存所有历史项
	SaveItems(items []*model.ClipboardItem) error

	// LoadItems 加载所有历史项
	LoadItems() ([]*model.ClipboardItem, error)

	// AddItem 添加新项
	AddItem(item *model.ClipboardItem) ([]*model.ClipboardItem, error)

	// DeleteItem 删除项
	DeleteItem(id string) ([]*model.ClipboardItem, error)

	// ToggleFavorite 切换收藏状态
	ToggleFavorite(id string) ([]*model.ClipboardItem, error)

	// Search 搜索项
	Search(keyword string) ([]*model.ClipboardItem, error)

	// GetImagePath 获取图片存储路径
	GetImagePath() string

	// 关闭存储
	Close() error
}
