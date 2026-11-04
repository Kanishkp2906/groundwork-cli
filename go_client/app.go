package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// screens the app walks through.
type screen int

const (
	screenSelect  screen = iota // language picker with the logo
	screenLoading               // spinner while the ticket is fetched
	screenMain                  // ticket + buggy code + submit/hints
	screenCode                  // code reader with neovim keys
	screenEditor                // alternate-screen editor
	screenError                 // request failure card
)

type busyKind int

const (
	busyNone   busyKind = iota
	busyTicket          // fetching a fresh ticket
	busyEval            // grading a submitted fix
)

// messages emitted by network commands.
type (
	ticketLoadedMsg   struct{ t Ticket }
	hintLoadedMsg     struct{ h Hint }
	evalDoneMsg       struct{ r Result }
	fetchErrMsg       struct{ reason string }
	chooseLanguageMsg struct{ lang languageOption }
	copiedMsg         struct {
		bytes        int
		lines        int
		clipboardErr error
	}
)

type app struct {
	width, height int

	client  *apiClient
	spinner spinner.Model
	help    help.Model
	keys    mainWindowKeys

	screen screen

	selectModel selectModel
	codeViewer  codeViewer
	editor      editor

	// ticket state
	lang     string // display label, e.g. "Python"
	slug     string // request value, e.g. "python"
	ticket   *Ticket
	hints    []Hint
	result   *Result
	busy     busyKind
	status   string // transient one-line message
	statusOK bool   // style the status line as success
	vp       viewport.Model
	bodyMD   string // glamour output; "" marks it stale
	fixDirty bool   // editor buffer differs from the original buggy code
	confirm  bool   // armed after warning about an unchanged fix

	quit bool
}

func newApp(c *apiClient) app {
	h := help.New()
	h.Styles = help.DefaultStyles(true)
	return app{
		client:      c,
		spinner:     newPointsSpinner(),
		help:        h,
		keys:        newMainWindowKeys(),
		screen:      screenSelect,
		selectModel: newSelectModel(),
		vp:          viewport.New(),
	}
}

func newPointsSpinner() spinner.Model {
	return spinner.New(
		spinner.WithSpinner(spinner.Points), // the requested Points spinner, nothing else
		spinner.WithStyle(spinnerStyle),
	)
}

func (a app) Init() tea.Cmd { return nil }

// ── update loop ────────────────────────────────────────────────────────────

