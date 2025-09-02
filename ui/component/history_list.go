package component

import (
	"clipboard/model"
	"fmt"
	"image/color"
	"log"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// HistoryList 历史记录列表组件
type HistoryList struct {
	*widget.List
	items      []*model.ClipboardItem     // 历史项列表
	onSelect   func(*model.ClipboardItem) // 选择回调
	onFavorite func(string)               // 收藏回调
	onDelete   func(string)               // 删除回调
}

// NewHistoryList 创建历史记录列表
func NewHistoryList(
	items []*model.ClipboardItem,
	onSelect func(*model.ClipboardItem),
	onFavorite func(string),
	onDelete func(string),
) *HistoryList {
	list := &HistoryList{
		items:      items,
		onSelect:   onSelect,
		onFavorite: onFavorite,
		onDelete:   onDelete,
	}

	list.List = widget.NewList(
		func() int {
			return len(list.items)
		},
		func() fyne.CanvasObject {
			return list.createItemWidget()
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			list.updateItemWidget(i, o)
		},
	)

	list.OnSelected = func(i widget.ListItemID) {
		if i >= 0 && i < len(list.items) && list.onSelect != nil {
			// 直接使用项目ID对应的实体，而非依赖索引
			selectedItem := list.items[i]
			list.onSelect(selectedItem)

			// 取消选中状态
			fyne.CurrentApp().Driver().CanvasForObject(list).Focus(nil)
			list.Unselect(i)
		}
	}

	return list
}

// UpdateItems 更新列表项
func (l *HistoryList) UpdateItems(items []*model.ClipboardItem) {
	log.Printf("更新列表项，接收数据量: %d", len(items))
	// 深拷贝新数据（双重保险，避免外部引用影响）
	newItems := make([]*model.ClipboardItem, 0, len(items))
	for _, item := range items {
		newItem := &model.ClipboardItem{
			ID:         item.ID,
			Type:       item.Type,
			Content:    item.Content,
			ImagePath:  item.ImagePath,
			IsFavorite: item.IsFavorite,
			Timestamp:  item.Timestamp,
		}
		newItems = append(newItems, newItem)
	}

	// 切换到主线程刷新UI（修改后：先清空再加载）
	fyne.Do(func() {
		// 步骤1：先清空旧数据和UI
		l.items = []*model.ClipboardItem{}
		l.Length = func() int { return 0 }
		l.Refresh() // 强制销毁旧UI项

		// 步骤2：再加载新数据
		l.items = newItems
		l.Length = func() int { return len(l.items) }
		l.Refresh() // 渲染新UI项

		l.UnselectAll()
		log.Printf("列表UI已更新，显示数量: %d", len(l.items))
	})
}

// 创建列表项控件
func (l *HistoryList) createItemWidget() fyne.CanvasObject {
	content := widget.NewLabel("")
	content.Wrapping = fyne.TextWrapWord

	timestamp := widget.NewLabel("")
	timestamp.TextStyle = fyne.TextStyle{Italic: true}

	// 使用通用图标替代星星图标（兼容所有版本）
	favoriteBtn := widget.NewButtonWithIcon("", theme.ConfirmIcon(), func() {})
	deleteBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {})

	// 按钮样式
	favoriteBtn.Importance = widget.LowImportance
	deleteBtn.Importance = widget.LowImportance

	// 主内容区域
	mainContent := container.NewVBox(
		content,
		timestamp,
	)

	// 按钮区域
	buttons := container.NewHBox(
		favoriteBtn,
		deleteBtn,
	)

	// 整个项的布局
	item := container.NewBorder(
		nil, nil, nil, buttons,
		mainContent,
	)

	// 添加分隔线
	return container.NewVBox(
		item,
		canvas.NewLine(color.Gray{Y: 200}),
	)
}

