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

// New 创建应用实例（保持原逻辑）
func New() (*Application, error) {
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

	// 设置剪贴板监听器
	app.setupClipboardListener()

	return app, nil
}

// Run 运行应用（保持原逻辑）
func (a *Application) Run() {
	a.window.ShowAndRun()
	a.storage.Close()
	a.monitor.Stop()
}

// 设置剪贴板监听器（修改为触发全量重建）
func (a *Application) setupClipboardListener() {
	// 启动剪贴板监控
	if err := a.monitor.Start(); err != nil {
		log.Printf("启动剪贴板监控失败: %v", err)
		return
	}

	// 监听剪贴板变化（触发UI全量重建）
	go func() {
		for {
			select {
			case <-a.monitor.ChangeChan():
				log.Println("应用层收到剪贴板变化，触发UI全量重建")
				fyne.Do(func() {
					a.window.UpdateHistory(nil) // 空入参触发重建
				})
			case <-a.monitor.StopChan:
				log.Println("剪贴板监听协程退出")
				return
			}
		}
	}()
}

// 处理保存设置（修改为触发全量重建）
func (a *Application) handleSaveSettings(newStorageCfg *config.StorageConfig) {
	// 更新配置
	a.config.Storage = *newStorageCfg
	config.Save(a.config)

	// 停止当前监听器
	a.monitor.Stop()

	// 关闭当前存储
	a.storage.Close()

	// 重建存储实例
	newStorage, err := storage.NewStorage(newStorageCfg)
	if err != nil {
		log.Printf("重建存储失败: %v", err)
		return
	}
	a.storage = newStorage

	// 重建监听器实例
	a.monitor, _ = clipboard.NewMonitor(newStorage)
	a.setupClipboardListener()

	// 触发UI全量重建
	log.Println("设置保存完成，触发UI全量重建")
	fyne.Do(func() {
		a.window.UpdateHistory(nil)
	})
}