func (a *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if km, ok := msg.(tea.KeyPressMsg); ok && km.String() == "ctrl+c" {
		a.quit = true
		return a, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		// markdown wrapping depends on width → re-render
		a.resize()
		a.invalidateBody()

	case tea.KeyPressMsg:
		switch a.screen {
		case screenSelect:
			if cmd := a.handleSelect(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		case screenLoading:
			// wait; ctrl+c handled above
		case screenMain:
			if cmd := a.handleMain(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		case screenCode:
			if cmd := a.handleCode(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		case screenEditor:
			if cmd := a.handleEditor(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		case screenError:
			if cmd := a.handleError(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case spinner.TickMsg:
		if a.busy != busyNone {
			var cmd tea.Cmd
			a.spinner, cmd = a.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case chooseLanguageMsg:
		a.startLoading(msg.lang)
		cmds = append(cmds, a.spinner.Tick, fetchTicketCmd(a.client, msg.lang.Slug))

	case ticketLoadedMsg:
		a.applyTicket(msg.t)

	case hintLoadedMsg:
		a.busy = busyNone
		a.hints = append(a.hints, msg.h)
		a.invalidateBody()
		if a.result == nil && len(a.hints) > 0 {
			// Hints land at the end of the doc while there is no review yet.
			a.vp.GotoBottom()
		}
		if msg.h.Exhausted {
			a.status = "Arjun 🔥: no more hints — commit to your fix."
			a.statusOK = false
		} else {
			a.status = fmt.Sprintf("Hint %d of %d added.", msg.h.Index, msg.h.Total)
			a.statusOK = true
		}

	case evalDoneMsg:
		a.result = &msg.r
		a.busy = busyNone
		a.invalidateBody()
		a.vp.GotoBottom() // the review lands at the end of the doc
		a.status = fmt.Sprintf("Review ready — %s · %d/100", msg.r.Verdict, msg.r.Score)
		a.statusOK = strings.EqualFold(msg.r.Verdict, "PASS") || strings.EqualFold(msg.r.Verdict, "PARTIAL")

	case copiedMsg:
		switch {
		case msg.clipboardErr != nil:
			a.status = fmt.Sprintf("Yanked %d bytes ✂️ sent via OSC52 (system clipboard: %v)", msg.bytes, msg.clipboardErr)
		default:
			a.status = fmt.Sprintf("Yanked %d bytes · %d lines ✂️ copied exactly (indentation + newlines kept).", msg.bytes, msg.lines)
		}
		a.statusOK = true

	case fetchErrMsg:
		a.busy = busyNone
		a.status = msg.reason
		a.statusOK = false
		if a.ticket == nil {
			// No ticket in hand: show the failure card
			a.screen = screenError
		}
	}

	return a, tea.Batch(cmds...)
}

func (a *app) handleSelect(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case keyMatches(msg, a.selectModel.keys.quit):
		a.quit = true
		return tea.Quit
	default:
		m, cmd := a.selectModel.update(msg)
		a.selectModel = m
		return cmd
	}
}

func (a *app) handleError(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		a.quit = true
		return tea.Quit
	case "enter":
		for _, lang := range languageOptions {
			if lang.Slug == a.slug {
				a.startLoading(lang)
				return tea.Batch(a.spinner.Tick, fetchTicketCmd(a.client, a.slug))
			}
		}
		a.screen = screenSelect
		return nil
	case "b":
		a.screen = screenSelect
	}
	return nil
}

func (a *app) handleMain(msg tea.KeyPressMsg) tea.Cmd {
	if a.busy != busyNone {
		return nil
	}
	switch {
	case keyMatches(msg, a.keys.help):
		a.help.ShowAll = !a.help.ShowAll
	case keyMatches(msg, a.keys.viewCode):
		a.codeViewer.rollIn(a.codeTitle(), a.ticket.BuggyCode)
		a.screen = screenCode
	case keyMatches(msg, a.keys.openEditor):
		a.editor.activate(a.codeTitle(), a.editorFix())
		a.screen = screenEditor
	case keyMatches(msg, a.keys.yank):
		return copyTicketCmd(a.ticket.BuggyCode)
	case keyMatches(msg, a.keys.hint):
		a.status = "Arjun is thinking…"
		a.statusOK = true
		return fetchHintCmd(a.client, a.ticket.SessionID)
	case keyMatches(msg, a.keys.submit):
		return a.submitFix()
	case keyMatches(msg, a.keys.newTicket):
		a.pickNewTicket()
	case keyMatches(msg, a.keys.quit):
		a.quit = true
		return tea.Quit
	default:
		// fall through: viewport scroll keys
		m, cmd := a.vp.Update(msg)
		a.vp = m
		if cmd != nil {
			return cmd
		}
	}
	return nil
}

func (a *app) handleCode(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case keyMatches(msg, a.codeViewer.keys.yank):
		return copyTicketCmd(a.ticket.BuggyCode)
	}
	m, back := a.codeViewer.update(msg)
	a.codeViewer = m
	if back {
		a.screen = screenMain
	}
	return nil
}

func (a *app) handleEditor(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case keyMatches(msg, editorKeymap.done):
		a.editor.deactivate()
		a.screen = screenMain
		a.status = "Fix kept — press s to submit it."
		a.statusOK = true
	case keyMatches(msg, editorKeymap.submit):
		a.editor.deactivate()
		a.screen = screenMain
		return a.submitFix()
	case msg.String() == "tab":
		// Tab inserts indentation (4 spaces)
		a.editor.ta.InsertString("    ")
		a.fixDirty = true
		a.confirm = false
		return nil
	default:
		// All other keys (including Enter for newline) go to textarea
		ta, cmd := a.editor.ta.Update(msg)
		a.editor.ta = ta
		// Any real change disarms the unchanged-fix warning.
		if strings.TrimSpace(a.editor.fix()) != strings.TrimSpace(a.editorFix()) {
			a.fixDirty = true
		}
		a.confirm = false
		return cmd
	}
	return nil
}

func (a *app) submitFix() tea.Cmd {
	if a.ticket == nil {
		return nil
	}
	if strings.TrimSpace(a.editor.fix()) == "" {
		a.status = "Write your fix first — press e to open the editor."
		a.statusOK = false
		return nil
	}
	// Guard against fat-thumbing s while the editor still holds the untouched
	// buggy code: warn once, submit on the next press.
	if !a.fixDirty && !a.confirm {
		a.confirm = true
		a.status = "The fix is still the original buggy code. Press s again to submit it as-is, or e to edit."
		a.statusOK = false
		return nil
	}
	a.busy = busyEval
	a.fixDirty = true
	a.confirm = false
	a.status = ""
	a.spinner = newPointsSpinner()
	return tea.Batch(a.spinner.Tick, evaluateFixCmd(a.client, a.ticket.SessionID, a.editor.fix()))
}

// ── transitions ────────────────────────────────────────────────────────────

func (a *app) startLoading(lang languageOption) {
	a.lang = lang.Label
	a.slug = lang.Slug
	a.screen = screenLoading
	a.busy = busyTicket
	a.ticket = nil
	a.hints = nil
	a.result = nil
	a.bodyMD = ""
	a.status = ""
	a.spinner = newPointsSpinner()
}

func (a *app) codeTitle() string {
	return fmt.Sprintf("%s · %s · pkg %s", a.lang, a.ticket.CveID, a.ticket.Package)
}

func (a *app) editorFix() string {
	if a.ticket == nil {
		return ""
	}
	return strings.ReplaceAll(a.ticket.BuggyCode, "\r\n", "\n")
}

func (a *app) applyTicket(t Ticket) {
	a.ticket = &t
	a.busy = busyNone
	a.screen = screenMain
	a.hints = nil
	a.result = nil
	a.fixDirty = false
	a.confirm = false
	a.status = fmt.Sprintf("Ticket %s ready — %s", t.Ticket.TicketID, t.CveID)
	a.statusOK = true
	a.editor = newEditor()
	a.editor.setFix(a.editorFix())
	a.codeViewer = newCodeViewer(a.codeTitle(), t.BuggyCode)
	a.resize()
	a.invalidateBody()
}

func (a *app) pickNewTicket() {
	a.ticket = nil
	a.hints = nil
	a.result = nil
	a.bodyMD = ""
	a.status = ""
	a.busy = busyNone
	a.fixDirty = false
	a.confirm = false
	a.selectModel.quitting = false
	a.screen = screenSelect
}

func (a *app) resize() {
	if a.width <= 0 || a.height <= 0 {
		return
	}
	reserved := 2 // status + help
	if a.busy == busyEval {
		reserved++
	}
	a.vp.SetWidth(a.width)
	a.vp.SetHeight(max(3, a.height-reserved))
	a.codeViewer.resize(a.width, a.height)
	a.editor.resize(a.width, a.height)
}

// invalidateBody marks the rendered document stale and rebuilds it when the
// pieces are available.
func (a *app) invalidateBody() {
	a.bodyMD = ""
	if a.ticket == nil {
		return
	}
	var b strings.Builder
	b.WriteString(ticketMarkdown(a.ticket))
	hintMarkdown(&b, a.hints)
	if a.result != nil {
		resultMarkdown(&b, *a.result)
	}
	md, err := renderMarkdown(b.String(), a.width)
	if err != nil {
		a.status = "❌ " + err.Error()
		a.statusOK = false
		return
	}
	a.bodyMD = md
	a.vp.SetContent(a.bodyMD)
}
