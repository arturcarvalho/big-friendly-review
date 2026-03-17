package main

import "charm.land/lipgloss/v2"

var (
	headerFocusedStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#fbf1c7")).
		Background(lipgloss.Color("#504945")).
		Padding(0, 1)

	headerUnfocusedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#928374")).
		Background(lipgloss.Color("#3c3836")).
		Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#928374"))

	separatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#504945"))

	selectedStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#fbf1c7")).
		Background(lipgloss.Color("#504945"))

	normalStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ebdbb2"))

	warningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fe8019")).
		Bold(true)

	dimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#928374")).
		Italic(true)

	gutterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fabd2f"))

	dimBlockStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#504945"))

	reviewedBlockStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#928374"))

	reviewedGutterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#b8bb26"))

	changedGutterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fabd2f"))

	unreviewedGutterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#504945"))

	blockSelectionGutterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#83a598"))

	reviewedFileStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#b8bb26"))

	changedFileStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fabd2f"))

	lineNumberStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#504945"))

	fileIndicatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#928374"))

	commentStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#83a598"))

	helpKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fabd2f"))

	helpModalStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ebdbb2")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#504945")).
		Padding(1, 3)

	helpTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fbf1c7")).
		Bold(true)

	helpDescStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7c6f64"))
)
