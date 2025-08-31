package driver

import (
	"clipboard/config"
	"clipboard/model"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// JSONStorage JSON文件存储实现
type JSONStorage struct {
	config    *config.StorageConfig
	filePath  string
	imagePath string
}

// NewJSONStorage 创建JSON存储实例
func NewJSONStorage(cfg *config.StorageConfig) (*JSONStorage, error) {
	// 确定存储路径 - 优先使用用户自定义路径
	storagePath := cfg.JSONPath

	// 如果未启用自定义路径或路径为空，使用默认路径
	if !cfg.CustomPath || storagePath == "" {
		appDataDir, err := os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		storagePath = filepath.Join(appDataDir, "clipboard-manager", "history")
	}

	// 确保存储目录存在
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, err
	}

	// 图片存储目录 - 始终在选定的存储路径下
	imagePath := filepath.Join(storagePath, "images")
	if err := os.MkdirAll(imagePath, 0755); err != nil {
		return nil, err
	}

	return &JSONStorage{
		config:    cfg,
		filePath:  filepath.Join(storagePath, "history.json"),
		imagePath: imagePath,
	}, nil
}

// SaveItems 保存所有历史项
func (s *JSONStorage) SaveItems(items []*model.ClipboardItem) error {
	// 确保不超过最大数量
	if len(items) > s.config.MaxItems {
		items = items[:s.config.MaxItems]
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

// LoadItems 加载所有历史项
func (s *JSONStorage) LoadItems() ([]*model.ClipboardItem, error) {
	var items []*model.ClipboardItem

	// 检查文件是否存在
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return items, nil
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	// 排序：收藏项在前，按时间降序
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsFavorite && !items[j].IsFavorite {
			return true
		}
		if !items[i].IsFavorite && items[j].IsFavorite {
			return false
		}
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	return items, nil
}

// AddItem 添加新项
func (s *JSONStorage) AddItem(newItem *model.ClipboardItem) ([]*model.ClipboardItem, error) {
	items, err := s.LoadItems()
	if err != nil {
		return nil, err
	}

	// 检查重复
	for _, item := range items {
		if item.Content == newItem.Content &&
			item.Type == newItem.Type &&
			item.ImagePath == newItem.ImagePath {
			return items, nil
		}
	}

	// 添加到开头
	items = append([]*model.ClipboardItem{newItem}, items...)

	// 限制数量
	if len(items) > s.config.MaxItems {
		items = items[:s.config.MaxItems]
	}

	if err := s.SaveItems(items); err != nil {
		return nil, err
	}

	return items, nil
}

// DeleteItem 删除项
func (s *JSONStorage) DeleteItem(id string) ([]*model.ClipboardItem, error) {
	items, err := s.LoadItems()
	if err != nil {
		return nil, err
	}

	// 查找并删除
	for i, item := range items {
		if item.ID == id {
			// 如果是图片，删除文件
			if item.Type == model.TypeImage && item.ImagePath != "" {
				os.Remove(item.ImagePath)
			}

			items = append(items[:i], items[i+1:]...)
			break
		}
	}

	if err := s.SaveItems(items); err != nil {
		return nil, err
	}

	return items, nil
}

// ToggleFavorite 切换收藏状态
func (s *JSONStorage) ToggleFavorite(id string) ([]*model.ClipboardItem, error) {
	items, err := s.LoadItems()
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		if item.ID == id {
			item.IsFavorite = !item.IsFavorite
			break
		}
	}

	if err := s.SaveItems(items); err != nil {
		return nil, err
	}

	return items, nil
}

// Search 搜索项
func (s *JSONStorage) Search(keyword string) ([]*model.ClipboardItem, error) {
	items, err := s.LoadItems()
	if err != nil {
		return nil, err
	}

	if keyword == "" {
		return items, nil
	}

	var results []*model.ClipboardItem
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Content), strings.ToLower(keyword)) {
			results = append(results, item)
		}
	}

	return results, nil
}

// GetImagePath 获取图片存储路径
func (s *JSONStorage) GetImagePath() string {
	return s.imagePath
}

// Close 关闭存储
func (s *JSONStorage) Close() error {
	return nil
}
