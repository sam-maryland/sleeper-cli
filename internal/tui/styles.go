package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700")).
			MarginLeft(2)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#888888")).
			MarginLeft(2)

	playoffRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	regularRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA"))

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			MarginLeft(2)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			MarginLeft(2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			MarginLeft(2)
)
