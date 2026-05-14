# Charm TUI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace crude `fmt.Scanln`/ANSI CLI with Bubble Tea + Lip Gloss TUI, zero backend logic changes.

**Architecture:** Single Bubble Tea model with view routing. Backend functions stay in `main.go`; the `tui` package calls them via function callbacks. Batch functions get a progress channel for real-time TUI updates.

**Tech Stack:** Go, bubbletea, lipgloss, bubbles (list, textarea, viewport, spinner, progress)

---

## File Map

| File | Role |
|---|---|
| `tui/styles.go` | Lip Gloss theme |
| `tui/tui.go` | Main model, routing, shared types |
| `tui/menu.go` | Main menu + rules submenu (list-based) |
| `tui/chat.go` | Chat view (textarea + viewport + spinner) |
| `tui/batch.go` | Batch processing (progress + live log) |
| `main.go` | Trim entry point, keep all business logic |

---

## Phase 1: Foundation

### Task 1: Add dependencies

**Files:** go.mod, go.sum

- [ ] **Step 1: Install Charm packages**

```bash
cd D:/goproject/src/llm-util && go get github.com/charmbracelet/bubbletea github.com/charmbracelet/lipgloss github.com/charmbracelet/bubbles
```

- [ ] **Step 2: Build**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: add Charm TUI dependencies (bubbletea, lipgloss, bubbles)"
```

---

### Task 2: Create styles.go and tui.go (types + model skeleton)

**Files:**
- Create: `tui/styles.go`
- Create: `tui/tui.go`

- [ ] **Step 1: Create `tui/styles.go`**

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	Purple = lipgloss.Color("#7C3AED")
	Green  = lipgloss.Color("#10B981")
	Red    = lipgloss.Color("#EF4444")
	Yellow = lipgloss.Color("#F59E0B")
	Blue   = lipgloss.Color("#3B82F6")
	Gray   = lipgloss.Color("#6B7280")
	White  = lipgloss.Color("#F9FAFB")
)

var (
	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(Purple).Padding(0, 2).MarginBottom(1)
	HelpStyle  = lipgloss.NewStyle().Foreground(Gray).MarginTop(1)
)

var (
	SuccessStyle = lipgloss.NewStyle().Foreground(Green)
	ErrorStyle   = lipgloss.NewStyle().Foreground(Red)
	InfoStyle    = lipgloss.NewStyle().Foreground(Blue)
	WarnStyle    = lipgloss.NewStyle().Foreground(Yellow)
)

var PanelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(White).Background(Purple).Padding(0, 1)
```

- [ ] **Step 2: Create `tui/tui.go`**

```go
package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

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

type Message struct {
	Role    string
	Content string
}

// Callbacks set by main.go
type ChatFunc func(prompt string, history []Message) (string, error)
type ChatFileFunc func(prompt, filePath string) (string, error)

type ProgressMsg struct {
	Index    int
	Total    int
	Filename string
	Status   string // "processing", "done", "error", "skip"
}

// StartBatchFunc receives poolSize and a progress channel, runs batch, closes channel when done.
type StartBatchFunc func(poolSize int, progress chan<- ProgressMsg) error

type Model struct {
	view View

	apiKey  string
	appId   string
	history []Message

	mainMenu  list.Model
	rulesMenu list.Model

	OnSend        ChatFunc
	OnSendFile    ChatFileFunc
	OnRunCase     StartBatchFunc
	OnRunPDF      StartBatchFunc
	OnRunDIY      StartBatchFunc
	OnRunWorkflow StartBatchFunc

	chat  chatPanel
	batch batchPanel

	width  int
	height int
}

func NewModel(apiKey, appId string) Model {
	return Model{
		view:      ViewMainMenu,
		apiKey:    apiKey,
		appId:     appId,
		mainMenu:  buildMainMenu(),
		rulesMenu: buildRulesMenu(),
		chat:      newChatPanel(),
		batch:     newBatchPanel(),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	switch m.view {
	case ViewMainMenu:
		return m.updateMainMenu(msg)
	case ViewRulesMenu:
		return m.updateRulesMenu(msg)
	case ViewChat:
		return m.updateChat(msg)
	case ViewRuleCase, ViewRulePDF, ViewRuleDIY, ViewRuleWorkflow:
		return m.updateBatch(msg)
	}
	return m, nil
}

func (m Model) View() string {
	switch m.view {
	case ViewMainMenu:
		return m.mainMenuView()
	case ViewRulesMenu:
		return m.rulesMenuView()
	case ViewChat:
		return m.chatView()
	case ViewRuleCase, ViewRulePDF, ViewRuleDIY, ViewRuleWorkflow:
		return m.batchView()
	}
	return ""
}
```

