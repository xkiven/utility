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
	"time"
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
				// 强制UI刷新
				w.historyList.Refresh()
				w.favoriteList.Refresh()
			}
		},
		func(id string) {
			updatedItems, err := w.storage.DeleteItem(id)
			if err == nil {
				fav, normal := splitItemsByFavorite(updatedItems)
				w.historyList.UpdateItems(normal)

				// 处理空收藏列表提示（修复后）
				if len(fav) == 0 {
					// 添加非交互的提示项（ID含特殊标识，避免与真实项冲突）
					w.favoriteList.UpdateItems([]*model.ClipboardItem{{
						ID:         "empty-favorite-placeholder", // 明确占位符ID
						Type:       model.TypeText,
						Content:    "暂无收藏内容",
						IsFavorite: false,
						Timestamp:  time.Time{}, // 空时间避免排序干扰
					}})
				} else {
					w.favoriteList.UpdateItems(fav) // 有数据时正常显示
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
				// 同时更新两个列表，保持状态一致
				w.favoriteList.UpdateItems(fav)
				w.historyList.UpdateItems(normal)
			}
		},
		func(id string) {
			// 删除项后全量刷新两个列表
			updatedItems, err := w.storage.DeleteItem(id)
			if err == nil {
				fav, normal := splitItemsByFavorite(updatedItems)
				w.historyList.UpdateItems(normal)
				w.favoriteList.UpdateItems(fav)
				// 强制UI刷新
				w.historyList.Refresh()
				w.favoriteList.Refresh()
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
	// 按时间降序排序（最新在前）
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	for _, item := range items {
		if item.IsFavorite {
			favorites = append(favorites, item) // 收藏列表：只包含收藏项
		}
		// 关键修改：普通列表保留所有项（包括已收藏的）
		normal = append(normal, item)
	}
	return
}

// UpdateHistory 更新历史记录列表
func (w *Window) UpdateHistory(items []*model.ClipboardItem) {
	currentSearch := w.searchBar.Text
	var results []*model.ClipboardItem

	log.Printf("开始更新历史记录，原始数据量: %d，搜索关键词: %s", len(items), currentSearch)

	// 1. 去重逻辑（保留不变）
	uniqueItems := make(map[string]*model.ClipboardItem)
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		if existing, ok := uniqueItems[item.ID]; !ok || item.Timestamp.After(existing.Timestamp) {
			uniqueItems[item.ID] = item
		}
	}
	deduplicated := make([]*model.ClipboardItem, 0, len(uniqueItems))
	for _, item := range uniqueItems {
		deduplicated = append(deduplicated, item)
	}
	log.Printf("去重后数据量: %d", len(deduplicated))

	// 2. 搜索逻辑（保留不变）
	if currentSearch != "" {
		keyword := strings.ToLower(currentSearch)
		for _, item := range deduplicated {
			if strings.Contains(strings.ToLower(item.Content), keyword) ||
				(item.Type == model.TypeImage && strings.Contains(keyword, "图片")) ||
				(item.Type == model.TypeFile && strings.Contains(keyword, "文件")) {
				results = append(results, item)
			}
		}
		log.Printf("搜索完成，匹配结果: %d 条", len(results))
	} else {
		results = deduplicated
		log.Printf("无搜索关键词，使用去重后数据: %d 条", len(results))
	}

	// 3. 分离列表数据（保留不变）
	favResults, normalResults := splitItemsByFavorite(results)
	log.Printf("分离收藏项: %d 条，普通项: %d 条", len(favResults), len(normalResults))

	// 关键修复：原子化更新两个列表，避免中间状态
	fyne.Do(func() {
		// 直接加载新数据（无需先清空，UpdateItems 内部已深拷贝并刷新）
		w.historyList.UpdateItems(normalResults)
		w.favoriteList.UpdateItems(favResults)

		// 仅刷新标签页容器（避免冗余刷新）
		w.contentTabs.Refresh()
		log.Println("UI列表已原子化刷新")
	})
}
