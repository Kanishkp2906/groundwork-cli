package main

import (
	"charm.land/lipgloss/v2"
)

var (
	// Unicode block logo
	logoArtStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("207")).
			Bold(true)

	// Simple container
	logoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 2)
)

// groundworkLogo is the figlet-style block logo. Lines are kept byte-identical
// (including trailing spaces) so lipgloss width math and subtitle centering stay exact.
const groundworkLogo = "▄▀▀▀  █▀▀▄ ▄▀▀▀▄ █  █ █▄  █ █▀▀▄ █   █ ▄▀▀▀▄ █▀▀▄ █ ▄▀\n" +
	"█  ▀█ █▀▀▄ █   █ █  █ █ ▀▄█ █  █ █ ▄ █ █   █ █▀▀▄ █▀▄ \n" +
	" ▀▀▀  ▀  ▀  ▀▀▀   ▀▀  ▀   ▀ ▀▀▀   ▀ ▀   ▀▀▀  ▀  ▀ ▀  ▀ "

// logoBlock returns the boxed Unicode logo. Placement is up to the caller, so
// screens can left-align or center the whole layout.
func logoBlock(width int) string {
	art := logoArtStyle.Render(groundworkLogo)

	content := lipgloss.JoinVertical(lipgloss.Left, art)

	boxed := logoBoxStyle.Render(content)

	return boxed
}