- [ ] **Step 3: Verify compile (expected: missing function errors — added in next tasks)**

```bash
go build ./...
```
Expected: errors about undefined `chatPanel`, `batchPanel`, `buildMainMenu`, etc.

- [ ] **Step 4: Commit**

```bash
git add tui/
git commit -m "feat: add TUI model skeleton and Lip Gloss styles"
```

---

### Task 3: Create menus + refactor main.go entry point

**Files:**
- Create: `tui/menu.go`
- Modify: `main.go` (main function only)

- [ ] **Step 1: Create `tui/menu.go`**

```go
package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type menuItem struct{ title, desc string }

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(menuItem)
	if !ok {
		return
	}
	line := i.title
	if index == m.Index() {
		line = lipgloss.NewStyle().Foreground(White).Background(Purple).Padding(0, 1).Render("> " + line)
	} else {
		line = lipgloss.NewStyle().Padding(0, 1).Render("  " + line)
	}
	fmt.Fprint(w, line)
}

func buildMainMenu() list.Model {
	items := []list.Item{
		menuItem{title: "开始/继续对话", desc: "自由模式，与 AI 对话"},
		menuItem{title: "新对话", desc: "清空历史，开启新对话"},
		menuItem{title: "规则模式", desc: "批量处理：Excel、PDF、工作流"},
		menuItem{title: "退出", desc: "退出程序"},
	}
	l := list.New(items, itemDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	return l
}

func buildRulesMenu() list.Model {
	items := []list.Item{
		menuItem{title: "案例查询", desc: "从 data.xlsx 批量提问，结果写回 B 列"},
		menuItem{title: "PDF 批量提问", desc: "对 pdfs/ 目录所有 PDF 统一提问"},
		menuItem{title: "DIY 提问", desc: "n×m 规模：多问题 × 多文件"},
		menuItem{title: "工作流调用", desc: "自定义业务参数调用百炼应用"},
	}
	l := list.New(items, itemDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	return l
}

func (m Model) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "enter":
			switch m.mainMenu.Index() {
			case 0:
				m.view = ViewChat
				return m, nil
			case 1:
				m.history = nil
				m.view = ViewChat
				return m, nil
			case 2:
				m.view = ViewRulesMenu
				return m, nil
			case 3:
				return m, tea.Quit
			}
		}
	}
	var cmd tea.Cmd
	m.mainMenu, cmd = m.mainMenu.Update(msg)
	return m, cmd
}

func (m Model) mainMenuView() string {
	title := TitleStyle.Render("LLM Util — 百炼批量查询工具")
	menu := lipgloss.NewStyle().Padding(1).Render(m.mainMenu.View())
	help := HelpStyle.Render("↑/↓ 选择  enter 确认  q 退出")
	return lipgloss.JoinVertical(lipgloss.Center, title, menu, help)
}

func (m Model) updateRulesMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.view = ViewMainMenu
			return m, nil
		case "enter":
			switch m.rulesMenu.Index() {
			case 0:
				m.view = ViewRuleCase
			case 1:
				m.view = ViewRulePDF
			case 2:
				m.view = ViewRuleDIY
			case 3:
				m.view = ViewRuleWorkflow
			}
			m.batch.reset()
			return m, m.batch.startCmd()
		}
	}
	var cmd tea.Cmd
	m.rulesMenu, cmd = m.rulesMenu.Update(msg)
	return m, cmd
}

func (m Model) rulesMenuView() string {
	title := TitleStyle.Render("规则模式")
	menu := lipgloss.NewStyle().Padding(1).Render(m.rulesMenu.View())
	help := HelpStyle.Render("↑/↓ 选择  enter 确认  esc 返回")
	return lipgloss.JoinVertical(lipgloss.Center, title, menu, help)
}
```

