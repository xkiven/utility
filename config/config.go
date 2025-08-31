package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// StorageType 存储类型
type StorageType string

const (
	StorageTypeJSON  StorageType = "json"
	StorageTypeMySQL StorageType = "mysql"
)

// StorageConfig 存储配置
type StorageConfig struct {
	Type       StorageType `json:"type"`
	JSONPath   string      `json:"jsonPath"`
	CustomPath bool        `json:"customPath"` // 是否使用自定义路径
	MySQL      MySQLConfig `json:"mySQL"`
	MaxItems   int         `json:"maxItems"`
}

// MySQLConfig MySQL数据库配置
type MySQLConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

// AppConfig 应用配置
type AppConfig struct {
	Storage StorageConfig `json:"storage"`
	Hotkey  string        `json:"hotkey"`
}

// ConfigPath 配置文件路径
func configPath() string {
	appDataDir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".", "config.json")
	}

	configDir := filepath.Join(appDataDir, "clipboard-manager")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		log.Printf("创建配置目录失败: %v，将使用当前目录", err)
		return filepath.Join(".", "config.json")
	}
	return filepath.Join(configDir, "config.json")

}

func Load() (*AppConfig, error) {
	path := configPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return defaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if config.Storage.MaxItems <= 0 {
		config.Storage.MaxItems = 100
	}

	if !config.Storage.CustomPath {
		appDataDir, _ := os.UserConfigDir()
		config.Storage.JSONPath = filepath.Join(appDataDir, "clipboard-manager", "history")
	}

	return &config, nil
}

func Save(config *AppConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Printf("配置序列化为JSON失败: %v", err) // 记录序列化错误
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}

// 默认配置
func defaultConfig() *AppConfig {
	appDataDir, _ := os.UserConfigDir()
	storagePath := filepath.Join(appDataDir, "clipboard-manager", "history")

	return &AppConfig{
		Storage: StorageConfig{
			Type:       StorageTypeJSON,
			JSONPath:   storagePath,
			CustomPath: false,
			MySQL: MySQLConfig{
				Host:     "localhost",
				Port:     3306,
				User:     "root",
				Password: "",
				Database: "clipboard",
			},
			MaxItems: 100,
		},
		Hotkey: "Ctrl+Shift+V",
	}
}
