package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
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
	skipped     int
	running     bool
	poolSize    int
	ruleName    string
	ch          chan ProgressMsg
	configuring bool
	cfgTextarea textarea.Model
	filePicker  bool
	fileList    list.Model
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

	fl := list.New([]list.Item{}, itemDelegate{}, 0, 10)
	fl.SetShowTitle(false)
	fl.SetShowStatusBar(false)
	fl.SetFilteringEnabled(false)
	fl.SetShowHelp(false)
	fl.SetShowPagination(false)
	fl.DisableQuitKeybindings()
	fl.KeyMap.NextPage.SetEnabled(false)
	fl.KeyMap.PrevPage.SetEnabled(false)

	return batchPanel{progress: p, poolSize: 10, cfgTextarea: ta, fileList: fl}
}

func (bp *batchPanel) reset() {
	bp.logs = nil
	bp.total = 0
	bp.done = 0
	bp.errors = 0
	bp.skipped = 0
	bp.running = false
	bp.poolSize = 10
	bp.configuring = false
	bp.filePicker = false
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
				m.batch.ch = make(chan ProgressMsg, 200)
				filename := sel.title
				go func() {
					defer close(m.batch.ch)
					_ = m.OnRunModeA(m.batch.poolSize, filename, m.batch.ch)
				}()
				return m, listenProgress(m.batch.ch)
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

	if m.batch.configuring {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				input := strings.TrimSpace(m.batch.cfgTextarea.Value())
				if input == "" {
					return m, nil
				}
				m.batch.configuring = false
				m.batch.running = true
				m.batch.ch = make(chan ProgressMsg, 200)
				prompt := input
				go func() {
					defer close(m.batch.ch)
					_ = m.OnRunPDF(m.batch.poolSize, prompt, m.batch.ch)
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

		if m.view == ViewModeA {
			items := scanXlsxFiles()
			if len(items) == 0 {
				return m, func() tea.Msg { return showTipMsg("当前目录没有 .xlsx 文件") }
			}
			m.batch.fileList.SetItems(items)
			m.batch.fileList.SetSize(m.width-4, len(items)*2)
			m.batch.filePicker = true
			return m, nil
		}

		if m.view == ViewRulePDF {
			m.batch.configuring = true
			return m, m.batch.cfgTextarea.Focus()
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
		return m, listenProgress(m.batch.ch)

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
		return m, listenProgress(m.batch.ch)

	case batchDoneMsg:
		m.batch.running = false
		return m, nil
	}
	return m, nil
}

func ruleName(v View) string {
	switch v {
	case ViewModeA:
		return "模式A"
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
	if m.batch.filePicker {
		title := PanelTitleStyle.Render(m.batch.ruleName)
		prompt := lipgloss.NewStyle().Foreground(Blue).Render("请选择要处理的 Excel 文件：")
		m.batch.fileList.SetHeight(len(m.batch.fileList.Items()) * 2)
		body := lipgloss.NewStyle().Padding(1, 2).Render(m.batch.fileList.View())
		help := HelpStyle.Render("↑/↓ 选择  enter 确认  esc 返回")
		return lipgloss.JoinVertical(lipgloss.Left, title, prompt, body, help)
	}

	if m.batch.configuring {
		title := PanelTitleStyle.Render(m.batch.ruleName)
		promptText := "请输入要提问的问题："
		prompt := lipgloss.NewStyle().Foreground(Blue).Render(promptText)
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

	completed := m.batch.done + m.batch.errors + m.batch.skipped
	body.WriteString(fmt.Sprintf("并发: %d  总计: %d  已处理: %d",
		m.batch.poolSize, m.batch.total, completed,
	))
	body.WriteString("\n")
	body.WriteString(SuccessStyle.Render(fmt.Sprintf("  ✅ 成功 %d", m.batch.done)))
	body.WriteString("  ")
	body.WriteString(ErrorStyle.Render(fmt.Sprintf("❌ 失败 %d", m.batch.errors)))
	if m.batch.skipped > 0 {
		body.WriteString("  ")
		body.WriteString(lipgloss.NewStyle().Foreground(Dim).Render(fmt.Sprintf("⏭️ 跳过 %d", m.batch.skipped)))
	}
	body.WriteString("\n\n")

	if m.batch.total > 0 {
		ratio := float64(completed) / float64(m.batch.total)
		if ratio > 1 {
			ratio = 1
		}
		body.WriteString(m.batch.progress.ViewAs(ratio) + "\n\n")
	}

	if !m.batch.running {
		if m.batch.done == 0 && m.batch.errors == 0 && m.batch.skipped > 0 {
			body.WriteString(WarnStyle.Render(fmt.Sprintf("所有 %d 条均已处理过，无需重复运行", m.batch.skipped)))
		} else if m.batch.errors == 0 {
			body.WriteString(SuccessStyle.Render("🎉 所有请求成功完成！"))
		} else {
			body.WriteString(ErrorStyle.Render(fmt.Sprintf("⚠️ 失败: %d", m.batch.errors)))
		}
	}

	help := HelpStyle.Render("按 q 返回")
	if m.batch.running {
		help = HelpStyle.Render("处理中，请等待...")
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, body.String(), help)
}
