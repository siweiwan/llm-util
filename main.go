package main

import (
	"fmt"
	"llm-util/conf"
	uploadfile "llm-util/file"
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

	// 构建配置
	cfg := conf.DefaultConfig()
	cfg.APIKey = os.Getenv("LLM_API_KEY")
	cfg.AppID = os.Getenv("LLM_APP_ID")
	if ws := os.Getenv("WORKSPACE_ID"); ws != "" {
		cfg.WorkspaceID = ws
	}
	if ps, _ := strconv.Atoi(os.Getenv("POOL_SIZE")); ps > 0 {
		cfg.PoolSize = ps
	}
	cfg.AccessKeyId = os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_ID")
	cfg.AccessKeySecret = os.Getenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET")
	conf.WORKSPACE_ID = cfg.WorkspaceID // 同步全局变量

	// 同步 AK/SK 到文件上传模块
	if cfg.AccessKeyId != "" {
		uploadfile.AccessKeyId = cfg.AccessKeyId
	}
	if cfg.AccessKeySecret != "" {
		uploadfile.AccessKeySecret = cfg.AccessKeySecret
	}

	slog.Info("应用启动", "appId", cfg.AppID, "workspaceId", cfg.WorkspaceID, "poolSize", cfg.PoolSize)

	a := app.New(cfg.APIKey, cfg.AppID)

	model := tui.NewModel(cfg)
	model.OnSaveSettings = func(c *conf.Config) error {
		a.APIKey = c.APIKey
		a.AppId = c.AppID
		conf.WORKSPACE_ID = c.WorkspaceID
		uploadfile.AccessKeyId = c.AccessKeyId
		uploadfile.AccessKeySecret = c.AccessKeySecret
		slog.Info("设置已保存", "appId", c.AppID, "workspaceId", c.WorkspaceID, "poolSize", c.PoolSize)
		return app.SaveEnvFile(c)
	}
	model.OnRunModeA = a.RunModeA
	model.OnRunModeB = a.RunModeB
	model.OnRunDIY = a.RunDIYQueryRule
	model.OnRunWorkflow = a.RunWorkflowQueryRule

	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		slog.Error("程序异常退出", "err", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
