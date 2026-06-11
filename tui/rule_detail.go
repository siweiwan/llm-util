package tui

import (
	"fmt"
	"llm-util/util/dirpicker"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/xuri/excelize/v2"
)

const helpModeA = `# 模式A — 批量请求

逐行读取 Excel 中的 问题，批量调用百炼应用接口。

## 前置配置

在 **配置管理** 中设置以下信息：

- **API Key**：百炼控制台 → 应用详情 → API Key
- **AppId**：百炼控制台 → 应用详情 → 应用 ID
- **Workspace ID**：百炼控制台 → 业务空间管理 → 空间 ID
  （可在 URL 中找到，如 llm-xxx）

## 使用步骤

1.  在 **配置管理**，设置 API Key、AppID、并发数
2.  点击 **模板下载**，生成模板 Excel
3.  在 **request** 列（A 列）填入每条请求内容
4.  选择 **运行任务**，选择 Excel 文件，按 Enter 执行

## 模板格式

| A(request) | B(response) | C(status) | D(time) | E(errMsg) |
|---|---|---|---|---|
| 提问内容 | （AI填写） | （自动） | （自动） | （自动） |

- 第 1 行标题，第 2 行起处理
- C 列为“完成”时自动跳过，失败行重新处理（断点续传）
- 每处理10条保存一次进度，避免中断后数据丢失

`

const helpModeB = `# 模式B — 批量请求

读取文件，批量调用百炼应用接口。

## 前置配置

在 **配置管理** 中设置以下信息：

- **API Key**：百炼控制台 → 应用详情 → API Key
- **AppId**：百炼控制台 → 应用详情 → 应用 ID
- **Workspace ID**：百炼控制台 → 业务空间管理 → 空间 ID
  （可在 URL 中找到，如 llm-xxx）
- **AccessKey ID / Secret**：阿里云控制台 → AccessKey 管理
  （建议用 RAM 子账号，授权 AliyunBailianFullAccess）
  用于文件上传到 OSS，模式 B 必须配置

## 使用步骤

1. 在 **配置管理** 设置以上全部配置
2. 点击 **选择文件夹** 或 **输入路径**，指定包含目标文件的文件夹
3. 点击 **模板下载**，生成模板，File 列自动填充
4. 在 **request** 列（A 列）填入每条请求内容
5. 选择 **运行任务**，选择填好的 Excel，按 Enter 执行

## 模板格式

| A(request) | B(fileName) | C(response) | D(status) | E(time) | F(errMsg) |
|---|---|---|---|---|---|
| 提问内容 | (自动填充) | (AI填写) | (自动) | (自动) | (自动) |

- 第 1 行标题，第 2 行起处理
- D 列为“完成”时自动跳过，失败行重新处理（断点续传）
- 每处理10条保存一次进度，避免中断后数据丢失
`

const helpDiyQuery = `# DIY 提问

处理 **n × m** 的提问规模（n 个问题 × m 个文件）。

## 前置配置

- **API Key / AppId**：百炼控制台 → 应用详情
- **Workspace ID**：百炼控制台 → 业务空间管理 → 空间 ID
- **AccessKey ID / Secret**：阿里云控制台 → AccessKey 管理
  （用于文件上传，建议用 RAM 子账号，授权 AliyunBailianFullAccess）

## 模板格式 (process.xlsx)

| A 列 | B 列 | C 列 | D 列 | E 列 |
|------|------|------|------|------|
| 问题1 | 文件1.pdf | （留空） | （自动） | （自动） |
| 问题2 | 文件2.pdf | （留空） | （自动） | （自动） |

- A 列：提问内容
- B 列：**files/** 目录下的文件名
- C 列：留空，AI 回复自动填入
- D 列为“完成”时自动跳过，失败行重新处理（断点续传）

## 文件准备

将需要提问的文件放入 **files/** 目录。
`

const helpWorkflow = `# 工作流调用

调用百炼应用，传递自定义业务参数。

## 前置配置

- **API Key / AppId**：百炼控制台 → 应用详情
- **Workspace ID**：百炼控制台 → 业务空间管理 → 空间 ID

## 模板格式 (workflow.xlsx)

| question | answer | status | 参数1 | 参数2 | ... |
|----------|--------|--------|-------|-------|-----|
| 提问内容 | （留空）| （自动） | 值 | 值 | ... |

- 第 1 行为参数名（表头）
- 第 2 行起每行一条请求
- A 列为问题，B 列为回答（自动填入）
- C 列为“完成”时自动跳过，失败行重新处理（断点续传）
- D 列及以后为自定义参数，列名即参数名
`

