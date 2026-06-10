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

func (d itemDelegate) Height() int                             { return 2 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(menuItem)
	if !ok {
		return
	}
	if index == m.Index() {
		fmt.Fprint(w, SelectedItemStyle.Render("> "+i.title)+"\n")
		fmt.Fprint(w, SelectedDescStyle.Render("  "+i.desc))
	} else {
		fmt.Fprint(w, NormalItemStyle.Render("  "+i.title)+"\n")
		fmt.Fprint(w, NormalDescStyle.Render("  "+i.desc))
	}
}

func buildMainMenu() list.Model {
	items := []list.Item{
		menuItem{title: "配置管理", desc: "设置 API Key 和 AppID 等..."},
		menuItem{title: "批处理", desc: "执行不同模式的批量调用"},
		menuItem{title: "退出", desc: "退出程序"},
	}
	l := list.New(items, itemDelegate{}, 0, len(items)*2)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	l.KeyMap.NextPage.SetEnabled(false)
	l.KeyMap.PrevPage.SetEnabled(false)
	return l
}

func buildRulesMenu() list.Model {
	items := []list.Item{
		menuItem{title: "模式A", desc: "读取 Excel，批量调用百炼应用接口"},
		menuItem{title: "模式B", desc: "读取文件，批量调用百炼应用接口"},
		menuItem{title: "DIY 提问", desc: "n×m 规模：多问题 × 多文件"},
		menuItem{title: "工作流调用", desc: "自定义业务参数调用百炼应用"},
	}
	l := list.New(items, itemDelegate{}, 0, len(items)*2)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	l.KeyMap.NextPage.SetEnabled(false)
	l.KeyMap.PrevPage.SetEnabled(false)
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
				m.settings.reset(m.apiKey, m.appId, m.poolSize)
				m.view = ViewSettings
				return m, nil
			case 1:
				m.view = ViewRulesMenu
				return m, nil
			case 2:
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
	menu := MenuListStyle.Render(m.mainMenu.View())
	help := HelpStyle.Render("↑/↓ 选择  enter 确认  q 退出")
	if m.tip != "" {
		help += "\n" + WarnStyle.Render(m.tip)
	}
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
			var target View
			switch m.rulesMenu.Index() {
			case 0:
				target = ViewModeA
			case 1:
				target = ViewRulePDF
			case 2:
				target = ViewRuleDIY
			case 3:
				target = ViewRuleWorkflow
			}
			m.ruleDetail.reset(target)
			m.view = ViewRuleDetail
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.rulesMenu, cmd = m.rulesMenu.Update(msg)
	return m, cmd
}

func (m Model) rulesMenuView() string {
	title := TitleStyle.Render("运行模式")
	m.rulesMenu.SetHeight(len(m.rulesMenu.Items()) * 2)
	menu := MenuListStyle.Render(m.rulesMenu.View())
	help := HelpStyle.Render("↑/↓ 选择  enter 确认  esc 返回")
	return lipgloss.JoinVertical(lipgloss.Center, title, menu, help)
}
