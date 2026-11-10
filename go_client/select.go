package main

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
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

// selectItem adapts a languageOption to the bubbles fancy list: the language
// label as the title, its package pools as the dimmed description underneath.
type selectItem struct{ lang languageOption }

func (i selectItem) Title() string       { return i.lang.Label }
func (i selectItem) Description() string { return i.lang.Note }
func (i selectItem) FilterValue() string { return i.lang.Label }

// selectModel is the language picker shown under the GroundWork logo, backed
// by the bubbles fancy list (title bar, status bar, paginated items).
type selectModel struct {
	list list.Model
	keys selectKeys
}

func newSelectModel() selectModel {
	items := make([]list.Item, 0, len(languageOptions))
	for _, lang := range languageOptions {
		items = append(items, selectItem{lang})
	}

	delegate := list.NewDefaultDelegate()
	// Two-line rows: title (language) + description (package pools) — the
	// delegate's default height of 2 and spacing of 1 give the airy fancy look.

	l := list.New(items, delegate, 60, 10)
	l.Title = "Select a language to debug a real CVE:"
	// Stock TitleBar padding bottom (1 row) = the gap between the title and
	// the items. StatusBar is hidden, so its style is untouched.
	l.Styles.TitleBar = l.Styles.TitleBar.Padding(0, 0, 1, 2)
	l.SetShowStatusBar(false) // no "N items" counter (user call)
	l.SetShowHelp(false)      // the picker draws its own dim help line
	// Filtering and pagination keep their stock defaults: '/' filters,
	// and the dots render only when a real page break exists.

	return selectModel{keys: newSelectKeys(), list: l}
}

// resize fits the list into the space left by the logo box + help line. Called
// from app.resize() on every WindowSizeMsg.
func (m *selectModel) resize(w, h int) {
	// Fitted to the v2 list layout: title bar (1) + its padding gap (1) +
	// the pager's reserved dots row (1) + one page of 4×2-line items with
	// 1-row gaps needs 15 rows. Below that the dots page item 4 away.
	listH := min(15, max(6, h-11))
	listW := min(62, w-2)
	if listW < 16 {
		listW = max(16, w)
	}
	m.list.SetSize(listW, listH)
}

func (m selectModel) update(msg tea.Msg) (selectModel, tea.Cmd) {
	if km, ok := msg.(tea.KeyPressMsg); ok {
		// Quit belongs to this screen, never to the list component.
		if keyMatches(km, m.keys.quit) {
			return m, tea.Quit
		}
	}

	// The fancy list owns navigation: arrows, k/j, home/end, mouse wheel…
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	// Enter picks the highlighted language and cycles into loading. While
	// the stock '/' filter is open, Enter accepts the filter instead.
	if km, ok := msg.(tea.KeyPressMsg); ok && keyMatches(km, m.keys.choose) &&
		m.list.FilterState() != list.Filtering {
		if it, ok := m.list.SelectedItem().(selectItem); ok {
			return m, chooseLanguageCmd(it.lang)
		}
	}

	return m, cmd
}

func (m selectModel) view(width, height int) string {
	// JoinVertical pads every row to the sheet width, so Place centers the
	// block as one sheet (left-aligned rows inside a centered card).
	content := lipgloss.JoinVertical(lipgloss.Left,
		logoBlock(width),
		"",
		m.list.View(),
		"",
		helpLine(m.keys),
	)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

func helpLine(keys selectKeys) string {
	parts := []string{
		hintPair(keys.moveUp),
		hintPair(keys.moveDown),
		"/ filter",
		hintPair(keys.choose),
		hintPair(keys.quit),
	}
	return dimStyle.Render(strings.Join(parts, "  ·  "))
}

func hintPair(b key.Binding) string {
	k, d := b.Help().Key, b.Help().Desc
	return k + " " + d
}
