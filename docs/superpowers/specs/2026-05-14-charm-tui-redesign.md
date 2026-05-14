# Charm TUI Redesign

## Context

当前 `main.go` (877行) 使用 `fmt.Scanln`、`bufio.Reader`、原始 ANSI 转义码做交互，体验简陋。在不改动业务逻辑的前提下，用 Charm 生态库（Bubble Tea + Lip Gloss + Bubbles）重构全部交互层。

**约束**：后端逻辑零改动。以下包保持不变：`file/qwen/*`、`ai/app/*`、`util/*`、`conf/*`、`app/*`。

## Dependencies

```go
github.com/charmbracelet/bubbletea
github.com/charmbracelet/lipgloss
github.com/charmbracelet/bubbles
```

## Files

| 文件 | 说明 |
|---|---|
| `tui/tui.go` | Bubble Tea 入口、主模型、页面路由 |
| `tui/styles.go` | Lip Gloss 主题：色板、边框、排版 |
| `tui/menu.go` | 主菜单 + 规则子菜单视图 |
| `tui/chat.go` | 对话视图 |
| `tui/batch.go` | 批量处理视图 |
| `main.go` | 精简为配置加载 + 启动 `tea.NewProgram()` |

## Architecture

### 主模型

```go
type View int
const (
    ViewMainMenu View = iota
    ViewChat
    ViewRulesMenu
    ViewRuleCase
    ViewRulePDF
    ViewRuleDIY
    ViewRuleWorkflow
)

type Model struct {
    view          View
    width, height int

    apiKey  string
    appId   string
    history []Message

    mainMenu  list.Model
    rulesMenu list.Model
    chat      chatPanel
    batch     batchPanel
}
```

### 页面路由

`Update()` 先判断 `tea.WindowSizeMsg` / 全局快捷键（`ctrl+c` 退出），再按 `m.view` 分发到对应处理方法。`View()` 同理。

### 导航

修改 `m.view` 实现页面切换。`list.Model` 选中项 → 进入对应视图，`esc` → 返回上级。

## Views

### 主菜单

`bubbles/list` 渲染 4 项（对话/新对话/规则模式/退出），标题用 Lip Gloss 渐变色。`↑/↓`/`j/k` 导航，`enter` 确认，`q`/`esc` 退出程序。

### 规则子菜单

`bubbles/list` 渲染 4 项，每项带描述文字。与主菜单交互一致，`esc` 返回主菜单。

### 对话视图

- 底部 `bubbles/textarea`：多行输入，`Ctrl+Enter` 发送
- 上部 `bubbles/viewport`：滚动对话历史，自动跟随最新
- 等待态 `bubbles/spinner`：请求期间禁用输入，显示动画
- `esc` 返回主菜单，`Ctrl+N` 新对话
- `sendRequest()` 包装为 `tea.Cmd`，结果通过 `chatResponseMsg` 返回

### 批量处理视图

- `bubbles/progress`：整体进度条
- 实时日志：当前处理文件名、状态（✅完成/🔄处理中/⬜等待）
- 开始前输入并发数，PDF 规则额外输入问题
- `ants` 协程池在 goroutine 中运行，通过 `batchMsg` channel 更新进度
- 完成后显示保存路径 + 成功/失败统计
- 处理完成后按 `q` 返回规则菜单

### 消息类型

```go
type chatResponseMsg struct { text string; err error }
type batchMsg struct { idx int; filename, status string; err error }
```

## Verification

1. `go build ./...` 编译通过
2. 手动运行，主菜单 ↑↓ 导航 / enter 进入 / esc 返回
3. 对话：输入问题 → 等待 spinner → 显示回复 → 滚动历史
4. 批处理：progress bar 推进 → 实时日志更新 → 完成后显示统计
5. 现有测试 `go test ./...` 全部通过
