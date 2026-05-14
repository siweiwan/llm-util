# main.go Code Refactoring

## 目标

将 `main.go`（848 行）按职责拆分为合理的包结构，`main.go` 仅保留入口。

## 拆分方案

```
main.go                              -- 入口：加载配置 + 启动 TUI（~50 行）

internal/app/
  app.go                             -- App 结构体（共享状态）+ New()
  chat.go                            -- SendRequest / SendRequestWithFile（HTTP 请求）
  batch_case.go                      -- RunCaseQueryRule（Excel 批量）
  batch_pdf.go                       -- RunPdfBatchQuery（PDF 批量）
  batch_diy.go                       -- RunDIYQueryRule（DIY 批量）
  batch_workflow.go                  -- RunWorkflowQueryRule（工作流批量）
  config.go                          -- ensureEnvFile / saveEnvFile（.env 持久化）
  helpers.go                         -- printQuestion / showLoading（控制台工具）
```

## App 结构体

```go
type App struct {
    APIKey  string
    AppId   string
    History []Message
}
```

所有原 package-level 变量（`apiKey`, `appId`, `conversationHistory`）收敛为 App 字段。原函数改为 App 方法。

## main.go 变化

```go
func main() {
    _ = godotenv.Load()
    app.EnsureEnvFile()
    
    apiKey := os.Getenv("LLM_API_KEY")
    appId := os.Getenv("LLM_APP_ID")
    
    a := app.New(apiKey, appId)
    
    model := tui.NewModel(a.APIKey, a.AppId)
    model.OnSaveSettings = func(key, id string) error {
        a.APIKey = key
        a.AppId = id
        return app.SaveEnvFile(key, id)
    }
    model.OnSend = func(prompt string, history []tui.Message) (string, error) {
        return a.SendRequest(prompt, convert(history))
    }
    // ... other callbacks similarly thin
    
    tea.NewProgram(model, tea.WithAltScreen()).Run()
}
```

## 不变

- 所有业务逻辑代码逐字搬运，不改一行逻辑
- 函数体、错误处理、注释全部保留
- TUI 回调接口不变

## 验证

1. `go build ./...` 通过
2. `go vet ./...` 通过  
3. `go test ./...` 仅预存集成测试失败
