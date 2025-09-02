package component

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"log"
)

// SearchBar 搜索框组件
type SearchBar struct {
	*widget.Entry
	onSearch func(string) // 搜索回调函数
}

// NewSearchBar 创建搜索框
func NewSearchBar(onSearch func(string)) *SearchBar {
	search := &SearchBar{
		Entry:    widget.NewEntry(),
		onSearch: onSearch,
	}

	search.SetPlaceHolder("搜索剪贴板历史...")
	search.OnChanged = func(text string) {
		log.Printf("搜索关键词变更: %s，触发重建", text)
		search.onSearch(text) // 回调由windows.go的rebuildFullUI实现
	}

	return search
}

// 处理搜索
func (s *SearchBar) handleSearch(text string) {
	if s.onSearch != nil {
		// 确保在UI线程中执行搜索
		fyne.Do(func() {
			s.onSearch(text)
		})
	}
}
