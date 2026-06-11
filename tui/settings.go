package tui

import (
	"llm-util/conf"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const settingsFieldCount = 6

type settingsPanel struct {
	apiKeyInput      textinput.Model
	appIdInput       textinput.Model
	workspaceIdInput textinput.Model
	poolSizeInput    textinput.Model
	akIdInput        textinput.Model
	akSecretInput    textinput.Model
	focusIndex       int
	saved            bool
}

func newSettingsPanel(cfg *conf.Config) settingsPanel {
	ak := textinput.New()
	ak.Placeholder = "输入 API Key..."
	ak.SetValue(cfg.APIKey)
	ak.EchoMode = textinput.EchoPassword
	ak.EchoCharacter = '*'
	ak.Width = 60

	aid := textinput.New()
	aid.Placeholder = "输入 AppId..."
	aid.SetValue(cfg.AppID)
	aid.Width = 60

	ws := textinput.New()
	ws.Placeholder = "输入 Workspace ID..."
	ws.SetValue(cfg.WorkspaceID)
	ws.Width = 60

	ps := textinput.New()
	ps.Placeholder = "输入 并发数..."
	ps.SetValue(strconv.Itoa(cfg.PoolSize))
	ps.Width = 10
	ps.CharLimit = 3

	akId := textinput.New()
	akId.Placeholder = "输入 AccessKey ID..."
	akId.SetValue(cfg.AccessKeyId)
	akId.Width = 60

	akSecret := textinput.New()
	akSecret.Placeholder = "输入 AccessKey Secret..."
	akSecret.SetValue(cfg.AccessKeySecret)
	akSecret.EchoMode = textinput.EchoPassword
	akSecret.EchoCharacter = '*'
	akSecret.Width = 60

	ak.Focus()

	return settingsPanel{
		apiKeyInput:      ak,
		appIdInput:       aid,
		workspaceIdInput: ws,
		poolSizeInput:    ps,
		akIdInput:        akId,
		akSecretInput:    akSecret,
		focusIndex:       0,
	}
}

func (sp *settingsPanel) reset(cfg *conf.Config) {
	sp.apiKeyInput.SetValue(cfg.APIKey)
	sp.appIdInput.SetValue(cfg.AppID)
	sp.workspaceIdInput.SetValue(cfg.WorkspaceID)
	sp.poolSizeInput.SetValue(strconv.Itoa(cfg.PoolSize))
	sp.akIdInput.SetValue(cfg.AccessKeyId)
	sp.akSecretInput.SetValue(cfg.AccessKeySecret)
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
			m.settings.focusIndex = (m.settings.focusIndex + 1) % settingsFieldCount
			return m, m.settings.focusInput()
		case "shift+tab":
			m.settings.focusIndex = (m.settings.focusIndex - 1 + settingsFieldCount) % settingsFieldCount
			return m, m.settings.focusInput()
		case "enter":
			m.cfg.APIKey = m.settings.apiKeyInput.Value()
			m.cfg.AppID = m.settings.appIdInput.Value()
			m.cfg.WorkspaceID = m.settings.workspaceIdInput.Value()
			ps, _ := strconv.Atoi(m.settings.poolSizeInput.Value())
			if ps <= 0 {
				ps = 10
			} else if ps > 200 {
				ps = 200
			}
			m.cfg.PoolSize = ps
			m.cfg.AccessKeyId = m.settings.akIdInput.Value()
			m.cfg.AccessKeySecret = m.settings.akSecretInput.Value()
			if m.OnSaveSettings != nil {
				_ = m.OnSaveSettings(m.cfg)
			}
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
		m.settings.workspaceIdInput, cmd = m.settings.workspaceIdInput.Update(msg)
	case 3:
		m.settings.poolSizeInput, cmd = m.settings.poolSizeInput.Update(msg)
	case 4:
		m.settings.akIdInput, cmd = m.settings.akIdInput.Update(msg)
	case 5:
		m.settings.akSecretInput, cmd = m.settings.akSecretInput.Update(msg)
	}
	return m, cmd
}

func (sp *settingsPanel) focusInput() tea.Cmd {
	sp.apiKeyInput.Blur()
	sp.appIdInput.Blur()
	sp.workspaceIdInput.Blur()
	sp.poolSizeInput.Blur()
	sp.akIdInput.Blur()
	sp.akSecretInput.Blur()
	switch sp.focusIndex {
	case 0:
		return sp.apiKeyInput.Focus()
	case 1:
		return sp.appIdInput.Focus()
	case 2:
		return sp.workspaceIdInput.Focus()
	case 3:
		return sp.poolSizeInput.Focus()
	case 4:
		return sp.akIdInput.Focus()
	case 5:
		return sp.akSecretInput.Focus()
	}
	return nil
}

func (m Model) settingsView() string {
	title := PanelTitleStyle.Render("配置管理")

	labelStyle := lipgloss.NewStyle().Foreground(Blue).Bold(true)

	keyLabel := labelStyle.Render("API Key:")
	appLabel := labelStyle.Render("AppId:")
	wsLabel := labelStyle.Render("Workspace ID:")
	poolLabel := labelStyle.Render("并发数:")
	akIdLabel := labelStyle.Render("AccessKey ID:")
	akSecretLabel := labelStyle.Render("AccessKey Secret:")

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
		wsLabel,
		m.settings.workspaceIdInput.View(),
		"",
		poolLabel,
		m.settings.poolSizeInput.View(),
		"",
		akIdLabel,
		m.settings.akIdInput.View(),
		"",
		akSecretLabel,
		m.settings.akSecretInput.View(),
	)

	help := HelpStyle.Render("enter 保存  tab 切换  esc 返回")
	return lipgloss.JoinVertical(lipgloss.Left, title, body, help)
}
