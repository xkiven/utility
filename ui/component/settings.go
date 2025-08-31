package component

import (
	"clipboard/config"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"os"
	"strconv"
)

// SettingsPanel 设置面板组件
type SettingsPanel struct {
	*container.Scroll
	storageType     *widget.Select
	maxItemsEntry   *widget.Entry
	mysqlHostEntry  *widget.Entry
	mysqlPortEntry  *widget.Entry
	mysqlUserEntry  *widget.Entry
	mysqlPassEntry  *widget.Entry
	mysqlDBEntry    *widget.Entry
	jsonPathEntry   *widget.Entry
	customPathCheck *widget.Check
	saveCallback    func(*config.StorageConfig)
	mysqlSettings   *fyne.Container
	jsonSettings    *fyne.Container
	window          fyne.Window
}

// NewSettingsPanel 创建设置面板
func NewSettingsPanel(window fyne.Window, cfg *config.StorageConfig, saveCallback func(*config.StorageConfig)) *SettingsPanel {
	panel := &SettingsPanel{
		saveCallback: saveCallback,
		window:       window,
	}

	// 存储类型选择
	panel.storageType = widget.NewSelect(
		[]string{string(config.StorageTypeJSON), string(config.StorageTypeMySQL)},
		func(value string) {
			panel.updateStorageSettingsVisibility(value)
		},
	)
	panel.storageType.SetSelected(string(cfg.Type))

	// 最大项目数
	panel.maxItemsEntry = widget.NewEntry()
	panel.maxItemsEntry.SetText(strconv.Itoa(cfg.MaxItems))

	// JSON路径设置
	panel.customPathCheck = widget.NewCheck("使用自定义存储路径", func(checked bool) {
		// 当勾选时启用路径输入框，否则禁用
		panel.jsonPathEntry.Disable()
		if checked {
			panel.jsonPathEntry.Enable()
		}
	})
	panel.customPathCheck.SetChecked(cfg.CustomPath)

	panel.jsonPathEntry = widget.NewEntry()
	panel.jsonPathEntry.SetText(cfg.JSONPath)
	if !cfg.CustomPath {
		panel.jsonPathEntry.Disable()
	}

	browseBtn := widget.NewButton("浏览...", func() {
		dialog.ShowFolderOpen(func(dir fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, panel.window)
				return
			}
			if dir != nil {
				panel.jsonPathEntry.SetText(dir.Path()) // 选择后更新路径
			}
		}, panel.window)
	})

	panel.jsonSettings = container.NewVBox(
		panel.customPathCheck,
		container.NewBorder(
			nil, nil, nil, browseBtn,
			panel.jsonPathEntry,
		),
	)

	// MySQL设置
	panel.mysqlHostEntry = widget.NewEntry()
	panel.mysqlHostEntry.SetText(cfg.MySQL.Host)

	panel.mysqlPortEntry = widget.NewEntry()
	panel.mysqlPortEntry.SetText(strconv.Itoa(cfg.MySQL.Port))

	panel.mysqlUserEntry = widget.NewEntry()
	panel.mysqlUserEntry.SetText(cfg.MySQL.User)

	panel.mysqlPassEntry = widget.NewEntry()
	panel.mysqlPassEntry.SetText(cfg.MySQL.Password)
	panel.mysqlPassEntry.Password = true

	panel.mysqlDBEntry = widget.NewEntry()
	panel.mysqlDBEntry.SetText(cfg.MySQL.Database)

	// MySQL设置容器
	panel.mysqlSettings = container.NewVBox(
		widget.NewLabel("MySQL 主机:"),
		panel.mysqlHostEntry,
		widget.NewLabel("MySQL 端口:"),
		panel.mysqlPortEntry,
		widget.NewLabel("MySQL 用户名:"),
		panel.mysqlUserEntry,
		widget.NewLabel("MySQL 密码:"),
		panel.mysqlPassEntry,
		widget.NewLabel("MySQL 数据库:"),
		panel.mysqlDBEntry,
	)

	// 测试连接按钮
	testMysqlBtn := widget.NewButton("测试连接", func() {
		// 这里可以添加测试MySQL连接的逻辑
		dialog.ShowInformation("测试结果", "连接测试功能尚未实现", panel.window)
	})
	panel.mysqlSettings.Add(testMysqlBtn)

	// 保存按钮
	saveBtn := widget.NewButton("保存设置", func() {
		panel.saveSettings()
	})

	// 组装面板
	content := container.NewVBox(
		widget.NewLabel("存储类型:"),
		panel.storageType,
		widget.NewLabel("最大历史记录数:"),
		panel.maxItemsEntry,
		widget.NewSeparator(),
		widget.NewLabel("JSON 存储设置:"),
		panel.jsonSettings,
		widget.NewSeparator(),
		widget.NewLabel("MySQL 设置:"),
		panel.mysqlSettings,
		saveBtn,
	)

	// 初始显示正确的存储设置
	panel.updateStorageSettingsVisibility(string(cfg.Type))

	panel.Scroll = container.NewScroll(content)
	return panel
}

// 更新存储设置可见性
func (p *SettingsPanel) updateStorageSettingsVisibility(storageType string) {
	if storageType == string(config.StorageTypeMySQL) {
		p.mysqlSettings.Show()
		p.jsonSettings.Hide()
	} else {
		p.mysqlSettings.Hide()
		p.jsonSettings.Show()
	}
}

// 保存设置
func (p *SettingsPanel) saveSettings() {
	if p.saveCallback == nil {
		return
	}

	// 解析最大项目数
	maxItems, err := strconv.Atoi(p.maxItemsEntry.Text)
	if err != nil || maxItems <= 0 {
		maxItems = 100
	}

	// 解析端口
	port, err := strconv.Atoi(p.mysqlPortEntry.Text)
	if err != nil || port <= 0 {
		port = 3306
	}

	// 验证JSON路径
	jsonPath := p.jsonPathEntry.Text
	if p.customPathCheck.Checked && jsonPath != "" {
		// 确保目录存在
		if err := os.MkdirAll(jsonPath, 0755); err != nil {
			dialog.ShowError(err, p.window)
			return
		}
	}

	// 创建配置对象
	cfg := &config.StorageConfig{
		Type:       config.StorageType(p.storageType.Selected),
		JSONPath:   jsonPath,
		CustomPath: p.customPathCheck.Checked,
		MySQL: config.MySQLConfig{
			Host:     p.mysqlHostEntry.Text,
			Port:     port,
			User:     p.mysqlUserEntry.Text,
			Password: p.mysqlPassEntry.Text,
			Database: p.mysqlDBEntry.Text,
		},
		MaxItems: maxItems,
	}

	// 调用回调保存配置
	p.saveCallback(cfg)

	// 显示保存成功提示
	dialog.ShowInformation("设置已保存", "您的设置已成功保存", p.window)
}
