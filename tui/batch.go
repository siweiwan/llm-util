package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type batchPanel struct {
	progress    progress.Model
	logs        []batchLogEntry
	total       int
	done        int
	errors      int
	running     bool
	poolSize    int
	ruleName    string
	ch          chan ProgressMsg
	configuring bool
	cfgTextarea textarea.Model
}

type batchLogEntry struct {
	idx    int
	name   string
	status string
}

func newBatchPanel() batchPanel {
	p := progress.New(progress.WithDefaultGradient())
	ta := textarea.New()
	ta.Placeholder = "输入要提问的问题..."
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.CharLimit = 4000
	return batchPanel{progress: p, poolSize: 10, cfgTextarea: ta}
}

func (bp *batchPanel) reset() {
	bp.logs = nil
	bp.total = 0
	bp.done = 0
	bp.errors = 0
	bp.running = false
	bp.poolSize = 10
	bp.configuring = false
	bp.cfgTextarea.Reset()
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
	if m.batch.configuring {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				question := strings.TrimSpace(m.batch.cfgTextarea.Value())
				if question == "" {
					return m, nil
				}
				m.batch.configuring = false
				m.batch.running = true
				m.batch.ch = make(chan ProgressMsg, 200)
				go func() {
					defer close(m.batch.ch)
					_ = m.OnRunPDF(m.batch.poolSize, question, m.batch.ch)
				}()
				return m, listenProgress(m.batch.ch)
			case "esc":
				m.batch.configuring = false
				m.view = ViewRulesMenu
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.batch.cfgTextarea, cmd = m.batch.cfgTextarea.Update(msg)
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

		if m.view == ViewRulePDF {
			m.batch.configuring = true
			return m, m.batch.cfgTextarea.Focus()
		}

		m.batch.running = true
		var sbf StartBatchFunc
		switch m.view {
		case ViewRuleCase:
			sbf = m.OnRunCase
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
	if m.batch.configuring {
		title := PanelTitleStyle.Render(m.batch.ruleName)
		prompt := lipgloss.NewStyle().Foreground(Blue).Render("请输入要提问的问题：")
		m.batch.cfgTextarea.SetWidth(m.width - 4)
		help := HelpStyle.Render("enter 确认  esc 返回")
		return lipgloss.JoinVertical(lipgloss.Left,
			title,
			prompt,
			m.batch.cfgTextarea.View(),
			help,
		)
	}

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
			icon, style = "⏭️", lipgloss.NewStyle().Foreground(Dim)
		}
		line := fmt.Sprintf("%s [%d] %s", icon, e.idx, e.name)
		if style.GetForeground() != lipgloss.Color("") {
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
