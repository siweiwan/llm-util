package main

import (
	"fmt"
	"llm-util/internal/app"
	"llm-util/tui"
	"llm-util/util/logger"
	"log/slog"
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	app.EnsureEnvFile()

	// 初始化日志系统，失败不阻断启动
	if err := logger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "日志初始化失败: %v\n", err)
	}

	apiKey := os.Getenv("LLM_API_KEY")
	appId := os.Getenv("LLM_APP_ID")
	poolSize, _ := strconv.Atoi(os.Getenv("POOL_SIZE"))
	if poolSize <= 0 {
		poolSize = 10
	}

	slog.Info("应用启动", "appId", appId, "poolSize", poolSize)

	a := app.New(apiKey, appId)

	model := tui.NewModel(a.APIKey, a.AppId, poolSize)
	model.OnSaveSettings = func(key, id string, ps int) error {
		a.APIKey = key
		a.AppId = id
		slog.Info("设置已保存", "appId", id, "poolSize", ps)
		return app.SaveEnvFile(key, id, ps)
	}
	model.OnRunModeA = a.RunModeA
	model.OnRunPDF = a.RunPdfBatchQuery
	model.OnRunDIY = a.RunDIYQueryRule
	model.OnRunWorkflow = a.RunWorkflowQueryRule

	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		slog.Error("程序异常退出", "err", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
