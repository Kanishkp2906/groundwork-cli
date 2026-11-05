package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ── help wiring for the bubbles help component ───────────────────────────────

// ShortHelp is the single line shown while on the main screen.
func (a app) ShortHelp() []key.Binding {
	return []key.Binding{
		a.keys.viewCode,
		a.keys.openEditor,
		a.keys.yank,
		a.keys.hint,
		a.keys.submit,
		a.keys.newTicket,
		a.keys.help,
		a.keys.quit,
	}
}

// FullHelp is the multi-column help toggled with ?.
func (a app) FullHelp() [][]key.Binding {
	editorK := []key.Binding{editorKeymap.submit, editorKeymap.done}
	viewer := []key.Binding{
		a.codeViewer.keys.lineDown,
		a.codeViewer.keys.lineUp,
		a.codeViewer.keys.gotoTop,
		a.codeViewer.keys.gotoBottom,
		a.codeViewer.keys.halfDown,
		a.codeViewer.keys.halfUp,
		a.codeViewer.keys.search,
		a.codeViewer.keys.nextMatch,
		a.codeViewer.keys.first,
		a.codeViewer.keys.yank,
		a.codeViewer.keys.done,
	}
	return [][]key.Binding{
		{a.keys.viewCode, a.keys.openEditor, a.keys.yank, a.keys.hint, a.keys.submit, a.keys.newTicket, a.keys.quit},
		viewer,
		editorK,
	}
}

// ── root view ───────────────────────────────────────────────────────────────

func (a app) View() tea.View {
	if a.quit {
		return tea.NewView("")
	}

	var v tea.View

	switch a.screen {
	case screenSelect:
		v = tea.NewView(a.selectModel.view(a.width, a.height))

	case screenLoading:
		v = tea.NewView(a.loadingView())

	case screenMain:
		v = tea.NewView(a.mainView())

	case screenCode:
		v = tea.NewView(a.codeViewer.view())

	case screenEditor:
		v = tea.NewView(a.editor.view())

	case screenError:
		v = tea.NewView(a.errorView())

	default:
		v = tea.NewView("")
	}

	// Use alternate screen buffer for all screens (full-screen TUI)
	v.AltScreen = true
	return v
}

// Cursor places the terminal cursor in the editor when it is open.
func (a app) Cursor() *tea.Cursor {
	if a.screen != screenEditor {
		return nil
	}
	c := a.editor.cursor()
	if c == nil {
		return nil
	}
	// editor view = 1 header line above the textarea
	c.Y += 1
	return c
}

// loading view: full screen with logo + Points spinner + status text.
func (a app) loadingView() string {
	fetchingStyle := lipgloss.NewStyle().Foreground(textFG)
	line := fmt.Sprintf("%s %s", a.spinner.View(), fetchingStyle.Render("Fetching CVEs for "+a.lang+"..."))

	sub := dimStyle.Render("The CVE pipeline can take up to a minute — hold tight.")
	quitLine := dimStyle.Render("ctrl+c to cancel")

	// Use small logo for loading
	logo := smallLogo(a.width)

	// Build content
	content := strings.Join([]string{
		logo,
		"",
		"  " + line,
		"  " + sub,
		"",
		"  " + quitLine,
	}, "\n")

	// Left-align vertically in the terminal
	return lipgloss.Place(a.width, a.height, lipgloss.Left, lipgloss.Center, content)
}

// mainView: rendered document filling the screen, status, busy line and help.
func (a app) mainView() string {
	// Viewport takes most of the screen
	vpContent := a.vp.View()

	status := a.statusLine()

	var parts []string
	parts = append(parts, vpContent)
	parts = append(parts, status)

	if a.busy == busyEval {
		parts = append(parts, fmt.Sprintf("%s %s",
			a.spinner.View(), "Evaluating your fix... Arjun is reading the code."))
	}

	parts = append(parts, a.help.View(a))

	return strings.Join(parts, "\n")
}

func (a app) statusLine() string {
	if a.status == "" {
		return dimStyle.Render("  pick an action below · ? expands help")
	}
	if a.statusOK {
		return statusOKStyle.Render("  " + a.status)
	}
	return statusErrStyle.Render("  " + a.status)
}

func (a app) errorView() string {
	body := strings.Join([]string{
		fmt.Sprintf("%s ticket fetch failed for %s", "❌", a.lang),
		"",
		a.status,
		"",
		dimStyle.Render("  The server runs at the URL given by -server / $GROUNDWORK_SERVER_URL."),
		dimStyle.Render("  Default: http://localhost:8000 — start groundwork-server first."),
	}, "\n")

	card := errorCardStyle.Render(body)
	instructions := dimStyle.Render(strings.Join([]string{
		"enter retry", "b back", "q quit",
	}, "  ·  "))

	frame := lipgloss.JoinVertical(lipgloss.Center, card, "", instructions)
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, frame)
}
