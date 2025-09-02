package clipboard

import (
	"clipboard/model"
	"clipboard/storage"
	"errors"
	"fmt"
	"golang.design/x/clipboard"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Monitor 剪贴板监听器
type Monitor struct {
	storage      storage.Storage             // 存储接口
	processor    *Processor                  // 内容处理器（图片等复杂内容）
	StopChan     chan struct{}               // 停止信号通道
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
		StopChan:   make(chan struct{}),
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
			case <-m.StopChan:
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

	close(m.StopChan)
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
		clipboard.Write(clipboard.FmtText, []byte(item.Content))
		return nil
	case model.TypeImage:
		if item.ImagePath == "" {
			return errors.New("图片路径为空")
		}
		// 新增日志：确认传递的图片路径
		log.Printf("准备复制图片，路径：%s", item.ImagePath)
		return m.processor.SetImageToClipboard(item.ImagePath) // 确保路径是绝对路径
	case model.TypeFile:
		clipboard.Write(clipboard.FmtText, []byte(item.Content))
		return nil
	default:
		return errors.New("不支持的内容类型")
	}
}

// checkClipboard 检查剪贴板变化
func (m *Monitor) checkClipboard() {
	// 1. 优先检查图片（避免文本读取干扰，调整顺序）
	isImage, imageID, err := m.processor.CheckImage()
	if err == nil && isImage && imageID != m.lastImageID {
		log.Printf("检测到图片变化，ID: %s", imageID)
		m.handleImageChange(imageID)
		return
	}

	// 2. 用统一库读取文本（替换 atotto/clipboard）
	textData := clipboard.Read(clipboard.FmtText) // 读取文本格式数据
	text := string(textData)                      // 转为字符串
	if len(textData) == 0 {
		// 无文本数据，直接返回（避免空文本判断）
		return
	}

	// 3. 后续文件检查、文本处理逻辑保持不变
	isFile, fileList := m.checkFilePaths(text)
	if isFile && fileList != m.lastFileList {
		log.Printf("检测到文件变化: %s", fileList)
		m.handleFileChange(fileList)
		return
	}

	if text != "" && text != m.lastText && !isFile {
		log.Printf("检测到文本变化: %s", text)
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
	item := model.NewClipboardItem(model.TypeText, text, "")
	items, err := m.storage.AddItem(item)
	if err != nil {
		fmt.Printf("保存文本失败: %v\n", err)
		return
	}

	// 同步存储层最新值（保持原逻辑）
	if len(items) > 0 {
		for _, i := range items {
			if i.Type == model.TypeText && i.Content == text {
				m.lastText = i.Content
				break
			}
		}
	}

	// 通知应用层触发全量重建（忽略通道满的情况，确保重建优先级）
	select {
	case m.changeChan <- items:
		log.Println("监控层：文本变化通知已发送（触发重建）")
	default:
		log.Println("监控层：通知通道已满，强制重试发送（确保重建）")
		// 通道满时阻塞发送，避免丢失更新（核心修改）
		m.changeChan <- items
	}
}

// handleImageChange 处理图片内容变化
func (m *Monitor) handleImageChange(imageID string) {
	// 1. 先判断是否为已处理的图片（避免重复触发）
	if imageID == m.lastImageID {
		log.Printf("忽略已处理的图片ID: %s", imageID)
		return
	}

	// 2. 标记为“处理中”，避免并发重复处理
	m.lastImageID = imageID

	// 3. 保存图片到本地
	imagePath, err := m.processor.SaveImage()
	if err != nil {
		fmt.Printf("保存图片失败: %v\n", err)
		// 保存失败时重置lastImageID，允许下次重试
		m.lastImageID = ""
		return
	}

	// 4. 保存图片记录到存储
	item := model.NewClipboardItem(model.TypeImage, "图片内容", imagePath)
	items, err := m.storage.AddItem(item)
	if err != nil {
		fmt.Printf("保存图片记录失败: %v\n", err)
		return
	}

	// 5. 关键修复：保存成功后，清空剪贴板图片数据（避免下次重复检测）
	// 注意：仅清空图片格式，不影响文本/文件内容
	clipboard.Write(clipboard.FmtImage, []byte{})

	// 6. 发送变化通知
	select {
	case m.changeChan <- items:
		log.Printf("图片 %s 处理完成，已清空剪贴板图片数据", imageID)
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
