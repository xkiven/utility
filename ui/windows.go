package ui

import (
	"clipboard/config"
	"clipboard/model"
	"clipboard/storage"
	"clipboard/ui/component"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"log"
	"sort"
)

// Window 应用主窗口
type Window struct {
	fyne.Window
	app            fyne.App
	storage        storage.Storage
	historyList    *component.HistoryList
	searchBar      *component.SearchBar
	settingsPanel  *component.SettingsPanel
	contentTabs    *container.AppTabs
	onSaveSettings func(*config.StorageConfig)
	clipboard      ClipboardSetter        // 用于设置剪贴板内容的接口
	favoriteList   *component.HistoryList // 新增收藏列表字段
}

// ClipboardSetter 剪贴板设置接口
type ClipboardSetter interface {
	SetContent(item *model.ClipboardItem) error
}

// NewWindow 创建主窗口（初始化逻辑不变）
func NewWindow(
	app fyne.App,
	storage storage.Storage,
	clipboard ClipboardSetter,
	onSaveSettings func(*config.StorageConfig),
) *Window {
	win := app.NewWindow("剪贴板历史管理器")
	win.Resize(fyne.NewSize(600, 400))

	w := &Window{
		Window:         win,
		app:            app,
		storage:        storage,
		clipboard:      clipboard,
		onSaveSettings: onSaveSettings,
	}

	// 初始化UI（首次创建）
	w.rebuildFullUI()

	return w
}

// 核心修改：全量重建UI（销毁旧实例+重建新实例）
func (w *Window) rebuildFullUI() {
	// 1. 销毁旧UI引用（解除内存关联）
	w.historyList = nil
	w.favoriteList = nil
	w.searchBar = nil
	w.settingsPanel = nil
	w.contentTabs = nil

	// 2. 重新加载最新数据（模拟重启时的数据初始化）
	items, err := w.storage.LoadItems()
	if err != nil {
		log.Printf("重建UI加载数据失败: %v", err)
		items = []*model.ClipboardItem{}
	}

	// 3. 分离收藏项和普通项（重新计算）
	favoriteItems := []*model.ClipboardItem{}
	normalItems := []*model.ClipboardItem{}
	for _, item := range items {
		if item.IsFavorite {
			favoriteItems = append(favoriteItems, item)
		} else {
			normalItems = append(normalItems, item)
		}
	}

	// 4. 重建搜索框（新实例）
	currentSearch := ""
	if w.searchBar != nil {
		currentSearch = w.searchBar.Text
	}
	w.searchBar = component.NewSearchBar(func(text string) {
		w.UpdateHistory(nil)
	})
	w.searchBar.SetText(currentSearch)

	// 新增：主动将焦点设置到搜索框
	fyne.Do(func() {
		if canvas := fyne.CurrentApp().Driver().CanvasForObject(w.searchBar); canvas != nil {
			canvas.Focus(w.searchBar) // 强制让搜索框获取焦点
		}
	})
	// 5. 重建普通历史列表（新实例+重新绑定回调）
	w.historyList = component.NewHistoryList(
		normalItems,
		func(item *model.ClipboardItem) {
			w.clipboard.SetContent(item)
		},
		func(id string) {
			// 收藏变更后触发全量重建
			_, err := w.storage.ToggleFavorite(id)
			if err == nil {
				w.rebuildFullUI()
			}
		},
		func(id string) {
			// 删除后触发全量重建
			_, err := w.storage.DeleteItem(id)
			if err == nil {
				w.rebuildFullUI()
			} else {
				log.Printf("删除失败: %v", err)
			}
		},
	)

	// 6. 重建收藏列表（新实例+重新绑定回调）
	w.favoriteList = component.NewHistoryList(
		favoriteItems,
		func(item *model.ClipboardItem) {
			w.clipboard.SetContent(item)
		},
		func(id string) {
			// 收藏变更后触发全量重建
			_, err := w.storage.ToggleFavorite(id)
			if err == nil {
				w.rebuildFullUI()
			}
		},
		func(id string) {
			// 删除后触发全量重建
			_, err := w.storage.DeleteItem(id)
			if err == nil {
				w.rebuildFullUI()
			}
		},
	)

	// 7. 重建主内容区域（新容器）
	historyContent := container.NewBorder(
		w.searchBar,
		nil, nil, nil,
		w.historyList,
	)

	// 8. 重建收藏内容区域（新容器）
	favoriteContent := container.NewBorder(
		nil, nil, nil, nil,
		w.favoriteList,
	)

	// 9. 重建设置面板（新实例+重新加载配置）
	cfg, _ := config.Load()
	w.settingsPanel = component.NewSettingsPanel(w.Window, &cfg.Storage, func(newCfg *config.StorageConfig) {
		// 设置保存后触发全量重建
		w.onSaveSettings(newCfg)
		w.rebuildFullUI()
	})

	// 10. 重建标签页（新容器）
	w.contentTabs = container.NewAppTabs(
		container.NewTabItemWithIcon("历史记录", theme.HistoryIcon(), historyContent),
		container.NewTabItemWithIcon("我的收藏", theme.ConfirmIcon(), favoriteContent),
		container.NewTabItemWithIcon("设置", theme.SettingsIcon(), w.settingsPanel),
	)

	// 11. 重新设置主内容（销毁旧UI树）
	w.SetContent(w.contentTabs)
	log.Println("UI全量重建完成（模拟重启效果）")
}

// 辅助函数：分离收藏项和普通项
func splitItemsByFavorite(items []*model.ClipboardItem) (favorites, normal []*model.ClipboardItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	for _, item := range items {
		if item.IsFavorite {
			favorites = append(favorites, item)
		}
		normal = append(normal, item)
	}
	return
}

// UpdateHistory 更新历史记录（改为触发全量重建）
func (w *Window) UpdateHistory(_ []*model.ClipboardItem) {
	log.Println("收到数据更新通知，触发UI全量重建")
	// 直接调用全量重建（忽略入参，重新从存储层加载最新数据）
	fyne.Do(func() {
		w.rebuildFullUI()
	})
}
