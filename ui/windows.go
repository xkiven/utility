package ui

import (
	"clipboard/config"
	"clipboard/model"
	"clipboard/storage"
	"clipboard/ui/component"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
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
	clipboard      ClipboardSetter // 用于设置剪贴板内容的接口
}

// ClipboardSetter 剪贴板设置接口
type ClipboardSetter interface {
	SetContent(item *model.ClipboardItem) error
}

// NewWindow 创建主窗口
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

	// 初始化UI
	w.initUI()

	return w
}

// 初始化UI
func (w *Window) initUI() {
	// 加载初始数据
	items, _ := w.storage.LoadItems()

	// 创建搜索框
	w.searchBar = component.NewSearchBar(func(text string) {
		results, err := w.storage.Search(text)
		if err == nil {
			w.historyList.UpdateItems(results)
		}
	})

	// 创建历史列表
	w.historyList = component.NewHistoryList(
		items,
		func(item *model.ClipboardItem) {
			// 点击项时复制到剪贴板
			w.clipboard.SetContent(item)
		},
		func(id string) {
			updatedItems, err := w.storage.ToggleFavorite(id)
			if err == nil {
				w.historyList.UpdateItems(updatedItems)
			}
		},
		func(id string) {
			updatedItems, err := w.storage.DeleteItem(id)
			if err == nil {
				w.historyList.UpdateItems(updatedItems)
			}
		},
	)

	// 创建主内容区域
	mainContent := container.NewBorder(
		w.searchBar,
		nil, nil, nil,
		w.historyList,
	)

	// 加载配置创建设置面板 - 传递窗口引用
	cfg, _ := config.Load()
	w.settingsPanel = component.NewSettingsPanel(w.Window, &cfg.Storage, w.onSaveSettings)

	// 创建标签页
	w.contentTabs = container.NewAppTabs(
		container.NewTabItemWithIcon("历史记录", theme.HistoryIcon(), mainContent),
		container.NewTabItemWithIcon("设置", theme.SettingsIcon(), w.settingsPanel),
	)

	// 设置主内容
	w.SetContent(w.contentTabs)
}

// UpdateHistory 更新历史记录列表
func (w *Window) UpdateHistory(items []*model.ClipboardItem) {
	currentSearch := w.searchBar.Text
	if currentSearch == "" {
		w.historyList.UpdateItems(items)
	} else {
		// 如果有搜索内容，重新执行搜索
		results, err := w.storage.Search(currentSearch)
		if err == nil {
			w.historyList.UpdateItems(results)
		}
	}
}