- [ ] **Step 2: Rewrite `main()` in `main.go`**

Replace the old `main()` function (lines 50-106) with:

```go
func main() {
	_ = godotenv.Load()

	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}
	if appId == "" {
		appId = os.Getenv("LLM_APP_ID")
	}
	if apiKey == "" {
		fmt.Print("请输入API Key: ")
		fmt.Scanln(&apiKey)
	}
	if appId == "" {
		fmt.Print("请输入AppId: ")
		fmt.Scanln(&appId)
	}

	model := tui.NewModel(apiKey, appId)
	model.OnSend = func(prompt string, history []tui.Message) (string, error) {
		conversationHistory = nil
		for _, m := range history {
			conversationHistory = append(conversationHistory, Message{Role: m.Role, Content: m.Content})
		}
		return sendRequest(prompt)
	}
	model.OnSendFile = func(prompt, filePath string) (string, error) {
		return sendRequestWithFile(prompt, filePath)
	}
	model.OnRunCase = func(poolSize int, progress chan<- tui.ProgressMsg) error {
		return runCaseQueryRule(poolSize, progress)
	}
	model.OnRunPDF = func(poolSize int, progress chan<- tui.ProgressMsg) error {
		return runPdfBatchQuery(poolSize, "", progress)
	}
	model.OnRunDIY = func(poolSize int, progress chan<- tui.ProgressMsg) error {
		return runDIYQueryRule(poolSize, progress)
	}
	model.OnRunWorkflow = func(poolSize int, progress chan<- tui.ProgressMsg) error {
		return runWorkflowQueryRule(poolSize, progress)
	}

	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

Add imports to main.go:
```go
"llm-util/tui"
tea "github.com/charmbracelet/bubbletea"
```

Remove unused functions from main.go: `startConversation()` (lines 108-146), `resetConversation()` (lines 148-150), `runRuleMode()` (lines 284-344), `printResp()` (lines 721-728), `printQuestion()` (lines 730-736), `showLoading()` (lines 738-754).

Remove unused imports if any (`bufio` may no longer be needed in main — keep it if any remaining function uses it; `bufio` IS used by remaining batch functions for old console output).

- [ ] **Step 3: Add temporary stubs to `tui/tui.go`**

Before Task 4/6, add these stubs so the code compiles:

```go
// Temporary stubs — replaced in Phase 2/3
type chatPanel struct{}
func newChatPanel() chatPanel { return chatPanel{} }
func (m Model) updateChat(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m Model) chatView() string { return "Chat — coming soon\n\nPress esc to return." }

type batchPanel struct{}
func newBatchPanel() batchPanel { return batchPanel{} }
func (m Model) updateBatch(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m Model) batchView() string { return "Batch — coming soon\n\nPress esc to return." }
func (bp *batchPanel) reset() {}
func (bp *batchPanel) startCmd() tea.Cmd { return nil }
```

- [ ] **Step 4: Build**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add tui/menu.go main.go tui/tui.go
git commit -m "feat: add TUI main/rule menus, refactor main.go entry point"
```

---

## Phase 2: Chat View

### Task 4: Create chat view

**Files:**
- Create: `tui/chat.go`
- Modify: `tui/tui.go` (remove chatPanel stub)

- [ ] **Step 1: Create `tui/chat.go`**

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type chatPanel struct {
	viewport  viewport.Model
	textarea  textarea.Model
	spinner   spinner.Model
	loading   bool
	err       error
}

func newChatPanel() chatPanel {
	ta := textarea.New()
	ta.Placeholder = "输入您的问题... (Ctrl+Enter 发送)"
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	vp := viewport.New(40, 10)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = InfoStyle

	return chatPanel{
		textarea: ta,
		viewport: vp,
		spinner:  sp,
	}
}

type chatResponseMsg struct {
	text string
	err  error
}

