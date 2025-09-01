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
	"strings"
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

	// 分离收藏项和普通项
	favoriteItems := []*model.ClipboardItem{}
	normalItems := []*model.ClipboardItem{}
	for _, item := range items {
		if item.IsFavorite {
			favoriteItems = append(favoriteItems, item)
		} else {
			normalItems = append(normalItems, item)
		}
	}

	// 创建搜索框
	w.searchBar = component.NewSearchBar(func(text string) {
		results, err := w.storage.Search(text)
		if err == nil {
			// 搜索结果也需要分离到两个列表
			favResults, normalResults := splitItemsByFavorite(results)
			w.historyList.UpdateItems(normalResults)
			w.favoriteList.UpdateItems(favResults)
		}
	})

	// 创建普通历史列表
	w.historyList = component.NewHistoryList(
		normalItems,
		func(item *model.ClipboardItem) {
			w.clipboard.SetContent(item)
		},
		func(id string) {
			// 收藏状态变更后全量刷新两个列表
			updatedItems, err := w.storage.ToggleFavorite(id)
			if err == nil {
				fav, normal := splitItemsByFavorite(updatedItems)
				w.historyList.UpdateItems(normal) // 全量刷新普通列表
				w.favoriteList.UpdateItems(fav)   // 全量刷新收藏列表
			}
		},
		func(id string) {
			// 删除项后全量刷新两个列表
			updatedItems, err := w.storage.DeleteItem(id)
			if err == nil {
				fav, normal := splitItemsByFavorite(updatedItems)

				// 先更新数据源，再强制刷新UI
				w.favoriteList.UpdateItems(fav)
				w.historyList.UpdateItems(normal)

				// 额外添加：如果收藏列表为空，给用户提示
				if len(fav) == 0 {
					w.favoriteList.UpdateItems([]*model.ClipboardItem{
						{
							ID:      "empty",
							Type:    model.TypeText,
							Content: "暂无收藏内容",
						},
					})
				}
			} else {
				log.Printf("删除失败: %v", err)
			}
		},
	)

	// 创建收藏列表
	w.favoriteList = component.NewHistoryList(
		favoriteItems,
		func(item *model.ClipboardItem) {
			w.clipboard.SetContent(item)
		},
		func(id string) {
			// 收藏状态变更后全量刷新两个列表
			updatedItems, err := w.storage.ToggleFavorite(id)
			if err == nil {
				fav, normal := splitItemsByFavorite(updatedItems)
				w.historyList.UpdateItems(normal) // 全量刷新普通列表
				w.favoriteList.UpdateItems(fav)   // 全量刷新收藏列表
			}
		},
		func(id string) {
			// 删除项后全量刷新两个列表
			updatedItems, err := w.storage.DeleteItem(id)
			if err == nil {
				fav, normal := splitItemsByFavorite(updatedItems)
				w.historyList.UpdateItems(normal)
				w.favoriteList.UpdateItems(fav)
			}
		},
	)

	// 创建主内容区域（普通历史）
	historyContent := container.NewBorder(
		w.searchBar,
		nil, nil, nil,
		w.historyList,
	)

	// 收藏内容区域
	favoriteContent := container.NewBorder(
		nil, nil, nil, nil,
		w.favoriteList,
	)

	// 加载配置创建设置面板
	cfg, _ := config.Load()
	w.settingsPanel = component.NewSettingsPanel(w.Window, &cfg.Storage, w.onSaveSettings)

	// 创建标签页
	w.contentTabs = container.NewAppTabs(
		container.NewTabItemWithIcon("历史记录", theme.HistoryIcon(), historyContent),
		container.NewTabItemWithIcon("我的收藏", theme.ConfirmIcon(), favoriteContent),
		container.NewTabItemWithIcon("设置", theme.SettingsIcon(), w.settingsPanel),
	)

	// 设置主内容
	w.SetContent(w.contentTabs)
}

// 辅助函数：分离收藏项和普通项
func splitItemsByFavorite(items []*model.ClipboardItem) (favorites, normal []*model.ClipboardItem) {
	// 先按时间排序，确保最新的在前面
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	for _, item := range items {
		if item.IsFavorite {
			favorites = append(favorites, item)
		}
		// 普通列表包含所有项（包括收藏项）
		normal = append(normal, item)
	}
	return
}

// UpdateHistory 更新历史记录列表
func (w *Window) UpdateHistory(items []*model.ClipboardItem) {
	currentSearch := w.searchBar.Text
	if currentSearch == "" {
		// 同时更新两个列表，收藏项会同时出现在两个列表中
		fav, normal := splitItemsByFavorite(items)
		w.historyList.UpdateItems(normal)
		w.favoriteList.UpdateItems(fav)
		return
	}

	// 执行搜索时过滤内容
	var results []*model.ClipboardItem
	keyword := strings.ToLower(currentSearch)
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Content), keyword) {
			results = append(results, item)
			continue
		}
		if (item.Type == model.TypeImage && strings.Contains(keyword, "图片")) ||
			(item.Type == model.TypeFile && strings.Contains(keyword, "文件")) {
			results = append(results, item)
		}
	}

	// 搜索结果也应用相同的显示规则
	favResults, normalResults := splitItemsByFavorite(results)
	fyne.Do(func() {
		w.historyList.UpdateItems(normalResults)
		w.favoriteList.UpdateItems(favResults)
	})
}
