package tui

import (
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type settingsPanel struct {
	apiKeyInput   textinput.Model
	appIdInput    textinput.Model
	poolSizeInput textinput.Model
	focusIndex    int
	saved         bool
}

func newSettingsPanel(apiKey, appId string, poolSize int) settingsPanel {
	ak := textinput.New()
	ak.Placeholder = "输入 API Key..."
	ak.SetValue(apiKey)
	ak.EchoMode = textinput.EchoPassword
	ak.EchoCharacter = '*'
	ak.Width = 60

	aid := textinput.New()
	aid.Placeholder = "输入 AppId..."
	aid.SetValue(appId)
	aid.Width = 60

	ps := textinput.New()
	ps.Placeholder = "输入 并发数..."
	ps.SetValue(strconv.Itoa(poolSize))
	ps.Width = 10
	ps.CharLimit = 3

	ak.Focus()

	return settingsPanel{
		apiKeyInput:   ak,
		appIdInput:    aid,
		poolSizeInput: ps,
		focusIndex:    0,
	}
}

func (sp *settingsPanel) reset(apiKey, appId string, poolSize int) {
	sp.apiKeyInput.SetValue(apiKey)
	sp.appIdInput.SetValue(appId)
	sp.poolSizeInput.SetValue(strconv.Itoa(poolSize))
	sp.saved = false
}

func (m Model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.view = ViewMainMenu
			m.settings.saved = false
			return m, nil
		case "tab":
			m.settings.focusIndex = (m.settings.focusIndex + 1) % 3
			return m, m.settings.focusInput()
		case "shift+tab":
			m.settings.focusIndex = (m.settings.focusIndex - 1 + 3) % 3
			return m, m.settings.focusInput()
		case "enter":
			key := m.settings.apiKeyInput.Value()
			id := m.settings.appIdInput.Value()
			ps, _ := strconv.Atoi(m.settings.poolSizeInput.Value())
			if ps <= 0 {
				ps = 10
			} else if ps > 200 {
				ps = 200
			}
			if m.OnSaveSettings != nil {
				_ = m.OnSaveSettings(key, id, ps)
			}
			m.apiKey = key
			m.appId = id
			m.poolSize = ps
			m.settings.saved = true
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.settings.focusIndex {
	case 0:
		m.settings.apiKeyInput, cmd = m.settings.apiKeyInput.Update(msg)
	case 1:
		m.settings.appIdInput, cmd = m.settings.appIdInput.Update(msg)
	case 2:
		m.settings.poolSizeInput, cmd = m.settings.poolSizeInput.Update(msg)
	}
	return m, cmd
}

func (sp *settingsPanel) focusInput() tea.Cmd {
	sp.apiKeyInput.Blur()
	sp.appIdInput.Blur()
	sp.poolSizeInput.Blur()
	switch sp.focusIndex {
	case 0:
		return sp.apiKeyInput.Focus()
	case 1:
		return sp.appIdInput.Focus()
	case 2:
		return sp.poolSizeInput.Focus()
	}
	return nil
}

func (m Model) settingsView() string {
	title := PanelTitleStyle.Render("配置管理")

	labelStyle := lipgloss.NewStyle().Foreground(Blue).Bold(true)

	keyLabel := labelStyle.Render("API Key:")
	appLabel := labelStyle.Render("AppId:")
	poolLabel := labelStyle.Render("并发数:")

	if m.settings.saved {
		title += " " + SuccessStyle.Render("✅ 已保存")
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		keyLabel,
		m.settings.apiKeyInput.View(),
		"",
		appLabel,
		m.settings.appIdInput.View(),
		"",
		poolLabel,
		m.settings.poolSizeInput.View(),
	)

	help := HelpStyle.Render("enter 保存  tab 切换  esc 返回")
	return lipgloss.JoinVertical(lipgloss.Left, title, body, help)
}
