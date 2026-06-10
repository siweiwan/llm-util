package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type batchPanel struct {
	progress     progress.Model
	logs         []batchLogEntry
	total        int
	done         int
	errors       int
	skipped      int
	running      bool
	poolSize     int
	ruleName     string
	filename     string
	ch           chan ProgressMsg
	filePicker   bool
	fileList     list.Model
	tickID       int // 动画标识，防止过期 tick 消息继续触发
	spinnerFrame int // 旋转动画帧
}

type batchLogEntry struct {
	idx    int
	name   string
	status string
}

func newBatchPanel() batchPanel {
	p := progress.New(
		progress.WithGradient("#06B6D4", "#22C55E"),
		progress.WithoutPercentage(),
	)

	fl := list.New([]list.Item{}, itemDelegate{}, 0, 10)
	fl.SetShowTitle(false)
	fl.SetShowStatusBar(false)
	fl.SetFilteringEnabled(false)
	fl.SetShowHelp(false)
	fl.SetShowPagination(false)
	fl.DisableQuitKeybindings()
	fl.KeyMap.NextPage.SetEnabled(false)
	fl.KeyMap.PrevPage.SetEnabled(false)

	return batchPanel{progress: p, poolSize: 4, fileList: fl}
}

func (bp *batchPanel) reset() {
	bp.logs = nil
	bp.total = 0
	bp.done = 0
	bp.errors = 0
	bp.skipped = 0
	bp.running = false
	bp.filename = ""
	bp.poolSize = 4
	bp.filePicker = false
	bp.tickID++
}

// ratio 计算当前完成比例
func (bp *batchPanel) ratio() float64 {
	if bp.total <= 0 {
		return 0
	}
	completed := bp.done + bp.errors + bp.skipped
	r := float64(completed) / float64(bp.total)
	if r > 1 {
		r = 1
	}
	return r
}

// progressTickMsg 驱动进度条流动动画
type progressTickMsg struct {
	id int
}

var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

func tickCmd(id int) tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(_ time.Time) tea.Msg {
		return progressTickMsg{id: id}
	})
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

func scanXlsxFiles() []list.Item {
	var items []list.Item
	entries, _ := os.ReadDir(".")
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".xlsx") {
			items = append(items, menuItem{title: e.Name(), desc: ""})
		}
	}
	return items
}

