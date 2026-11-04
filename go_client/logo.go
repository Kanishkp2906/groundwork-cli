package main

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

var bannerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

// The GroundWork wordmark, identical to the art groundwork-cli prints at
// startup. The one backtick inside the art is written as a placeholder (§)
// because Go raw string literals cannot contain backticks.
const groundworkArtRaw = `   ____                           ___        __         §
  / ___|_ __ ___  _   _ _ __   __| \ \      / /__  _ __| | __
 | |  _| '__/ _ \| | | | '_ \ / _§ |\ \ /\ / / _ \| '__| |/ /
 | |_| | | | (_) | |_| | | | | (_| | \ V  V / (_) | |  |   <
  \____|_|  \___/ \__,_|_| |_|\__,_|  \_/\_/ \___/|_|  |_|\_\`

// banner line used by groundwork-cli, kept verbatim.
const bannerText = "Real CVE Training for Developers"

func groundworkArt() string {
	return strings.ReplaceAll(groundworkArtRaw, "§", "\x60")
}

const bannerWidth = 60

func bannerBox() string {
	inner := bannerText
	padLeft := (bannerWidth - 2 - len(inner)) / 2
	padRight := bannerWidth - 2 - len(inner) - padLeft
	lines := []string{
		"╔" + strings.Repeat("═", bannerWidth-2) + "╗",
		"║" + strings.Repeat(" ", padLeft) + inner + strings.Repeat(" ", padRight) + "║",
		"╚" + strings.Repeat("═", bannerWidth-2) + "╝",
	}
	return bannerStyle.Render(strings.Join(lines, "\n"))
}

// logoBlock renders the wordmark plus the banner box, LEFT ALIGNED.
func logoBlock(width int) string {
	if width <= 0 {
		width = bannerWidth
	}
	block := bannerStyle.Render(groundworkArt()) + "\n" + bannerBox()
	return block // left-aligned; no PlaceHorizontal centering
}

// smallLogo is used on screens where the full art would be noise.
func smallLogo() string {
	return fmt.Sprintf("%s %s %s",
		spinnerStyle.Render("◆"),
		bannerStyle.Render("groundwork"),
		dimStyle.Render("real CVE training"),
	)
}