// 更新列表项控件
func (l *HistoryList) updateItemWidget(i int, o fyne.CanvasObject) {
	if i < 0 || i >= len(l.items) {
		return
	}

	item := l.items[i]
	box := o.(*fyne.Container)
	var itemContainer *fyne.Container

	// 正确逻辑：遍历找到内容容器（忽略背景矩形和分隔线）
	for _, obj := range box.Objects {
		// 内容容器是 VBox（包含 mainContent 和 buttons），排除 canvas 类型（背景/分隔线）
		if container, ok := obj.(*fyne.Container); ok && container.Layout != nil {
			// 进一步验证：内容容器的子项包含 mainContent（VBox）和 buttons（HBox）
			if len(container.Objects) == 2 {
				itemContainer = container
				break
			}
		}
	}

	// 防御：若未找到内容容器，直接返回（避免 panic）
	if itemContainer == nil {
		log.Printf("警告：未找到索引 %d 的内容容器，跳过更新", i)
		return
	}

	mainContent := itemContainer.Objects[0].(*fyne.Container)
	buttons := itemContainer.Objects[1].(*fyne.Container)

	// 获取内容标签
	contentLabel := mainContent.Objects[0].(*widget.Label)
	timeLabel := mainContent.Objects[1].(*widget.Label)

	// 获取按钮
	favoriteBtn := buttons.Objects[0].(*widget.Button)
	deleteBtn := buttons.Objects[1].(*widget.Button)

	// 准备内容文本
	var contentText string
	switch item.Type {
	case model.TypeText:
		content := item.Content
		if len(content) > 15 {
			content = content[:15] + "..."
		}
		contentText = content
	case model.TypeImage:
		contentText = "[图片内容] " + filepath.Base(item.ImagePath) // 显示图片文件名
	case model.TypeFile:
		content := item.Content
		if len(content) > 15 {
			content = content[:15] + "..."
		}
		contentText = "[文件] " + content
	}

	// 准备时间文本
	timeText := formatTime(item.Timestamp)

	// 使用 fyne.Do 确保在主线程执行 UI 操作
	fyne.Do(func() {
		// 设置内容
		contentLabel.SetText(contentText)

		// 设置时间
		timeLabel.SetText(timeText)

		// 设置收藏状态
		if item.IsFavorite {
			favoriteBtn.SetIcon(theme.ConfirmIcon())
		} else {
			favoriteBtn.SetIcon(theme.ContentAddIcon())
		}

		// 设置按钮点击事件
		id := item.ID
		favoriteBtn.OnTapped = func() {
			if l.onFavorite != nil {
				itemID := id // 保存当前ID的副本
				l.onFavorite(itemID)
			}
		}

		deleteBtn.OnTapped = func() {
			if l.onDelete != nil {
				// 使用当前item的ID副本
				itemID := id
				l.onDelete(itemID)
			}
		}

		// 收藏项高亮显示
		if item.IsFavorite {
			// 检查是否已经有背景
			var background *canvas.Rectangle
			var hasBackground bool

			// 检查第一个对象是否是背景矩形
			if rect, isRect := box.Objects[0].(*canvas.Rectangle); isRect {
				background = rect
				hasBackground = true
			}

			if !hasBackground {
				// 创建新背景
				background = canvas.NewRectangle(color.RGBA{R: 255, G: 255, B: 200, A: 100})
				// 保留分隔线但不使用变量引用
				box.Objects = []fyne.CanvasObject{
					background,
					itemContainer,
					box.Objects[1], // 分隔线
				}
			}

			// 更新背景位置和大小
			background.Move(fyne.NewPos(0, 0))
			background.Resize(box.Size())
			background.FillColor = color.RGBA{R: 255, G: 255, B: 200, A: 100}
			background.Refresh()
		} else {
			// 如果不是收藏项，确保没有背景
			if _, isRect := box.Objects[0].(*canvas.Rectangle); isRect {
				// 移除背景，保留分隔线
				box.Objects = []fyne.CanvasObject{
					itemContainer,
					box.Objects[2], // 分隔线
				}
			}
		}

		// 强制刷新控件
		contentLabel.Refresh()
		timeLabel.Refresh()
		favoriteBtn.Refresh()
		deleteBtn.Refresh()
		itemContainer.Refresh()
		box.Refresh()
	})
}

// 格式化时间显示
func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return fmt.Sprintf("%d秒前", int(diff.Seconds()))
	} else if diff < time.Hour {
		return fmt.Sprintf("%d分钟前", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%d小时前", int(diff.Hours()))
	} else if diff < 7*24*time.Hour {
		return fmt.Sprintf("%d天前", int(diff.Hours()/24))
	}

	return t.Format("2006-01-02 15:04")
}