func (m Model) updateBatch(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.batch.filePicker {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				if len(m.batch.fileList.Items()) == 0 {
					return m, nil
				}
				sel := m.batch.fileList.SelectedItem().(menuItem)
				m.batch.filePicker = false
				m.batch.running = true
				m.batch.filename = sel.title
				m.batch.ch = make(chan ProgressMsg, 200)
				go func() {
					defer close(m.batch.ch)
					switch m.view {
					case ViewRuleFile:
						_ = m.OnRunModeB(m.batch.poolSize, m.batch.filename, m.batch.ch)
					default:
						_ = m.OnRunModeA(m.batch.poolSize, m.batch.filename, m.batch.ch)
					}
				}()
				return m, tea.Batch(listenProgress(m.batch.ch), tickCmd(m.batch.tickID))
			case "esc":
				m.batch.filePicker = false
				m.view = ViewRulesMenu
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.batch.fileList, cmd = m.batch.fileList.Update(msg)
		return m, cmd
	}

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
		m.batch.ruleName = ruleName(m.view)

		// Mode A and Mode B both use file picker
		if m.view == ViewModeA || m.view == ViewRuleFile {
			items := scanXlsxFiles()
			if len(items) == 0 {
				return m, func() tea.Msg { return showTipMsg("当前目录没有 .xlsx 文件") }
			}
			m.batch.fileList.SetItems(items)
			m.batch.fileList.SetSize(m.width-4, len(items)*2)
			m.batch.filePicker = true
			return m, nil
		}

		m.batch.running = true
		var sbf StartBatchFunc
		switch m.view {
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
			defer close(m.batch.ch)
			_ = sbf(m.batch.poolSize, m.batch.ch)
		}()
		return m, tea.Batch(listenProgress(m.batch.ch), tickCmd(m.batch.tickID))

	case progressTickMsg:
		if msg.id == m.batch.tickID && m.batch.running {
			cmd := m.batch.progress.SetPercent(m.batch.ratio())
			m.batch.spinnerFrame = (m.batch.spinnerFrame + 1) % len(spinnerFrames)
			return m, tea.Batch(cmd, tickCmd(msg.id))
		}
		return m, nil

	case progress.FrameMsg:
		progressModel, cmd := m.batch.progress.Update(msg)
		m.batch.progress = progressModel.(progress.Model)
		return m, cmd

	case batchProgressMsg:
		m.batch.total = msg.Total
		switch msg.Status {
		case "done":
			m.batch.done++
		case "error":
			m.batch.errors++
		case "skip":
			m.batch.skipped++
		}
		m.batch.logs = append(m.batch.logs, batchLogEntry{
			idx: msg.Index, name: msg.Filename, status: msg.Status,
		})
		if len(m.batch.logs) > 200 {
			m.batch.logs = m.batch.logs[len(m.batch.logs)-200:]
		}
		// 立即更新进度条，不等待定时器
		cmd := m.batch.progress.SetPercent(m.batch.ratio())
		return m, tea.Batch(cmd, listenProgress(m.batch.ch))

	case batchDoneMsg:
		m.batch.running = false
		// 确保进度条显示完成状态
		cmd := m.batch.progress.SetPercent(1.0)
		return m, cmd
	}
	return m, nil
}

func ruleName(v View) string {
	switch v {
	case ViewModeA:
		return "模式A"
	case ViewRuleFile:
		return "模式B"
	case ViewRuleDIY:
		return "DIY 提问"
	case ViewRuleWorkflow:
		return "工作流调用"
	}
	return "批量处理"
}

func (m Model) batchView() string {
	if m.batch.filePicker {
		title := PanelTitleStyle.Render(m.batch.ruleName)
		prompt := lipgloss.NewStyle().Foreground(Blue).Render("请选择要处理的 Excel 文件：")
		m.batch.fileList.SetHeight(len(m.batch.fileList.Items()) * 2)
		body := lipgloss.NewStyle().Padding(1, 2).Render(m.batch.fileList.View())
		help := HelpStyle.Render("↑/↓ 选择  enter 确认  esc 返回")
		return lipgloss.JoinVertical(lipgloss.Left, title, prompt, body, help)
	}

	title := PanelTitleStyle.Render(m.batch.ruleName)
	var body strings.Builder

	fmt.Fprintf(&body, "⚡ 并发 %d  📊 总计 %d  %s  %s  %s",
		m.batch.poolSize, m.batch.total,
		SuccessStyle.Render(fmt.Sprintf("✅ 成功 %d", m.batch.done)),
		ErrorStyle.Render(fmt.Sprintf("❌ 失败 %d", m.batch.errors)),
		lipgloss.NewStyle().Foreground(Dim).Render(fmt.Sprintf("⏭️ 跳过 %d", m.batch.skipped)),
	)
	body.WriteString("\n\n")

	if m.batch.total > 0 {
		body.WriteString(m.batch.progress.View() + "\n\n")
	}

	if !m.batch.running {
		body.WriteString(SuccessStyle.Render("✅ 处理完成"))
		if m.batch.filename != "" {
			body.WriteString("  " + lipgloss.NewStyle().Foreground(Dim).Render("→ "+m.batch.filename))
		}
		body.WriteString("\n")
	}

	help := HelpStyle.Render("按 q 返回")
	if m.batch.running {
		spinner := string(spinnerFrames[m.batch.spinnerFrame])
		help = HelpStyle.Render(spinner + " 处理中，请等待...")
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, body.String(), help)
}
