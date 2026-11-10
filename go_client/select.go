package main

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// languageOptions mirrors the language list of groundwork-cli. The Slug is what
// gets sent to the ticket endpoint, the Note mirrors the package pools in the
// server's language_conf.py.
type languageOption struct {
	Label string
	Slug  string
	Note  string
}

var languageOptions = []languageOption{
	{"Python", "python", "django · flask · fastapi · requests · cryptography"},
	{"Javascript", "javascript", "express · axios · jsonwebtoken · lodash · mongoose"},
	{"Java", "java", "log4j-core · spring-core · commons-text · jackson-databind"},
	{"C/C++", "c++", "curl · libpng · libtiff · ffmpeg · sqlite3"},
}

// selectModel is the language picker shown under the GroundWork logo.
type selectModel struct {
	cursor   int
	quitting bool
	keys     selectKeys
}

func newSelectModel() selectModel {
	return selectModel{keys: newSelectKeys()}
}

func (m selectModel) choice() (languageOption, bool) {
	if m.quitting || len(languageOptions) == 0 {
		return languageOption{}, false
	}
	return languageOptions[m.cursor%len(languageOptions)], true
}

func (m selectModel) update(msg tea.Msg) (selectModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case keyMatches(msg, m.keys.moveUp):
			m.cursor = (m.cursor - 1 + len(languageOptions)) % len(languageOptions)
		case keyMatches(msg, m.keys.moveDown):
			m.cursor = (m.cursor + 1) % len(languageOptions)
		case keyMatches(msg, m.keys.choose):
			if choice, ok := m.choice(); ok {
				// Enter confirms the language; the app cycles into loading mode.
				return m, chooseLanguageCmd(choice)
			}
		case keyMatches(msg, m.keys.quit):
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectModel) view(width, height int) string {
	var b strings.Builder

	// Logo
	b.WriteString(logoBlock(width))
	b.WriteString("\n\n")

	// Title
	b.WriteString(optionsTitleStyle.Render("Select a language to debug a real CVE:"))
	b.WriteString("\n\n")

	// Language options — simple list, marker on the selected row
	for i, lang := range languageOptions {
		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render("› " + lang.Label))
		} else {
			b.WriteString(normalItemStyle.Render(lang.Label))
		}
		b.WriteString("\n")
	}

	// Help line
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Join([]string{
		hintPair(m.keys.moveUp),
		hintPair(m.keys.moveDown),
		hintPair(m.keys.choose),
		hintPair(m.keys.quit),
	}, "  ·  ")))
	b.WriteString("\n")

	content := b.String()

	// Pad every line to the widest line so Place centers the block as a
	// single sheet (left-aligned rows inside a centered card) instead of
	// centering each line individually.
	content = lipgloss.NewStyle().
		Width(lipgloss.Width(content)).
		Render(content)

	// Center the whole sheet horizontally and vertically in the terminal
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

func hintPair(b key.Binding) string {
	k, d := b.Help().Key, b.Help().Desc
	return k + " " + d
}