func sendChatCmd(fn ChatFunc, prompt string, history []Message) tea.Cmd {
	return func() tea.Msg {
		resp, err := fn(prompt, history)
		return chatResponseMsg{text: resp, err: err}
	}
}

func (m Model) updateChat(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.view = ViewMainMenu
			return m, nil
		case "ctrl+n":
			m.history = nil
			m.chat.viewport.SetContent("")
			return m, nil
		case "enter":
			if m.chat.loading {
				return m, nil
			}
			input := strings.TrimSpace(m.chat.textarea.Value())
			if input == "" {
				return m, nil
			}
			m.chat.textarea.Reset()
			m.chat.loading = true
			m.chat.err = nil
			m.history = append(m.history, Message{Role: "user", Content: input})
			m.updateChatContent()
			return m, tea.Batch(m.chat.spinner.Tick, sendChatCmd(m.OnSend, input, m.history))
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.chat.spinner, cmd = m.chat.spinner.Update(msg)
		return m, cmd

	case chatResponseMsg:
		m.chat.loading = false
		if msg.err != nil {
			m.chat.err = msg.err
		} else {
			m.history = append(m.history, Message{Role: "assistant", Content: msg.text})
			m.chat.err = nil
		}
		m.updateChatContent()
		return m, nil
	}

	var cmd tea.Cmd
	m.chat.textarea, cmd = m.chat.textarea.Update(msg)
	return m, cmd
}

func (m *Model) updateChatContent() {
	var sb strings.Builder
	for _, msg := range m.history {
		switch msg.Role {
		case "user":
			sb.WriteString(lipgloss.NewStyle().Foreground(Blue).Bold(true).Render("🧑 您") + "\n")
		case "assistant":
			sb.WriteString(lipgloss.NewStyle().Foreground(Green).Bold(true).Render("🤖 助手") + "\n")
		}
		sb.WriteString(msg.Content + "\n\n")
	}
	m.chat.viewport.SetContent(sb.String())
	m.chat.viewport.GotoBottom()
}