type ruleDetailView int

const (
	ruleDetailMenu ruleDetailView = iota
	ruleDetailHelp
	ruleDetailDirInput // 手动输入目录路径
)

type ruleDetailPanel struct {
	view         ruleDetailView
	menu         list.Model
	helpVP       viewport.Model
	dirInput     textinput.Model // 手动输入目录路径
	ruleName     string
	selectedRule View
	markdown     string
	saved        bool
	templateMsg  string
	selectedDir  string // Mode B: user-selected directory
	fileCount    int    // Mode B: number of files scanned
}

func newRuleDetailPanel() ruleDetailPanel {
	l := buildDefaultDetailMenu()
	vp := viewport.New(40, 10)

	ti := textinput.New()
	ti.Placeholder = "输入文件夹路径，如 D:\\docs"
	ti.CharLimit = 260
	ti.Width = 50

	return ruleDetailPanel{
		view:     ruleDetailMenu,
		menu:     l,
		helpVP:   vp,
		dirInput: ti,
	}
}

func buildDefaultDetailMenu() list.Model {
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
	return l
}

func buildModeBDetailMenu() list.Model {
	items := []list.Item{
		menuItem{title: "使用说明", desc: "查看此规则的详细用法"},
		menuItem{title: "选择文件夹", desc: "打开系统目录选择对话框"},
		menuItem{title: "输入路径", desc: "手动输入文件夹路径"},
		menuItem{title: "模板下载", desc: "生成模板，File列自动填充"},
		menuItem{title: "运行任务", desc: "开始执行批量任务"},
	}
	l := list.New(items, itemDelegate{}, 0, len(items)*2)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	return l
}

func (p *ruleDetailPanel) reset(rule View) {
	p.view = ruleDetailMenu
	p.selectedRule = rule
	p.saved = false
	p.selectedDir = ""
	p.fileCount = 0
	switch rule {
	case ViewModeA:
		p.ruleName = "模式A"
		p.markdown = helpModeA
		p.menu = buildDefaultDetailMenu()
	case ViewRuleFile:
		p.ruleName = "模式B"
		p.markdown = helpModeB
		p.menu = buildModeBDetailMenu()
	case ViewRuleDIY:
		p.ruleName = "DIY 提问"
		p.markdown = helpDiyQuery
		p.menu = buildDefaultDetailMenu()
	case ViewRuleWorkflow:
		p.ruleName = "工作流调用"
		p.markdown = helpWorkflow
		p.menu = buildDefaultDetailMenu()
	}
}

func (p *ruleDetailPanel) templateLetter() string {
	switch p.selectedRule {
	case ViewModeA:
		return "A"
	case ViewRuleFile:
		return "B"
	case ViewRuleDIY:
		return "C"
	case ViewRuleWorkflow:
		return "D"
	}
	return "A"
}

type dirPickerDoneMsg struct {
	path string
}

