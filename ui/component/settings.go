package component

import (
	"clipboard/config"
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"os"
	"path/filepath"
	"strconv"
)

// SettingsPanel 应用设置面板
type SettingsPanel struct {
	*fyne.Container
	window          fyne.Window
	storageType     *widget.Select
	maxItemsEntry   *widget.Entry
	customPathCheck *widget.Check
	jsonPathEntry   *widget.Entry
	browseBtn       *widget.Button
	saveBtn         *widget.Button
	mysqlSettings   *fyne.Container // MySQL设置容器
	jsonSettings    *fyne.Container // JSON设置容器
	saveCallback    func(*config.StorageConfig)
}

// NewSettingsPanel 创建设置面板
func NewSettingsPanel(window fyne.Window, cfg *config.StorageConfig, saveCallback func(*config.StorageConfig)) *SettingsPanel {
	p := &SettingsPanel{
		window:       window,
		saveCallback: saveCallback,
	}

	// 初始化存储类型选择器
	p.storageType = widget.NewSelect(
		[]string{string(config.StorageTypeJSON), string(config.StorageTypeMySQL)},
		nil,
	)

	// 初始化最大项目数输入框
	p.maxItemsEntry = widget.NewEntry()
	p.maxItemsEntry.SetText(strconv.Itoa(cfg.MaxItems))

	// 初始化JSON存储相关控件
	p.customPathCheck = widget.NewCheck("使用自定义路径", func(checked bool) {
		p.jsonPathEntry.Disable()
		p.browseBtn.Disable()
		if checked {
			p.jsonPathEntry.Enable()
			p.browseBtn.Enable()
		}
	})
	p.customPathCheck.SetChecked(cfg.CustomPath)

	p.jsonPathEntry = widget.NewEntry()
	p.jsonPathEntry.SetText(cfg.JSONPath)

	p.browseBtn = widget.NewButton("浏览...", func() {
		dialog.ShowFolderOpen(func(dir fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, p.window)
				return
			}
			if dir != nil {
				p.jsonPathEntry.SetText(dir.Path())
			}
		}, p.window)
	})

	// 创建JSON设置容器
	p.jsonSettings = container.NewVBox(
		container.NewHBox(p.customPathCheck),
		container.NewHBox(
			widget.NewLabel("存储路径:"),
			p.jsonPathEntry,
			p.browseBtn,
		),
	)

	// 初始化MySQL设置控件
	mysqlHostEntry := widget.NewEntry()
	mysqlHostEntry.SetText(cfg.MySQL.Host)

	mysqlPortEntry := widget.NewEntry()
	mysqlPortEntry.SetText(strconv.Itoa(cfg.MySQL.Port))

	mysqlUserEntry := widget.NewEntry()
	mysqlUserEntry.SetText(cfg.MySQL.User)

	mysqlPassEntry := widget.NewPasswordEntry()
	mysqlPassEntry.SetText(cfg.MySQL.Password)

	mysqlDBEntry := widget.NewEntry()
	mysqlDBEntry.SetText(cfg.MySQL.Database)

	// 创建MySQL设置容器
	p.mysqlSettings = container.NewVBox(
		container.NewHBox(widget.NewLabel("主机:"), mysqlHostEntry),
		container.NewHBox(widget.NewLabel("端口:"), mysqlPortEntry),
		container.NewHBox(widget.NewLabel("用户名:"), mysqlUserEntry),
		container.NewHBox(widget.NewLabel("密码:"), mysqlPassEntry),
		container.NewHBox(widget.NewLabel("数据库:"), mysqlDBEntry),
	)

	// 设置保存按钮（回调由windows.go实现重建）
	p.saveBtn = widget.NewButton("保存设置", func() {
		// 解析最大项目数
		maxItems, err := strconv.Atoi(p.maxItemsEntry.Text)
		if err != nil || maxItems <= 0 {
			maxItems = 100
		}

		// 解析端口
		port, err := strconv.Atoi(mysqlPortEntry.Text)
		if err != nil || port <= 0 || port > 65535 {
			port = 3306
		}

		// 验证并处理JSON路径
		jsonPath := p.jsonPathEntry.Text
		if p.customPathCheck.Checked && jsonPath != "" {
			if err := os.MkdirAll(jsonPath, 0755); err != nil {
				dialog.ShowError(errors.New("无法创建JSON存储目录: "+err.Error()), p.window)
				return
			}
		} else if !p.customPathCheck.Checked {
			appDataDir, _ := os.UserConfigDir()
			jsonPath = filepath.Join(appDataDir, "clipboard-manager", "history")
			os.MkdirAll(jsonPath, 0755)
		}

		// 创建配置对象
		newCfg := &config.StorageConfig{
			Type:       config.StorageType(p.storageType.Selected),
			JSONPath:   jsonPath,
			CustomPath: p.customPathCheck.Checked,
			MySQL: config.MySQLConfig{
				Host:     mysqlHostEntry.Text,
				Port:     port,
				User:     mysqlUserEntry.Text,
				Password: mysqlPassEntry.Text,
				Database: mysqlDBEntry.Text,
			},
			MaxItems: maxItems,
		}

		// 调用回调（由windows.go触发重建）
		if p.saveCallback != nil {
			p.saveCallback(newCfg)
		}

		dialog.ShowInformation("设置已保存", "您的设置已成功保存（已触发UI重建）", p.window)
	})

	// 设置存储类型变更回调
	p.storageType.OnChanged = func(value string) {
		p.updateStorageSettingsVisibility(value)
	}

	// 设置初始选中值
	p.storageType.SetSelected(string(cfg.Type))
	p.updateStorageSettingsVisibility(string(cfg.Type))

	// 构建主容器
	p.Container = container.NewVBox(
		widget.NewLabel("存储类型:"),
		p.storageType,
		widget.NewSeparator(),
		widget.NewLabel("最大历史项目数:"),
		p.maxItemsEntry,
		widget.NewSeparator(),
		widget.NewLabel("存储设置:"),
		container.NewVBox(p.jsonSettings, p.mysqlSettings),
		layout.NewSpacer(),
		p.saveBtn,
	)

	return p
}

// updateStorageSettingsVisibility 根据存储类型更新设置面板可见性
func (p *SettingsPanel) updateStorageSettingsVisibility(storageType string) {
	if p.jsonSettings == nil || p.mysqlSettings == nil {
		return
	}

	if storageType == string(config.StorageTypeJSON) {
		p.jsonSettings.Show()
		p.mysqlSettings.Hide()
	} else if storageType == string(config.StorageTypeMySQL) {
		p.jsonSettings.Hide()
		p.mysqlSettings.Show()
	}
}
