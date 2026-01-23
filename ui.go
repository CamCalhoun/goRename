package main

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	DividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	ItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	HighlightStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))
)