func (m Model) chatView() string {
	help := "esc 返回  Ctrl+N 新对话  Enter 发送"
	if m.chat.loading {
		help = m.chat.spinner.View() + " 请求中..."
	}
	if m.chat.err != nil {
		help += "\n" + ErrorStyle.Render(fmt.Sprintf("❌ %v", m.chat.err))
	}

	title := PanelTitleStyle.Render("对话 — 自由模式")
	vpHeight := m.height - 10
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.chat.viewport.Width = m.width - 4
	m.chat.viewport.Height = vpHeight
	m.chat.textarea.SetWidth(m.width - 4)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.chat.viewport.View(),
		m.chat.textarea.View(),
		HelpStyle.Render(help),
	)
}
```

- [ ] **Step 2: Remove `chatPanel` stub from `tui/tui.go`**

Delete the `type chatPanel struct{}` line and the `newChatPanel`, `updateChat`, `chatView` stubs.

- [ ] **Step 3: Build**

```bash
go build ./...
```
Expected: no errors (batchPanel stub still in place).

- [ ] **Step 4: Commit**

```bash
git add tui/chat.go tui/tui.go
git commit -m "feat: add Bubble Tea chat view with textarea, viewport, spinner"
```

---

## Phase 3: Batch Views

### Task 5: Add progress channel to batch functions

**Files:** Modify: `main.go`

Each batch function gets a `progress chan<- tui.ProgressMsg` parameter. When non-nil, send progress; when nil, use old console output.

- [ ] **Step 1: Modify `runCaseQueryRule`**

Change signature (line 346):
```go
func runCaseQueryRule(poolSize int, progress chan<- tui.ProgressMsg) {
```

Remove input prompts (lines 347-357, the bufio.Reader poolSize prompt). Remove the `start := time.Now()` and file open logic — keep it, it's business logic.

Replace `printQuestion(question)` (line 409) with:
```go
if progress != nil {
    progress <- tui.ProgressMsg{Index: i, Total: len(rows), Filename: question, Status: "processing"}
} else {
    printQuestion(question)
}
```

After `file.SetCellValue(...)` (line 429), add:
```go
if progress != nil {
    progress <- tui.ProgressMsg{Index: i, Total: len(rows), Filename: question, Status: "done"}
}
```

Replace `fmt.Printf("请求失败: %v\n", err)` (line 420) with:
```go
if progress != nil {
    progress <- tui.ProgressMsg{Index: i, Total: len(rows), Filename: question, Status: "error"}
}
fmt.Printf("请求失败: %v\n", err)
```

Function returns `error` (always nil for now — batch functions don't return meaningful errors currently, but the signature allows it for future use). Add `return nil` at end.

- [ ] **Step 2: Modify `runPdfBatchQuery`**

Change signature (line 454):
```go
func runPdfBatchQuery(poolSize int, question string, progress chan<- tui.ProgressMsg) error {
```

Remove the `bufio.Reader` prompts for poolSize (lines 469-480) and question (lines 482-486).

Replace `console.Colorful` inside `pool.Submit` (lines 585-593) with progress channel calls to report status per file.

- [ ] **Step 3: Modify `runDIYQueryRule`**

Change signature (line 621):
```go
func runDIYQueryRule(poolSize int, progress chan<- tui.ProgressMsg) error {
```

Remove `bufio.Reader` poolSize prompt (lines 636-647). Replace console output with progress channel calls same pattern as above.

- [ ] **Step 4: Modify `runWorkflowQueryRule`**

Change signature (line 756):
```go
func runWorkflowQueryRule(poolSize int, progress chan<- tui.ProgressMsg) error {
```

Remove `bufio.Reader` poolSize prompt (lines 757-767). Replace `printQuestion(question)` and console output with progress channel calls.

- [ ] **Step 5: Add import for `tui` in main.go**

```go
"llm-util/tui"
```

- [ ] **Step 6: Build**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add main.go
git commit -m "feat: add progress channel to batch functions for TUI integration"
```

---

### Task 6: Create batch view

**Files:**
- Create: `tui/batch.go`
- Modify: `tui/tui.go` (remove batchPanel stub)

- [ ] **Step 1: Create `tui/batch.go`**

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type batchPanel struct {
	progress progress.Model
	logs     []batchLogEntry
	total    int
	done     int
	errors   int
	running  bool
	poolSize int
	ruleName string
	ch       chan ProgressMsg
}

type batchLogEntry struct {
	idx    int
	name   string
	status string
}

func newBatchPanel() batchPanel {
	p := progress.New(progress.WithDefaultGradient())
	return batchPanel{progress: p, poolSize: 10}
}

func (bp *batchPanel) reset() {
	bp.logs = nil
	bp.total = 0
	bp.done = 0
	bp.errors = 0
	bp.running = false
	bp.poolSize = 10
}

type batchStartMsg struct{}

func (bp *batchPanel) startCmd() tea.Cmd {
	return func() tea.Msg { return batchStartMsg{} }
}

type batchProgressMsg ProgressMsg

type batchDoneMsg struct{}

func listenProgress(ch <-chan ProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return batchDoneMsg{}
		}
		return batchProgressMsg(msg)
	}
}

func (m Model) updateBatch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			if !m.batch.running {
				m.view = ViewRulesMenu
				return m, nil
			}
		}

	case batchStartMsg:
		m.batch.running = true
		m.batch.ruleName = ruleName(m.view)
		m.batch.poolSize = 10

		var sbf StartBatchFunc
		switch m.view {
		case ViewRuleCase:
			sbf = m.OnRunCase
		case ViewRulePDF:
			sbf = m.OnRunPDF
		case ViewRuleDIY:
			sbf = m.OnRunDIY
		case ViewRuleWorkflow:
			sbf = m.OnRunWorkflow
		}
		if sbf == nil {
			m.batch.running = false
			return m, nil
		}
		m.batch.ch = make(chan ProgressMsg, 200)
		go func() {
			_ = sbf(m.batch.poolSize, m.batch.ch)
			close(m.batch.ch)
		}()
		return m, listenProgress(m.batch.ch)

	case batchProgressMsg:
		m.batch.total = msg.Total
		switch msg.Status {
		case "done":
			m.batch.done++
		case "error":
			m.batch.errors++
		}
		m.batch.logs = append(m.batch.logs, batchLogEntry{
			idx: msg.Index, name: msg.Filename, status: msg.Status,
		})
		if len(m.batch.logs) > 200 {
			m.batch.logs = m.batch.logs[len(m.batch.logs)-200:]
		}
		return m, listenProgress(m.batch.ch)

	case batchDoneMsg:
		m.batch.running = false
		return m, nil
	}
	return m, nil
}

