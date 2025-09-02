package app

import (
	"clipboard/clipboard"
	"clipboard/config"
	"clipboard/storage"
	"clipboard/ui"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"log"
)

// Application 应用程序核心
type Application struct {
	fyneApp fyne.App
	config  *config.AppConfig
	storage storage.Storage
	monitor *clipboard.Monitor
	window  *ui.Window
}

// New 创建应用实例
func New() (*Application, error) {
	// 初始化Fyne应用
	fyneApp := app.New()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	// 创建存储
	store, err := storage.NewStorage(&cfg.Storage)
	if err != nil {
		return nil, err
	}

	// 创建剪贴板监听器
	monitor, err := clipboard.NewMonitor(store)
	if err != nil {
		return nil, err
	}
	// 创建应用实例
	app := &Application{
		fyneApp: fyneApp,
		config:  cfg,
		storage: store,
		monitor: monitor,
	}

	// 创建主窗口
	app.window = ui.NewWindow(fyneApp, store, monitor, app.handleSaveSettings)

	// 设置剪贴板变化处理
	app.setupClipboardListener()

	return app, nil
}

// Run 运行应用
func (a *Application) Run() {
	a.window.ShowAndRun()
	a.storage.Close()
	a.monitor.Stop()
}

// 设置剪贴板监听器
func (a *Application) setupClipboardListener() {
	// 先启动剪贴板监控（原代码中可能遗漏，需确认补充，避免监控未运行）
	if err := a.monitor.Start(); err != nil {
		log.Printf("启动剪贴板监控失败: %v", err)
		return
	}

	// 启动协程监听剪贴板变化通知
	go func() {
		for {
			select {
			// 接收存储更新后的所有项
			case items := <-a.monitor.ChangeChan():

				fyne.Do(func() {
					a.window.UpdateHistory(items)
					log.Printf("应用层触发UI更新，新数据量：%d", len(items))
				})
			// 增加退出case，避免协程泄露
			case <-a.monitor.StopChan:
				log.Println("剪贴板监听协程退出")
				return
			}
		}
	}()
}

// 处理保存设置
func (a *Application) handleSaveSettings(newStorageCfg *config.StorageConfig) {
	// 更新配置
	a.config.Storage = *newStorageCfg

	// 保存配置
	config.Save(a.config)

	// 停止当前监听器
	a.monitor.Stop()

	// 关闭当前存储
	a.storage.Close()

	// 重新创建存储
	newStorage, err := storage.NewStorage(newStorageCfg)
	if err != nil {
		return
	}
	a.storage = newStorage

	// 重新创建剪贴板监听器
	a.monitor, _ = clipboard.NewMonitor(newStorage)
	a.setupClipboardListener()

	// 重新加载历史记录
	items, _ := a.storage.LoadItems()
	a.window.UpdateHistory(items)
}
