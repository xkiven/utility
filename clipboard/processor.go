package clipboard

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/skratchdot/open-golang/open"
	"golang.design/x/clipboard"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
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
	imageID := p.imageID(imgCfg.Width, imgCfg.Height, len(data))
	return true, imageID, nil
}

// SaveImage 保存剪贴板中的图片到文件
func (p *Processor) SaveImage() (string, error) {
	// 读取剪贴板图片数据
	data := clipboard.Read(clipboard.FmtImage)
	if len(data) == 0 {
		return "", ErrNoImageData
	}

	// 解码图片获取格式信息
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("图片解码失败: %w", err)
	}

	// 生成唯一文件名
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("clip_%s.%s", timestamp, format)
	filePath := filepath.Join(p.imagePath, filename)

	// 创建文件并写入图片数据
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 根据图片格式编码并保存
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

	return filePath, nil
}

// SetImageToClipboard 将图片文件设置到剪贴板
func (p *Processor) SetImageToClipboard(imagePath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return ErrFileNotFound
	}

	// 读取图片文件
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return fmt.Errorf("读取图片失败: %w", err)
	}

	// 将图片数据写入剪贴板
	clipboard.Write(clipboard.FmtImage, data)
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
func (p *Processor) imageID(width, height, size int) string {
	return fmt.Sprintf("%s_%dx%d_%d",
		time.Now().Format("20060102150405"),
		width, height, size)
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
