package component

import (
	"fyne.io/fyne/v2/widget"
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
	search.OnChanged = search.handleSearch

	return search
}

// 处理搜索
func (s *SearchBar) handleSearch(text string) {
	if s.onSearch != nil {
		s.onSearch(text)
	}
}
