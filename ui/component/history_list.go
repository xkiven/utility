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
			list.onSelect(list.items[i])
		}
	}

	return list
}

// UpdateItems 更新列表项
func (l *HistoryList) UpdateItems(items []*model.ClipboardItem) {
	l.items = items
	l.Refresh()
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
	itemContainer := box.Objects[0].(*fyne.Container)
	mainContent := itemContainer.Objects[0].(*fyne.Container)
	buttons := itemContainer.Objects[1].(*fyne.Container)

	// 获取内容标签
	contentLabel := mainContent.Objects[0].(*widget.Label)
	timeLabel := mainContent.Objects[1].(*widget.Label)

	// 获取按钮
	favoriteBtn := buttons.Objects[0].(*widget.Button)
	deleteBtn := buttons.Objects[1].(*widget.Button)

	// 设置内容
	switch item.Type {
	case model.TypeText:
		content := item.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		contentLabel.SetText(content)
	case model.TypeImage:
		contentLabel.SetText("[图片内容]")
	case model.TypeFile:
		contentLabel.SetText("[文件] " + item.Content)
	}

	// 设置时间
	timeLabel.SetText(formatTime(item.Timestamp))

	// 设置收藏状态 - 使用通用图标替代
	if item.IsFavorite {
		// 已收藏状态使用确认图标
		favoriteBtn.SetIcon(theme.ConfirmIcon())
	} else {
		// 未收藏状态使用添加图标
		favoriteBtn.SetIcon(theme.ContentAddIcon())
	}

	// 设置按钮点击事件
	id := item.ID
	favoriteBtn.OnTapped = func() {
		if l.onFavorite != nil {
			l.onFavorite(id)
		}
	}

	deleteBtn.OnTapped = func() {
		if l.onDelete != nil {
			l.onDelete(id)
		}
	}

	// 收藏项高亮显示
	if item.IsFavorite {
		background := canvas.NewRectangle(color.RGBA{R: 255, G: 255, B: 200, A: 100})
		box.Objects = []fyne.CanvasObject{
			background,
			itemContainer,
			box.Objects[1], // 分隔线
		}
		background.Move(fyne.NewPos(0, 0))
		background.Resize(box.Size())
	}
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
