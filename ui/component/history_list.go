package component

import (
	"clipboard/model"
	"fmt"
	"image/color"
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
	// 关键修改：创建全新的切片，避免引用旧数据
	newItems := make([]*model.ClipboardItem, 0, len(items))
	for _, item := range items {
		// 深度拷贝每个项，确保数据完全隔离
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
	l.items = newItems

	// 强制UI完全重建列表
	fyne.Do(func() {
		l.items = newItems
		l.Refresh()
		l.UnselectAll()
		l.Length = func() int { return len(l.items) }
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

	// 确定 itemContainer 和分隔线的位置
	var itemContainer *fyne.Container
	var separator fyne.CanvasObject

	// 检查第一个对象的类型
	if _, isRect := box.Objects[0].(*canvas.Rectangle); isRect {
		// 有背景矩形的情况
		itemContainer = box.Objects[1].(*fyne.Container)
		separator = box.Objects[2]
	} else {
		// 没有背景矩形的情况
		itemContainer = box.Objects[0].(*fyne.Container)
		separator = box.Objects[1]
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
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		contentText = content
	case model.TypeImage:
		contentText = "[图片内容]"
	case model.TypeFile:
		contentText = "[文件] " + item.Content
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

				// 执行收藏操作（不预先更新UI）
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
				box.Objects = []fyne.CanvasObject{
					background,
					itemContainer,
					separator,
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
				// 移除背景
				box.Objects = []fyne.CanvasObject{
					itemContainer,
					separator,
				}
			}
		}
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
