package main

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// keyMatches wraps bubbletea's binding matcher for the shared key maps.
func keyMatches(msg tea.KeyPressMsg, b key.Binding) bool {
	return key.Matches(msg, b)
}

// selectKeys are the bindings on the language picker.
type selectKeys struct {
	moveUp, moveDown, choose, quit key.Binding
}

func newSelectKeys() selectKeys {
	return selectKeys{
		moveUp:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
		moveDown: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
		choose:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "pick & fetch")),
		quit:     key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// mainWindowKeys are the bindings on the ticket screen. The submit and hint
// keys asked for live here.
type mainWindowKeys struct {
	viewCode   key.Binding
	openEditor key.Binding
	yank       key.Binding
	hint       key.Binding
	submit     key.Binding
	newTicket  key.Binding
	quit       key.Binding
	help       key.Binding
}

func newMainWindowKeys() mainWindowKeys {
	return mainWindowKeys{
		viewCode:   key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "read code (vim keys)")),
		openEditor: key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "open editor (alt screen)")),
		yank:       key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yank exact code")),
		hint:       key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "get a hint")),
		submit:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "submit fix")),
		newTicket:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new ticket")),
		quit:       key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp("q", "quit")),
		help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more help")),
	}
}

// codeViewerKeys are the neovim-style motions for the code reader.
type codeViewerKeys struct {
	lineDown, lineUp, left, right      key.Binding
	halfDown, halfUp, pageDown, pageUp key.Binding
	gotoTop, gotoBottom                key.Binding
	lineStart, lineEnd                 key.Binding
	first, last                        key.Binding
	search, nextMatch, prevMatch       key.Binding
	yank, done                         key.Binding
}

func newCodeViewerKeys() codeViewerKeys {
	return codeViewerKeys{
		lineDown:   key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "down")),
		lineUp:     key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "up")),
		left:       key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h", "left")),
		right:      key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l", "right")),
		halfDown:   key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("^d", "½ down")),
		halfUp:     key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("^u", "½ up")),
		pageDown:   key.NewBinding(key.WithKeys("ctrl+f", "pgdown"), key.WithHelp("^f", "page down")),
		pageUp:     key.NewBinding(key.WithKeys("ctrl+b", "pgup"), key.WithHelp("^b", "page up")),
		gotoTop:    key.NewBinding(key.WithKeys("gg"), key.WithHelp("gg", "top")),
		gotoBottom: key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
		lineStart:  key.NewBinding(key.WithKeys("0", "^"), key.WithHelp("0", "line start")),
		lineEnd:    key.NewBinding(key.WithKeys("$"), key.WithHelp("$", "line end")),
		first:      key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "prev match")),
		search:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		nextMatch:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next match")),
		yank:       key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yank code")),
		done:       key.NewBinding(key.WithKeys("esc", "q"), key.WithHelp("esc", "back")),
	}
}

// editorKeys are the bindings for the alternate-screen editor.
type editorKeys struct {
	submit key.Binding
	done   key.Binding
}

func newEditorKeys() editorKeys {
	return editorKeys{
		submit: key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "submit fix")),
		done:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back (fix kept)")),
	}
}