func (m Model) updateRuleDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	p := &m.ruleDetail

	// Handle dir picker result
	if dpm, ok := msg.(dirPickerDoneMsg); ok {
		if dpm.path != "" {
			p.selectedDir = dpm.path
			files := scanDirFiles(dpm.path)
			p.fileCount = len(files)
			p.saved = true
			p.templateMsg = SuccessStyle.Render(fmt.Sprintf("✅ 已选择目录，扫描到 %d 个文件", p.fileCount))
		}
		return m, nil
	}

	switch p.view {
	case ruleDetailMenu:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.view = ViewRulesMenu
				return m, nil
			case "enter":
				return m.handleDetailMenuEnter()
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

	case ruleDetailDirInput:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				p.dirInput.Blur()
				p.view = ruleDetailMenu
				return m, nil
			case "enter":
				p.dirInput.Blur()
				dirPath := p.dirInput.Value()
				if dirPath == "" {
					p.view = ruleDetailMenu
					return m, nil
				}
				// 验证目录是否存在
				info, err := os.Stat(dirPath)
				if err != nil || !info.IsDir() {
					p.saved = true
					p.templateMsg = ErrorStyle.Render("❌ 路径不存在或不是文件夹: " + dirPath)
					p.view = ruleDetailMenu
					return m, nil
				}
				p.selectedDir = dirPath
				files := scanDirFiles(dirPath)
				p.fileCount = len(files)
				p.saved = true
				p.templateMsg = SuccessStyle.Render(fmt.Sprintf("✅ 已选择目录，扫描到 %d 个文件", p.fileCount))
				p.view = ruleDetailMenu
				return m, nil
			}
		}
		var cmd tea.Cmd
		p.dirInput, cmd = p.dirInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleDetailMenuEnter() (tea.Model, tea.Cmd) {
	p := &m.ruleDetail
	idx := p.menu.Index()

	// Mode B has 4 items, others have 3
	if p.selectedRule == ViewRuleFile {
		switch idx {
		case 0: // 使用说明
			return m.showHelp()
		case 1: // 选择文件夹（系统对话框）
			return m.startDirPicker()
		case 2: // 输入路径（手动输入）
			p.dirInput.SetValue(p.selectedDir) // 回显已有路径
			p.dirInput.Focus()
			p.view = ruleDetailDirInput
			return m, textinput.Blink
		case 3: // 模板下载
			return m.downloadModeBTemplate()
		case 4: // 运行任务
			m.view = p.selectedRule
			m.batch.reset()
			return m, m.batch.startCmd()
		}
	} else {
		switch idx {
		case 0: // 使用说明
			return m.showHelp()
		case 1: // 模板下载
			return m.downloadTemplate()
		case 2: // 运行任务
			m.view = p.selectedRule
			m.batch.reset()
			return m, m.batch.startCmd()
		}
	}
	return m, nil
}

func (m Model) showHelp() (tea.Model, tea.Cmd) {
	p := &m.ruleDetail
	md, _ := glamour.Render(p.markdown, "dark")
	p.helpVP.SetContent(md)
	p.helpVP.GotoTop()
	p.view = ruleDetailHelp
	return m, nil
}

func (m Model) startDirPicker() (tea.Model, tea.Cmd) {
	req, err := dirpicker.NewPickerRequest()
	if err != nil {
		m.ruleDetail.saved = true
		m.ruleDetail.templateMsg = ErrorStyle.Render("❌ 无法打开目录选择器: " + err.Error())
		return m, nil
	}
	// tea.ExecProcess suspends the TUI, runs the process, then restores
	return m, tea.ExecProcess(req.Cmd, func(err error) tea.Msg {
		path, _ := req.ReadResult()
		return dirPickerDoneMsg{path: path}
	})
}

func (m Model) downloadTemplate() (tea.Model, tea.Cmd) {
	p := &m.ruleDetail
	filename, msg, err := generateTemplate(p.templateLetter())
	if err == nil {
		p.saved = true
		if msg != "" {
			p.templateMsg = InfoStyle.Render(msg)
		} else {
			p.templateMsg = SuccessStyle.Render("✅ 已生成: " + filename)
		}
	}
	return m, nil
}

func (m Model) downloadModeBTemplate() (tea.Model, tea.Cmd) {
	p := &m.ruleDetail
	if p.selectedDir == "" {
		p.saved = true
		p.templateMsg = WarnStyle.Render("⚠️  请先选择文件目录")
		return m, nil
	}
	files := scanDirFiles(p.selectedDir)
	if len(files) == 0 {
		p.saved = true
		p.templateMsg = WarnStyle.Render("⚠️  目录下没有文件")
		return m, nil
	}
	filename, msg, err := generateModeBTemplate(p.selectedDir, files)
	if err == nil {
		p.saved = true
		if msg != "" {
			p.templateMsg = InfoStyle.Render(msg)
		} else {
			p.templateMsg = SuccessStyle.Render("✅ 已生成: " + filename)
		}
	}
	return m, nil
}

