package tui

import "github.com/charmbracelet/lipgloss"

var (
	Purple = lipgloss.Color("#7C3AED")
	Green  = lipgloss.Color("#10B981")
	Red    = lipgloss.Color("#EF4444")
	Yellow = lipgloss.Color("#F59E0B")
	Blue   = lipgloss.Color("#3B82F6")
	Gray   = lipgloss.Color("#6B7280")
	White  = lipgloss.Color("#F9FAFB")
)

var (
	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(Purple).Padding(0, 2).MarginBottom(1)
	HelpStyle  = lipgloss.NewStyle().Foreground(Gray).MarginTop(1)
)

var (
	SuccessStyle = lipgloss.NewStyle().Foreground(Green)
	ErrorStyle   = lipgloss.NewStyle().Foreground(Red)
	InfoStyle    = lipgloss.NewStyle().Foreground(Blue)
	WarnStyle    = lipgloss.NewStyle().Foreground(Yellow)
)

var PanelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(White).Background(Purple).Padding(0, 1)
