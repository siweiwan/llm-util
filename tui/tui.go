package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type View int

const (
	ViewMainMenu View = iota
	ViewChat
	ViewRulesMenu
	ViewRuleCase
	ViewRulePDF
	ViewRuleDIY
	ViewRuleWorkflow
)

type Message struct {
	Role    string
	Content string
}

// Callbacks set by main.go
type ChatFunc func(prompt string, history []Message) (string, error)
type ChatFileFunc func(prompt, filePath string) (string, error)

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

	apiKey  string
	appId   string
	history []Message

	mainMenu  list.Model
	rulesMenu list.Model

	OnSend        ChatFunc
	OnSendFile    ChatFileFunc
	OnRunCase     StartBatchFunc
	OnRunPDF      func(poolSize int, question string, progress chan<- ProgressMsg) error
	OnRunDIY      StartBatchFunc
	OnRunWorkflow StartBatchFunc

	chat  chatPanel
	batch batchPanel

	width  int
	height int
}

func NewModel(apiKey, appId string) Model {
	return Model{
		view:      ViewMainMenu,
		apiKey:    apiKey,
		appId:     appId,
		mainMenu:  buildMainMenu(),
		rulesMenu: buildRulesMenu(),
		chat:      newChatPanel(),
		batch:     newBatchPanel(),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	switch m.view {
	case ViewMainMenu:
		return m.updateMainMenu(msg)
	case ViewRulesMenu:
		return m.updateRulesMenu(msg)
	case ViewChat:
		return m.updateChat(msg)
	case ViewRuleCase, ViewRulePDF, ViewRuleDIY, ViewRuleWorkflow:
		return m.updateBatch(msg)
	}
	return m, nil
}

func (m Model) View() string {
	switch m.view {
	case ViewMainMenu:
		return m.mainMenuView()
	case ViewRulesMenu:
		return m.rulesMenuView()
	case ViewChat:
		return m.chatView()
	case ViewRuleCase, ViewRulePDF, ViewRuleDIY, ViewRuleWorkflow:
		return m.batchView()
	}
	return ""
}



