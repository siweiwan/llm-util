package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const logDir = "logs"

// Init 初始化日志系统，将日志写入 logs/YYYY-MM-DD.log 文件。
// 调用一次即可，通常在 main 函数启动时调用。
func Init() error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	logFile := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	handler := slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
	return nil
}