func ruleName(v View) string {
	switch v {
	case ViewRuleCase:
		return "规则1 · 案例查询"
	case ViewRulePDF:
		return "规则2 · PDF 批量提问"
	case ViewRuleDIY:
		return "规则3 · DIY 提问"
	case ViewRuleWorkflow:
		return "规则4 · 工作流调用"
	}
	return "批量处理"
}

func (m Model) batchView() string {
	title := PanelTitleStyle.Render(m.batch.ruleName)
	var body strings.Builder

	body.WriteString(fmt.Sprintf("⚡ 并发: %d  ✅ 完成: %d  ❌ 失败: %d",
		m.batch.poolSize, m.batch.done, m.batch.errors,
	))
	if m.batch.total > 0 {
		body.WriteString(fmt.Sprintf("  📊 总计: %d", m.batch.total))
	}
	body.WriteString("\n\n")

	if m.batch.total > 0 {
		ratio := float64(m.batch.done+m.batch.errors) / float64(m.batch.total)
		body.WriteString(m.batch.progress.ViewAs(ratio) + "\n\n")
	}

	// Recent logs (last 40)
	start := len(m.batch.logs) - 40
	if start < 0 {
		start = 0
	}
	for _, e := range m.batch.logs[start:] {
		var icon string
		var style lipgloss.Style
		switch e.status {
		case "done":
			icon, style = "✅", SuccessStyle
		case "error":
			icon, style = "❌", ErrorStyle
		case "processing":
			icon, style = "🔄", InfoStyle
		case "skip":
			icon, style = "⏭️", lipgloss.NewStyle().Foreground(Gray)
		}
		line := fmt.Sprintf("%s [%d] %s", icon, e.idx, e.name)
		if style.GetForeground() != lipgloss.NoColor {
			line = style.Render(line)
		}
		body.WriteString(line + "\n")
	}

	if !m.batch.running {
		body.WriteString("\n")
		if m.batch.errors == 0 {
			body.WriteString(SuccessStyle.Render("🎉 所有请求成功完成！"))
		} else {
			body.WriteString(ErrorStyle.Render(fmt.Sprintf("⚠️ 失败: %d", m.batch.errors)))
		}
	}

	help := HelpStyle.Render("按 q 返回规则菜单")
	if m.batch.running {
		help = HelpStyle.Render("处理中，请等待...")
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, body.String(), help)
}
```

- [ ] **Step 2: Remove `batchPanel` stub from `tui/tui.go`**

Delete `type batchPanel struct{}`, `newBatchPanel`, `updateBatch`, `batchView`, `reset`, `startCmd` stubs.

- [ ] **Step 3: Build**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add tui/batch.go tui/tui.go
git commit -m "feat: add batch processing view with progress bar and live log"
```

---

## Task 7: Final cleanup and verification

- [ ] **Step 1: Run full build and vet**

```bash
go build ./...
go vet ./...
```
Expected: no errors.

- [ ] **Step 2: Run existing tests**

```bash
go test ./...
```
Expected: all tests pass.

- [ ] **Step 3: Remove unused ANSI/console code from main.go**

Remove the old ANSI color constants (lines 41-48) and `console`/`constant` imports if they're no longer used. If batch functions still reference them for the nil-progress fallback path, keep them.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: final cleanup, remove unused code"
```

---

## Verification Checklist

1. `go build ./...` — passes
2. `go vet ./...` — passes
3. `go test ./...` — passes
4. Manual: run binary → main menu renders with styled list, ↑↓/enter/q work
5. Manual: chat → type prompt → see spinner → see response in viewport → scroll history
6. Manual: rules menu → select each rule → batch starts → progress bar moves → logs update → finished message appears
7. PDF rule known gap: question defaults to "" (improve in follow-up)
