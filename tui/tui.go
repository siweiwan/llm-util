package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type View int

const (
	ViewMainMenu View = iota
	ViewSettings
	ViewRulesMenu
	ViewRuleDetail
	ViewModeA
	ViewRuleFile
	ViewRuleDIY
	ViewRuleWorkflow
)

// Callbacks set by main.go
type showTipMsg string

type ProgressMsg struct {
	Index    int
	Total    int
	Filename string
	Status   string // "processing", "done", "error", "skip"
}

// StartBatchFunc receives poolSize and a progress channel, runs batch, closes channel when done.
type StartBatchFunc func(poolSize int, progress chan<- ProgressMsg) error

type Model struct {
	view View

	apiKey   string
	appId    string
	poolSize int

	mainMenu  list.Model
	rulesMenu list.Model

	OnRunModeA    func(poolSize int, filename string, progress chan<- ProgressMsg) error
	OnRunModeB    func(poolSize int, xlsxFile string, progress chan<- ProgressMsg) error
	OnRunDIY      StartBatchFunc
	OnRunWorkflow StartBatchFunc

	OnSaveSettings func(apiKey, appId string, poolSize int) error

	settings   settingsPanel
	batch      batchPanel
	ruleDetail ruleDetailPanel

	tip    string
	width  int
	height int
}

func NewModel(apiKey, appId string, poolSize int) Model {
	return Model{
		view:       ViewMainMenu,
		apiKey:     apiKey,
		appId:      appId,
		poolSize:   poolSize,
		mainMenu:   buildMainMenu(),
		rulesMenu:  buildRulesMenu(),
		settings:   newSettingsPanel(apiKey, appId, poolSize),
		batch:      newBatchPanel(),
		ruleDetail: newRuleDetailPanel(),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.mainMenu.SetSize(msg.Width-4, len(m.mainMenu.Items())*2)
		m.rulesMenu.SetSize(msg.Width-4, len(m.rulesMenu.Items())*2)
		return m, nil
	case tea.KeyMsg:
		m.tip = ""
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case showTipMsg:
		m.tip = string(msg)
		return m, nil
	}
	switch m.view {
	case ViewMainMenu:
		return m.updateMainMenu(msg)
	case ViewSettings:
		return m.updateSettings(msg)
	case ViewRulesMenu:
		return m.updateRulesMenu(msg)
	case ViewRuleDetail:
		return m.updateRuleDetail(msg)
	case ViewModeA, ViewRuleFile, ViewRuleDIY, ViewRuleWorkflow:
		return m.updateBatch(msg)
	}
	return m, nil
}

func (m Model) View() string {
	switch m.view {
	case ViewMainMenu:
		return m.mainMenuView()
	case ViewSettings:
		return m.settingsView()
	case ViewRulesMenu:
		return m.rulesMenuView()
	case ViewRuleDetail:
		return m.ruleDetailView()
	case ViewModeA, ViewRuleFile, ViewRuleDIY, ViewRuleWorkflow:
		return m.batchView()
	}
	return ""
}
