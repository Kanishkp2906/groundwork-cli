package main

import (
	"charm.land/lipgloss/v2"
)

var (
	// Unicode block logo
	logoArtStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("207")).
			Bold(true)

	logoSubStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)

	// Simple container
	logoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 2).
			MarginTop(1).
			MarginBottom(1)
)

// The new Unicode block logo
const groundworkLogo = " ▗▄▄▖▗▄▄▖  ▗▄▖ ▗▖ ▗▖▗▖  ▗▖▗▄▄▄ ▗▖ ▗▖ ▗▄▖ ▗▄▄▖ ▗▖ ▗▖\n" +
	"▐▌   ▐▌ ▐▌▐▌ ▐▌▐▌ ▐▌▐▛▚▖▐▌▐▌  █▐▌ ▐▌▐▌ ▐▌▐▌ ▐▌▐▌▗▞▘\n" +
	"▐▌▝▜▌▐▛▀▚▖▐▌ ▐▌▐▌ ▐▌▐▌ ▝▜▌▐▌  █▐▌ ▐▌▐▌ ▐▌▐▛▀▚▖▐▛▚▖ \n" +
	"▝▚▄▞▘▐▌ ▐▌▝▚▄▞▘▝▚▄▞▘▐▌  ▐▌▐▙▄▄▀▐▙█▟▌▝▚▄▞▘▐▌ ▐▌▐▌ ▐▌"

// logoBlock returns the Unicode logo with centered subtitle.
func logoBlock(width int) string {
	art := logoArtStyle.Render(groundworkLogo)

	// Art left-aligned, subtitle centered
	subtitleStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		Width(lipgloss.Width(art)).
		Align(lipgloss.Center).
		Render("Real CVE Training for Developers")

	content := lipgloss.JoinVertical(lipgloss.Left, art, "", subtitleStyled)

	boxed := logoBoxStyle.Render(content)

	return lipgloss.PlaceHorizontal(width, lipgloss.Left, boxed)
}

// smallLogo returns a compact version for loading screens.
func smallLogo(width int) string {
	art := logoArtStyle.Render(groundworkLogo)

	// For small logo, center the subtitle
	subtitleStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		Width(lipgloss.Width(art)).
		Align(lipgloss.Center).
		Render("Real CVE Training for Developers")

	content := lipgloss.JoinVertical(lipgloss.Left, art, "", subtitleStyled)

	boxed := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 2).
		Render(content)

	return lipgloss.PlaceHorizontal(width, lipgloss.Left, boxed)
}
