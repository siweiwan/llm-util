@echo off
REM 程序名称，改成你想要的名字即可
REM set BIN_NAME=llm-util-v0.0.1
if "%BIN_NAME%"=="" set BIN_NAME=llm-util

set GOOS=windows
set GOARCH=amd64
go build -ldflags "-s -w" -o "%BIN_NAME%.exe" main.go

if %ERRORLEVEL% equ 0 (
    echo Built: %BIN_NAME%.exe
) else (
    echo Build failed.
)
