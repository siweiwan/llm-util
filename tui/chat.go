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
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model
	loading  bool
	err      error
}

func newChatPanel() chatPanel {
	ta := textarea.New()
	ta.Placeholder = "输入您的问题... (Enter 发送)"
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.CharLimit = 4000

	vp := viewport.New(40, 10)
	vp.Style = lipgloss.NewStyle().Padding(0, 1)

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

func (cp *chatPanel) Focus() tea.Cmd {
	return cp.textarea.Focus()
}

func (cp *chatPanel) Blur() {
	cp.textarea.Blur()
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
			return m, tea.Batch(
				m.chat.spinner.Tick,
				sendChatCmd(m.OnSend, input, m.history),
			)
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

	if !m.chat.loading {
		var cmd tea.Cmd
		m.chat.textarea, cmd = m.chat.textarea.Update(msg)
		return m, cmd
	}
	return m, nil
}

var (
	bulletStyle   = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	respBarStyle  = lipgloss.NewStyle().Foreground(Accent)
	respTextStyle = lipgloss.NewStyle().Padding(0, 0, 0, 2)
)

func (m *Model) updateChatContent() {
	var sb strings.Builder
	for _, msg := range m.history {
		switch msg.Role {
		case "user":
			sb.WriteString(bulletStyle.Render("●") + " " + msg.Content + "\n")
		case "assistant":
			sb.WriteString(respBarStyle.Render("│") + respTextStyle.Render(msg.Content) + "\n")
		}
	}
	m.chat.viewport.SetContent(sb.String())
	m.chat.viewport.GotoBottom()
}

func (m Model) chatView() string {
	help := "esc 返回  Ctrl+N 新对话  Enter 发送"
	if m.chat.loading {
		help = m.chat.spinner.View() + " 正在生成..."
	}
	if m.chat.err != nil {
		help += "\n" + ErrorStyle.Render(fmt.Sprintf("✗ %v", m.chat.err))
	}

	title := PanelTitleStyle.Render("对话")
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
