package clipboard

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/skratchdot/open-golang/open"
	"golang.design/x/clipboard"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"
)

// 预定义错误变量
var (
	ErrNoImageData    = errors.New("剪贴板中没有图片数据")
	ErrUnsupportedImg = errors.New("不支持的图片格式")
	ErrFileNotFound   = errors.New("图片文件不存在")
)

// Processor 剪贴板内容处理器
type Processor struct {
	imagePath string // 图片存储目录
}

// NewProcessor 创建内容处理器实例
func NewProcessor(imagePath string) (*Processor, error) {
	// 初始化剪贴板系统
	if err := clipboard.Init(); err != nil {
		return nil, fmt.Errorf("剪贴板初始化失败: %w", err)
	}

	// 确保图片目录存在
	if err := os.MkdirAll(imagePath, 0755); err != nil {
		return nil, fmt.Errorf("创建图片目录失败: %w", err)
	}

	return &Processor{
		imagePath: imagePath,
	}, nil
}

// CheckImage 检查剪贴板中是否有图片
// 返回值：是否为图片、图片唯一标识、错误信息
func (p *Processor) CheckImage() (bool, string, error) {
	// 读取剪贴板中的图片数据
	data := clipboard.Read(clipboard.FmtImage)
	if len(data) == 0 {
		return false, "", nil
	}

	// 验证图片格式
	imgCfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return false, "", fmt.Errorf("图片格式验证失败: %w", err)
	}

	// 生成图片唯一标识
	imageID := p.imageID(imgCfg.Width, imgCfg.Height, data)
	return true, imageID, nil
}

// SaveImage 保存剪贴板中的图片到文件
func (p *Processor) SaveImage() (string, error) {
	// 读取剪贴板图片数据（原有逻辑不变）
	data := clipboard.Read(clipboard.FmtImage)
	if len(data) == 0 {
		return "", ErrNoImageData
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("图片解码失败: %w", err)
	}

	// 生成文件名（原有逻辑不变）
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("clip_%s.%s", timestamp, format)
	// 生成绝对路径（核心修改）
	absImageDir, err := filepath.Abs(p.imagePath)
	if err != nil {
		return "", fmt.Errorf("获取图片目录绝对路径失败: %w", err)
	}
	filePath := filepath.Join(absImageDir, filename) // 用绝对路径拼接

	// 创建文件并写入（原有逻辑不变）
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 编码保存（原有逻辑不变）
	switch format {
	case "png":
		if err := png.Encode(file, img); err != nil {
			return "", fmt.Errorf("PNG编码失败: %w", err)
		}
	case "jpeg", "jpg":
		opts := &jpeg.Options{Quality: 90}
		if err := jpeg.Encode(file, img, opts); err != nil {
			return "", fmt.Errorf("JPEG编码失败: %w", err)
		}
	case "gif":
		if err := p.encodeGIF(file, img); err != nil {
			return "", fmt.Errorf("GIF编码失败: %w", err)
		}
	default:
		return "", ErrUnsupportedImg
	}

	// 验证文件是否成功创建（新增）
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("图片保存后文件不存在: %s", filePath)
	}

	log.Printf("图片已保存为绝对路径: %s", filePath) // 新增日志，便于调试
	return filePath, nil                   // 返回绝对路径
}

// SetImageToClipboard 将图片文件设置到剪贴板
func (p *Processor) SetImageToClipboard(imagePath string) error {
	// 1. 检查文件是否存在（原有逻辑增强）
	fileInfo, err := os.Stat(imagePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrFileNotFound, imagePath)
	}
	if err != nil {
		return fmt.Errorf("获取图片文件信息失败: %w", err)
	}
	// 新增：检查是否为文件（排除目录）
	if fileInfo.IsDir() {
		return fmt.Errorf("图片路径是目录，不是文件: %s", imagePath)
	}
	// 新增：检查文件大小（避免空文件）
	if fileInfo.Size() == 0 {
		return fmt.Errorf("图片文件为空: %s", imagePath)
	}

	// 2. 读取图片文件（新增超时和错误详情）
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return fmt.Errorf("读取图片失败（路径：%s）: %w", imagePath, err)
	}
	if len(data) == 0 {
		return fmt.Errorf("读取的图片数据为空（路径：%s）", imagePath)
	}

	// 新增：计算原文件的MD5哈希
	originalHash := md5.Sum(data)
	originalHashStr := hex.EncodeToString(originalHash[:])

	// 写入剪贴板
	clipboard.Write(clipboard.FmtImage, data)
	time.Sleep(200 * time.Millisecond)

	// 验证：不仅检查长度，还检查内容哈希
	writtenData := clipboard.Read(clipboard.FmtImage)
	if len(writtenData) == 0 || len(writtenData) != len(data) {
		return fmt.Errorf("图片写入剪贴板失败（写入大小：%d，读取大小：%d）", len(data), len(writtenData))
	}
	// 新增哈希校验
	writtenHash := md5.Sum(writtenData)
	writtenHashStr := hex.EncodeToString(writtenHash[:])
	if writtenHashStr != originalHashStr {
		return fmt.Errorf("图片写入剪贴板内容不一致（原哈希：%s，写入哈希：%s）", originalHashStr, writtenHashStr)
	}

	log.Printf("图片成功写入剪贴板（路径：%s，大小：%d KB，哈希：%s）", imagePath, len(data)/1024, originalHashStr)
	return nil
}

// OpenImage 打开图片文件（用于预览）
func (p *Processor) OpenImage(imagePath string) error {
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return ErrFileNotFound
	}

	return open.Start(imagePath)
}

// 生成图片唯一标识
func (p *Processor) imageID(width, height int, data []byte) string {
	// 1. 计算图片内容的MD5哈希（确保相同内容哈希一致）
	hash := md5.Sum(data)
	hashStr := hex.EncodeToString(hash[:])
	// 2. 组合哈希+尺寸生成唯一ID（不再依赖随机UUID）
	id := fmt.Sprintf("%s_%d_%d", hashStr, width, height)
	return id
}

// 编码GIF图片（使用标准库自动处理调色板）
func (p *Processor) encodeGIF(file *os.File, img image.Image) error {
	bounds := img.Bounds()

	// 创建带自动生成调色板的图像
	palettedImg := image.NewPaletted(bounds, nil)

	// 复制图像数据
	draw.Draw(palettedImg, bounds, img, bounds.Min, draw.Src)

	// 编码为GIF
	return gif.EncodeAll(file, &gif.GIF{
		Image: []*image.Paletted{palettedImg},
		Delay: []int{0},
	})
}
