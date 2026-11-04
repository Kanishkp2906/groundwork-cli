package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// editorKeymap is shared with app.go so both sides bind the same keys.
var editorKeymap = newEditorKeys()

// editor is the alternate-screen editor where the fix gets written. It starts
// preloaded with the buggy code so the fix can be edited in place.
type editor struct {
	ta     textarea.Model
	title  string
	active bool
	ready  bool // false until newEditor() ran; a zero textarea cannot be resized
}

func newEditor() editor {
	t := textarea.New()
	t.Prompt = ""
	t.Placeholder = "Your fix starts from the buggy code above — edit it until it's safe."
	t.ShowLineNumbers = true
	t.SetVirtualCursor(true) // use virtual cursor so Cursor() returns position

	s := t.Styles()
	s.Cursor.Color = lipgloss.Color("212")
	s.Cursor.Shape = tea.CursorBlock // block cursor like real terminal
	s.Cursor.Blink = true
	s.Focused.Base = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("170"))
	s.Blurred.Base = lipgloss.NewStyle().Border(lipgloss.HiddenBorder())
	s.Focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	s.Blurred.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	s.Focused.EndOfBuffer = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	s.Blurred.EndOfBuffer = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	t.SetStyles(s)

	// Allow normal editing keys: Tab for indent, Enter for newline, etc.
	t.KeyMap.DeleteWordBackward.SetEnabled(false) // let ctrl+w be ctrl+w
	t.KeyMap.Paste.SetEnabled(false)              // avoid terminal paste races
	// Add Tab key binding for indentation
	t.KeyMap.InsertNewline.SetEnabled(true) // ensure Enter works
	t.Blur()

	e := editor{ta: t}
	e.ready = true
	return e
}

func (e *editor) setFix(code string) {
	e.ta.SetValue(code)
}

// activate brings the editor up focused with the current fix.
func (e *editor) activate(title, code string) {
	e.title = title
	if strings.TrimSpace(e.ta.Value()) == "" && code != "" {
		e.setFix(code)
	}
	e.active = true
	e.ta.Focus()
}

func (e *editor) deactivate() {
	e.active = false
	e.ta.Blur()
}

// fix returns the buffer contents exactly as typed: real newlines, real indentation.
func (e *editor) fix() string {
	return strings.ReplaceAll(e.ta.Value(), "\r\n", "\n")
}

func (e *editor) resize(w, h int) {
	if !e.ready {
		return
	}
	// editor screen = 1 title line + textarea + 1 button line + 1 help line;
	// the textarea's own border takes 2 rows.
	e.ta.SetWidth(max(20, w))
	e.ta.SetHeight(max(4, h-4))
}

// cursor gives a terminal cursor position adjusted for the editor layout.
func (e *editor) cursor() *tea.Cursor {
	if !e.active {
		return nil
	}
	c := e.ta.Cursor()
	if c == nil {
		return nil
	}
	c.Y += 1 // title line above the textarea
	c.X += 1 // editor border
	return c
}

var buttonStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("16")).
	Background(lipgloss.Color("170")).
	Padding(0, 2).
	MarginTop(1).
	Bold(true)

func (e editor) view() string {
	title := headerBarStyle.Render(fmt.Sprintf(" ✎ %s ", strings.TrimSpace(e.title)))
	taView := e.ta.View()
	btnView := buttonStyle.Render("[ Ctrl+S to Submit ]")
	help := dimStyle.Render("  ctrl+s submit · esc back (fix kept) · tab indent · enter newline")
	return strings.Join([]string{title, taView, btnView, help}, "\n")
}
