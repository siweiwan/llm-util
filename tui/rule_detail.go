package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/xuri/excelize/v2"
)

const helpModeA = `# 模式A — 批量请求

逐行读取 Excel 中的 问题，批量调用百炼应用接口。

## 使用步骤

1.  在 **配置管理**，设置 并发数、AppID
2.  点击 **模板下载**，生成模板 Excel
3.  在 **request** 列（A 列）填入每条请求内容
4.  选择 **运行任务**，拖入或输入 Excel 文件路径，按 Enter 执行

## 模板格式

| A(request) | B(response) | C(status) | D(time) | E(errMsg) |
|---|---|---|---|---|
| 提问内容 | （AI填写） | （自动） | （自动） | （自动） |

- 第 1 行标题，第 2 行起处理
- B 列有值自动跳过，支持断点续传

`

const helpPdfBatch = `# PDF 批量提问

对 **pdfs/** 目录下所有 PDF 文件提问同一个问题。

## 使用步骤

1. 在程序目录下创建 **pdfs/** 文件夹
2. 将要提问的 PDF 文件放入该文件夹
3. 选择此规则，输入问题
4. 结果保存至 **{问题前20字}.xlsx**

## 断点续传

通过 MD5 校验跳过已处理的文件，中断后重新运行不会重复处理。
`

const helpDiyQuery = `# DIY 提问

处理 **n × m** 的提问规模（n 个问题 × m 个文件）。

## 模板格式 (process.xlsx)

| A 列 | B 列 | C 列 |
|------|------|------|
| 问题1 | 文件1.pdf | （留空） |
| 问题2 | 文件2.pdf | （留空） |

- A 列：提问内容
- B 列：**files/** 目录下的文件名
- C 列：留空，AI 回复自动填入

## 文件准备

将需要提问的文件放入 **files/** 目录。
`

const helpWorkflow = `# 工作流调用

调用百炼应用，传递自定义业务参数。

## 模板格式 (workflow.xlsx)

| question | answer | 参数1 | 参数2 | ... |
|----------|--------|-------|-------|-----|
| 提问内容 | （留空）| 值 | 值 | ... |

- 第 1 行为参数名（表头）
- 第 2 行起每行一条请求
- A 列为问题，B 列为回答（自动填入）
- C 列及以后为自定义参数，列名即参数名
`

type ruleDetailView int

const (
	ruleDetailMenu ruleDetailView = iota
	ruleDetailHelp
)

type ruleDetailPanel struct {
	view         ruleDetailView
	menu         list.Model
	helpVP       viewport.Model
	ruleName     string
	selectedRule View
	markdown     string
	saved        bool
	templateMsg  string
}

func newRuleDetailPanel() ruleDetailPanel {
	items := []list.Item{
		menuItem{title: "使用说明", desc: "查看此规则的详细用法"},
		menuItem{title: "模板下载", desc: "生成模板文件到当前目录"},
		menuItem{title: "运行任务", desc: "开始执行批量任务"},
	}
	l := list.New(items, itemDelegate{}, 0, len(items)*2)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()

	vp := viewport.New(40, 10)

	return ruleDetailPanel{
		view:   ruleDetailMenu,
		menu:   l,
		helpVP: vp,
	}
}

func (p *ruleDetailPanel) reset(rule View) {
	p.view = ruleDetailMenu
	p.selectedRule = rule
	p.saved = false
	switch rule {
	case ViewModeA:
		p.ruleName = "模式A"
		p.markdown = helpModeA
	case ViewRulePDF:
		p.ruleName = "PDF 批量提问"
		p.markdown = helpPdfBatch
	case ViewRuleDIY:
		p.ruleName = "DIY 提问"
		p.markdown = helpDiyQuery
	case ViewRuleWorkflow:
		p.ruleName = "工作流调用"
		p.markdown = helpWorkflow
	}
}

func (p *ruleDetailPanel) templateLetter() string {
	switch p.selectedRule {
	case ViewModeA:
		return "A"
	case ViewRulePDF:
		return "B"
	case ViewRuleDIY:
		return "C"
	case ViewRuleWorkflow:
		return "D"
	}
	return "A"
}

func (m Model) updateRuleDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	p := &m.ruleDetail

	switch p.view {
	case ruleDetailMenu:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.view = ViewRulesMenu
				return m, nil
			case "enter":
				switch p.menu.Index() {
				case 0:
					md, _ := glamour.Render(p.markdown, "dark")
					p.helpVP.SetContent(md)
					p.helpVP.GotoTop()
					p.view = ruleDetailHelp
					return m, nil
				case 1:
					filename, err := generateTemplate(p.templateLetter())
					if err == nil {
						p.saved = true
						p.templateMsg = SuccessStyle.Render("✅ 已生成: " + filename)
					}
					return m, nil
				case 2:
					m.view = p.selectedRule
					m.batch.reset()
					return m, m.batch.startCmd()
				}
			}
		}
		var cmd tea.Cmd
		p.menu, cmd = p.menu.Update(msg)
		return m, cmd

	case ruleDetailHelp:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				p.view = ruleDetailMenu
				return m, nil
			}
		}
		var cmd tea.Cmd
		p.helpVP, cmd = p.helpVP.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) ruleDetailView() string {
	p := m.ruleDetail
	title := PanelTitleStyle.Render(p.ruleName)

	switch p.view {
	case ruleDetailMenu:
		p.menu.SetHeight(len(p.menu.Items()) * 2)
		menu := lipgloss.NewStyle().Padding(1, 2).Render(p.menu.View())
		help := HelpStyle.Render("↑/↓ 选择  enter 确认  esc 返回")
		if p.saved {
			help += "\n" + p.templateMsg
		}
		return lipgloss.JoinVertical(lipgloss.Center, title, menu, help)

	case ruleDetailHelp:
		p.helpVP.Width = m.width - 4
		p.helpVP.Height = m.height - 4
		return lipgloss.JoinVertical(lipgloss.Left,
			title,
			p.helpVP.View(),
			HelpStyle.Render("esc 返回"),
		)
	}
	return ""
}

func generateTemplate(letter string) (string, error) {
	now := time.Now()
	filename := fmt.Sprintf("template-%s-%s.xlsx", letter, now.Format("20060102150405"))

	f := excelize.NewFile()
	defer f.Close()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#06B6D4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetSheetRow("Sheet1", "A1", &[]string{"request", "response", "status", "time", "errMsg"})
	f.SetCellStyle("Sheet1", "A1", "E1", headerStyle)
	f.SetColWidth("Sheet1", "A", "B", 40)
	f.SetColWidth("Sheet1", "C", "C", 12)
	f.SetColWidth("Sheet1", "D", "D", 20)
	f.SetColWidth("Sheet1", "E", "E", 40)
	f.SetPanes("Sheet1", &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	return filename, f.SaveAs(filename)
}
