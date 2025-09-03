@echo off
chcp 65001
set APP=clipboard-manager
set ICON=app.ico            :: 换成你自己的 ico 路径，没有就删掉 -icon 这一行
set OUT=%APP%.exe

echo 正在编译 %OUT% ...
go mod tidy
go build -ldflags "-s -w -H=windowsgui" -o %OUT% .

if exist %OUT% (
    echo 成功生成 %OUT%
    pause
) else (
    echo 编译失败，请检查错误
    pause
)