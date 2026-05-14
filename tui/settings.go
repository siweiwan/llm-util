package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type settingsPanel struct {
	apiKeyInput textinput.Model
	appIdInput  textinput.Model
	focusIndex  int
	saved       bool
}

func newSettingsPanel(apiKey, appId string) settingsPanel {
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

	ak.Focus()

	return settingsPanel{
		apiKeyInput: ak,
		appIdInput:  aid,
		focusIndex:  0,
	}
}

func (sp *settingsPanel) reset(apiKey, appId string) {
	sp.apiKeyInput.SetValue(apiKey)
	sp.appIdInput.SetValue(appId)
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
			m.settings.focusIndex = (m.settings.focusIndex + 1) % 2
			return m, m.settings.focusInput()
		case "shift+tab":
			m.settings.focusIndex = (m.settings.focusIndex - 1 + 2) % 2
			return m, m.settings.focusInput()
		case "enter":
			key := m.settings.apiKeyInput.Value()
			id := m.settings.appIdInput.Value()
			if m.OnSaveSettings != nil {
				_ = m.OnSaveSettings(key, id)
			}
			m.apiKey = key
			m.appId = id
			m.settings.saved = true
			return m, nil
		}
	}

	var cmd tea.Cmd
	if m.settings.focusIndex == 0 {
		m.settings.apiKeyInput, cmd = m.settings.apiKeyInput.Update(msg)
	} else {
		m.settings.appIdInput, cmd = m.settings.appIdInput.Update(msg)
	}
	return m, cmd
}

func (sp *settingsPanel) focusInput() tea.Cmd {
	if sp.focusIndex == 0 {
		sp.apiKeyInput.Focus()
		sp.appIdInput.Blur()
		return sp.apiKeyInput.Focus()
	}
	sp.apiKeyInput.Blur()
	sp.appIdInput.Focus()
	return sp.appIdInput.Focus()
}

func (m Model) settingsView() string {
	title := PanelTitleStyle.Render("配置管理")

	labelStyle := lipgloss.NewStyle().Foreground(Blue).Bold(true)

	keyLabel := labelStyle.Render("API Key:")
	appLabel := labelStyle.Render("AppId:")

	if m.settings.saved {
		title += " " + SuccessStyle.Render("✅ 已保存")
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		keyLabel,
		m.settings.apiKeyInput.View(),
		"",
		appLabel,
		m.settings.appIdInput.View(),
	)

	help := HelpStyle.Render("enter 保存  tab 切换  esc 返回")
	return lipgloss.JoinVertical(lipgloss.Left, title, body, help)
}
