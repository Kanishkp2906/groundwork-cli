package main

import "charm.land/lipgloss/v2"

// The palette the CLI family uses: violet accents on dark backgrounds.
var (
	accent      = lipgloss.Color("170")
	spinnerFG   = lipgloss.Color("69")
	dangerFG    = lipgloss.Color("212")
	warnFG      = lipgloss.Color("214")
	dimFG       = lipgloss.Color("241")
	textFG      = lipgloss.Color("252")
	selectedBG  = lipgloss.Color("170")
	selectedTxt = lipgloss.Color("16")
)

var (
	spinnerStyle = lipgloss.NewStyle().Foreground(spinnerFG)
	dimStyle     = lipgloss.NewStyle().Foreground(dimFG)

	titleStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true).
			MarginBottom(0)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(selectedTxt).
				Background(selectedBG).
				Padding(0, 1)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(textFG).
			Padding(0, 1, 0, 3)

	optionsTitleStyle = lipgloss.NewStyle().
				Foreground(textFG).
				Bold(true).
				Padding(0, 0, 0, 0)

	statusOKStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("150"))
	statusErrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))

	errorCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("203")).
			Padding(1, 3)

	headerBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("206")).
			Bold(true).
			Padding(0, 1)

	searchPromptStyle = lipgloss.NewStyle().Foreground(spinnerFG)
)
