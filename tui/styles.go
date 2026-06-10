package tui

import "github.com/charmbracelet/lipgloss"

var (
	Accent = lipgloss.Color("#06B6D4")
	Green  = lipgloss.Color("#22C55E")
	Red    = lipgloss.Color("#EF4444")
	Yellow = lipgloss.Color("#EAB308")
	Blue   = lipgloss.Color("#3B82F6")
	Dim    = lipgloss.Color("#9CA3AF")
	White  = lipgloss.Color("#F8FAFC")
	Dark   = lipgloss.Color("#111827")
)

var (
	TitleStyle        = lipgloss.NewStyle().Bold(true).Foreground(Accent).Padding(0, 2).MarginBottom(1)
	HelpStyle         = lipgloss.NewStyle().Foreground(Dim).MarginTop(1)
	SuccessStyle      = lipgloss.NewStyle().Foreground(Green)
	ErrorStyle        = lipgloss.NewStyle().Foreground(Red)
	InfoStyle         = lipgloss.NewStyle().Foreground(Blue)
	WarnStyle         = lipgloss.NewStyle().Foreground(Yellow)
	DimStyle          = lipgloss.NewStyle().Foreground(Dim)
	PanelTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(Dark).Background(Accent).Padding(0, 1)
	SelectedItemStyle = lipgloss.NewStyle().Foreground(Dark).Background(Accent).Padding(0, 1)
	NormalItemStyle   = lipgloss.NewStyle().Padding(0, 1)
	SelectedDescStyle = lipgloss.NewStyle().Foreground(Dark).Background(Accent).Padding(0, 1)
	NormalDescStyle   = lipgloss.NewStyle().Foreground(Dim).Padding(0, 1)
	MenuListStyle     = lipgloss.NewStyle().Padding(1, 2)
)
