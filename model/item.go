package model

import (
	"gorm.io/gorm"
	"time"
)

// ItemType 定义剪贴板内容类型
type ItemType int

const (
	TypeText ItemType = iota
	TypeImage
	TypeFile
)

// ClipboardItem 表示一个剪贴板历史项
type ClipboardItem struct {
	ID         string         `json:"id" gorm:"primaryKey"`
	Type       ItemType       `json:"type"`
	Content    string         `json:"content"`   // 文本内容或文件路径
	ImagePath  string         `json:"imagePath"` // 图片临时文件路径
	Timestamp  time.Time      `json:"timestamp"`
	IsFavorite bool           `json:"isFavorite"`
	CreatedAt  time.Time      `json:"-"`
	UpdatedAt  time.Time      `json:"-"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

// NewClipboardItem 创建新的剪贴板历史项
func NewClipboardItem(itemType ItemType, content, imagePath string) *ClipboardItem {
	return &ClipboardItem{
		ID:         generateID(),
		Type:       itemType,
		Content:    content,
		ImagePath:  imagePath,
		Timestamp:  time.Now(),
		IsFavorite: false,
	}
}

// 生成唯一ID
func generateID() string {
	return time.Now().Format("20060102150405000000")
}
