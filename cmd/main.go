package main

import (
	"clipboard/app"
)

func main() {
	// 创建应用
	application, err := app.New()
	if err != nil {
		//fmt.Printf("创建应用失败: %v\n", err)
		return
	}

	// 运行应用
	application.Run()
}
