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

// NewHistoryList 创建历史记录列表（保持原初始化逻辑）
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
			selectedItem := list.items[i]
			list.onSelect(selectedItem) // 先触发复制逻辑
			// 延迟100ms清除焦点（核心修改：避免打断剪贴板写入）
			time.AfterFunc(100*time.Millisecond, func() {
				fyne.Do(func() {
					list.Unselect(i) // 取消选中状态
					// 清除画布焦点
					if canvas := fyne.CurrentApp().Driver().CanvasForObject(list); canvas != nil {
						canvas.Focus(nil)
					}
					list.RefreshItem(i) // 刷新列表项
				})
			})
		}
	}

	return list
}

// UpdateItems 禁用增量更新，强制通过重建实现刷新
func (l *HistoryList) UpdateItems(items []*model.ClipboardItem) {
	l.items = items
	l.Refresh()
}

// 创建列表项控件（保持原逻辑）
func (l *HistoryList) createItemWidget() fyne.CanvasObject {
	content := widget.NewLabel("")
	content.Wrapping = fyne.TextWrapWord

	timestamp := widget.NewLabel("")
	timestamp.TextStyle = fyne.TextStyle{Italic: true}

	favoriteBtn := widget.NewButtonWithIcon("", theme.ConfirmIcon(), func() {})
	deleteBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {})

	favoriteBtn.Importance = widget.LowImportance
	deleteBtn.Importance = widget.LowImportance

	mainContent := container.NewVBox(content, timestamp)
	buttons := container.NewHBox(favoriteBtn, deleteBtn)
	item := container.NewBorder(nil, nil, nil, buttons, mainContent)

	return container.NewVBox(item, canvas.NewLine(color.Gray{Y: 200}))
}

// 更新列表项控件（保持原逻辑）
func (l *HistoryList) updateItemWidget(i int, o fyne.CanvasObject) {
	if i < 0 || i >= len(l.items) {
		return
	}

	item := l.items[i]
	box := o.(*fyne.Container)
	var itemContainer *fyne.Container

	// 找到内容容器
	for _, obj := range box.Objects {
		if container, ok := obj.(*fyne.Container); ok && container.Layout != nil && len(container.Objects) == 2 {
			itemContainer = container
			break
		}
	}

	if itemContainer == nil {
		log.Printf("警告：未找到索引 %d 的内容容器，跳过更新", i)
		return
	}

	mainContent := itemContainer.Objects[0].(*fyne.Container)
	buttons := itemContainer.Objects[1].(*fyne.Container)

	contentLabel := mainContent.Objects[0].(*widget.Label)
	timeLabel := mainContent.Objects[1].(*widget.Label)
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
		contentText = "[图片内容] " + filepath.Base(item.ImagePath)
	case model.TypeFile:
		content := item.Content
		if len(content) > 15 {
			content = content[:15] + "..."
		}
		contentText = "[文件] " + content
	}

	// 准备时间文本
	timeText := formatTime(item.Timestamp)

	// 主线程更新UI
	fyne.Do(func() {
		contentLabel.SetText(contentText)
		timeLabel.SetText(timeText)

		// 设置收藏状态图标
		if item.IsFavorite {
			favoriteBtn.SetIcon(theme.ConfirmIcon())
		} else {
			favoriteBtn.SetIcon(theme.ContentAddIcon())
		}

		// 绑定按钮事件
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

		// 收藏项高亮
		if item.IsFavorite {
			var background *canvas.Rectangle
			var hasBackground bool

			if rect, isRect := box.Objects[0].(*canvas.Rectangle); isRect {
				background = rect
				hasBackground = true
			}

			if !hasBackground {
				background = canvas.NewRectangle(color.RGBA{R: 255, G: 255, B: 200, A: 100})
				box.Objects = []fyne.CanvasObject{background, itemContainer, box.Objects[1]}
			}

			background.Move(fyne.NewPos(0, 0))
			background.Resize(box.Size())
			background.FillColor = color.RGBA{R: 255, G: 255, B: 200, A: 100}
			background.Refresh()
		} else {
			if _, isRect := box.Objects[0].(*canvas.Rectangle); isRect {
				box.Objects = []fyne.CanvasObject{itemContainer, box.Objects[2]}
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

// 格式化时间显示（保持原逻辑）
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