func (m Model) ruleDetailView() string {
	p := m.ruleDetail
	title := PanelTitleStyle.Render(p.ruleName)

	switch p.view {
	case ruleDetailMenu:
		maxH := m.height - 6
		menuH := len(p.menu.Items()) * 2
		if menuH > maxH {
			menuH = maxH
		}
		if menuH < 2 {
			menuH = 2
		}
		p.menu.SetHeight(menuH)
		menu := lipgloss.NewStyle().Padding(1, 2).Render(p.menu.View())
		help := HelpStyle.Render("↑/↓ 选择  enter 确认  esc 返回")
		if p.saved {
			help += "\n" + p.templateMsg
		}
		parts := []string{title, menu}
		if p.selectedDir != "" {
			dirInfo := InfoStyle.Render("📂 " + p.selectedDir)
			parts = append(parts, dirInfo)
		}
		parts = append(parts, help)
		return lipgloss.JoinVertical(lipgloss.Center, parts...)

	case ruleDetailDirInput:
		title := PanelTitleStyle.Render(p.ruleName + " — 输入路径")
		input := p.dirInput.View()
		help := HelpStyle.Render("enter 确认  esc 取消")
		return lipgloss.JoinVertical(lipgloss.Center, title, input, help)

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

// scanDirFiles returns all file names (not directories) in the given directory.
func scanDirFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	return files
}

func generateTemplate(letter string) (filename string, msg string, err error) {
	switch letter {
	case "C":
		return generateDIYTemplate()
	case "D":
		return generateWorkflowTemplate()
	default:
		return generateModeATemplate()
	}
}

func generateModeATemplate() (string, string, error) {
	now := time.Now()
	filename := fmt.Sprintf("template-A-%s.xlsx", now.Format("20060102150405"))

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

	return filename, "", f.SaveAs(filename)
}

func generateModeBTemplate(dir string, files []string) (string, string, error) {
	now := time.Now()
	filename := fmt.Sprintf("template-B-%s.xlsx", now.Format("20060102150405"))

	f := excelize.NewFile()
	defer f.Close()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#06B6D4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetSheetRow("Sheet1", "A1", &[]string{"request", "fileName", "response", "status", "time", "errMsg"})
	f.SetCellStyle("Sheet1", "A1", "F1", headerStyle)
	// Store directory path in G1 for backend use
	f.SetCellValue("Sheet1", "G1", dir)
	f.SetColWidth("Sheet1", "A", "A", 40)
	f.SetColWidth("Sheet1", "B", "B", 30)
	f.SetColWidth("Sheet1", "C", "C", 40)
	f.SetColWidth("Sheet1", "D", "D", 12)
	f.SetColWidth("Sheet1", "E", "E", 20)
	f.SetColWidth("Sheet1", "F", "F", 40)
	f.SetPanes("Sheet1", &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Pre-fill B column with file names
	for i, name := range files {
		row := i + 2 // row 1 is header
		f.SetCellValue("Sheet1", fmt.Sprintf("B%d", row), name)
	}

	return filename, "", f.SaveAs(filename)
}

func generateDIYTemplate() (string, string, error) {
	now := time.Now()
	filename := fmt.Sprintf("template-C-%s.xlsx", now.Format("20060102150405"))

	f := excelize.NewFile()
	defer f.Close()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#06B6D4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetSheetRow("Sheet1", "A1", &[]string{"问题", "文件名", "回答", "status", "errMsg"})
	f.SetCellStyle("Sheet1", "A1", "E1", headerStyle)
	f.SetColWidth("Sheet1", "A", "A", 40)
	f.SetColWidth("Sheet1", "B", "B", 30)
	f.SetColWidth("Sheet1", "C", "C", 40)
	f.SetColWidth("Sheet1", "D", "D", 12)
	f.SetColWidth("Sheet1", "E", "E", 40)
	f.SetPanes("Sheet1", &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	return filename, "", f.SaveAs(filename)
}

func generateWorkflowTemplate() (string, string, error) {
	now := time.Now()
	filename := fmt.Sprintf("template-D-%s.xlsx", now.Format("20060102150405"))

	f := excelize.NewFile()
	defer f.Close()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#06B6D4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetSheetRow("Sheet1", "A1", &[]string{"question", "answer", "status", "参数1", "参数2"})
	f.SetCellStyle("Sheet1", "A1", "E1", headerStyle)
	f.SetColWidth("Sheet1", "A", "B", 40)
	f.SetColWidth("Sheet1", "C", "C", 12)
	f.SetColWidth("Sheet1", "D", "E", 20)
	f.SetPanes("Sheet1", &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	return filename, "", f.SaveAs(filename)
}
