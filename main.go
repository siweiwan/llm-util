package main

import (
	"fmt"
	"llm-util/internal/app"
	"llm-util/tui"
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	app.EnsureEnvFile()

	apiKey := os.Getenv("LLM_API_KEY")
	appId := os.Getenv("LLM_APP_ID")
	poolSize, _ := strconv.Atoi(os.Getenv("POOL_SIZE"))
	if poolSize <= 0 {
		poolSize = 10
	}

	a := app.New(apiKey, appId)

	model := tui.NewModel(a.APIKey, a.AppId, poolSize)
	model.OnSaveSettings = func(key, id string, ps int) error {
		a.APIKey = key
		a.AppId = id
		return app.SaveEnvFile(key, id, ps)
	}
	model.OnRunModeA = a.RunModeA
	model.OnRunPDF = a.RunPdfBatchQuery
	model.OnRunDIY = a.RunDIYQueryRule
	model.OnRunWorkflow = a.RunWorkflowQueryRule

	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
