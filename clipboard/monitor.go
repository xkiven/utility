package clipboard

import (
	"clipboard/model"
	"clipboard/storage"
	"errors"
	"fmt"
	"github.com/atotto/clipboard"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Monitor 剪贴板监听器
type Monitor struct {
	storage      storage.Storage             // 存储接口
	processor    *Processor                  // 内容处理器（图片等复杂内容）
	stopChan     chan struct{}               // 停止信号通道
	changeChan   chan []*model.ClipboardItem // 变化通知通道
	lastText     string                      // 上次文本内容
	lastImageID  string                      // 上次图片ID
	lastFileList string                      // 上次文件列表
	isRunning    bool                        // 运行状态标识
}

// NewMonitor 创建剪贴板监听器
func NewMonitor(s storage.Storage) (*Monitor, error) {
	processor, err := NewProcessor(s.GetImagePath())
	if err != nil {
		return nil, fmt.Errorf("初始化处理器失败: %w", err)
	}

	return &Monitor{
		storage:    s,
		processor:  processor,
		stopChan:   make(chan struct{}),
		changeChan: make(chan []*model.ClipboardItem, 10),
	}, nil
}

// Start 开始监听剪贴板变化
func (m *Monitor) Start() error {
	if m.isRunning {
		return errors.New("监控器已在运行中")
	}

	m.isRunning = true
	go func() {
		defer func() {
			m.isRunning = false
		}()

		for {
			select {
			case <-m.stopChan:
				return
			default:
				m.checkClipboard()
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()

	return nil
}

// Stop 停止监听剪贴板
func (m *Monitor) Stop() {
	if !m.isRunning {
		return
	}

	close(m.stopChan)
	time.Sleep(100 * time.Millisecond)
}

// IsRunning 检查监控器是否在运行
func (m *Monitor) IsRunning() bool {
	return m.isRunning
}

// ChangeChan 获取变化通知通道
func (m *Monitor) ChangeChan() <-chan []*model.ClipboardItem {
	return m.changeChan
}

// SetContent 设置剪贴板内容
func (m *Monitor) SetContent(item *model.ClipboardItem) error {
	if item == nil {
		return errors.New("无效的剪贴板项")
	}

	switch item.Type {
	case model.TypeText:
		return clipboard.WriteAll(item.Content)
	case model.TypeImage:
		if item.ImagePath == "" {
			return errors.New("图片路径为空")
		}
		return m.processor.SetImageToClipboard(item.ImagePath)
	case model.TypeFile:
		// 对于文件，直接写入路径字符串
		return clipboard.WriteAll(item.Content)
	default:
		return errors.New("不支持的内容类型")
	}
}

// checkClipboard 检查剪贴板变化
func (m *Monitor) checkClipboard() {
	// 先获取剪贴板文本内容
	text, err := clipboard.ReadAll()
	if err != nil {
		return
	}

	// 检查是否为文件路径
	isFile, fileList := m.checkFilePaths(text)
	if isFile && fileList != m.lastFileList {
		m.handleFileChange(fileList)
		return
	}

	// 检查图片
	isImage, imageID, err := m.processor.CheckImage()
	if err == nil && isImage && imageID != m.lastImageID {
		m.handleImageChange(imageID)
		return
	}

	// 检查文本（排除文件情况）
	if text != "" && text != m.lastText && !isFile {
		m.handleTextChange(text)
		return
	}
}

// checkFilePaths 检查文本内容是否包含有效的文件路径
func (m *Monitor) checkFilePaths(text string) (bool, string) {
	if text == "" {
		return false, ""
	}

	// 可能的路径分隔符
	separators := []string{"\r\n", "\n", ";", "\t"}
	var paths []string

	// 尝试用不同分隔符拆分文本
	for _, sep := range separators {
		parts := strings.Split(text, sep)
		if len(parts) > 1 {
			paths = parts
			break
		}
	}

	// 如果没有拆分出多个部分，尝试将整个文本作为单个路径
	if len(paths) == 0 {
		paths = []string{text}
	}

	// 验证路径是否有效
	var validPaths []string
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		// 检查路径是否存在
		if isFileOrDirExists(path) {
			validPaths = append(validPaths, path)
		}
	}

	// 如果有有效的文件路径，视为文件剪贴板内容
	if len(validPaths) > 0 {
		return true, strings.Join(validPaths, ";")
	}

	return false, ""
}

// handleTextChange 处理文本内容变化
func (m *Monitor) handleTextChange(text string) {
	m.lastText = text
	item := model.NewClipboardItem(model.TypeText, text, "")
	items, err := m.storage.AddItem(item)
	if err != nil {
		fmt.Printf("保存文本失败: %v\n", err)
		return
	}

	select {
	case m.changeChan <- items:
	default:
		fmt.Println("通知通道已满，丢弃文本更新")
	}
}

// handleImageChange 处理图片内容变化
func (m *Monitor) handleImageChange(imageID string) {
	m.lastImageID = imageID
	imagePath, err := m.processor.SaveImage()
	if err != nil {
		fmt.Printf("保存图片失败: %v\n", err)
		return
	}

	item := model.NewClipboardItem(model.TypeImage, "图片内容", imagePath)
	items, err := m.storage.AddItem(item)
	if err != nil {
		fmt.Printf("保存图片记录失败: %v\n", err)
		return
	}

	select {
	case m.changeChan <- items:
	default:
		fmt.Println("通知通道已满，丢弃图片更新")
	}
}

// handleFileChange 处理文件内容变化
func (m *Monitor) handleFileChange(fileList string) {
	m.lastFileList = fileList
	item := model.NewClipboardItem(model.TypeFile, fileList, "")
	items, err := m.storage.AddItem(item)
	if err != nil {
		fmt.Printf("保存文件记录失败: %v\n", err)
		return
	}

	select {
	case m.changeChan <- items:
	default:
		fmt.Println("通知通道已满，丢弃文件更新")
	}
}

// isFileOrDirExists 检查文件或目录是否存在
func isFileOrDirExists(path string) bool {
	if path == "" {
		return false
	}

	// 尝试获取绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// 检查路径是否存在
	_, err = os.Stat(absPath)
	return !os.IsNotExist(err)
}
