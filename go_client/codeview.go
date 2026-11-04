package main

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// vpMatch is a /search hit: 0-based line plus column span.
type vpMatch struct {
	line     int
	colStart int
	colEnd   int
}

// codeViewer reads the buggy code with neovim-style motion keys.
type codeViewer struct {
	title string
	vp    viewport.Model
	keys  codeViewerKeys

	search     textinput.Model // prompt for /
	searching  bool            // eating keys for the search buffer
	query      string          // last accepted search pattern
	matches    []vpMatch
	matchIdx   int
	countBuf   string // pending numeric count, e.g. 12j
	pendingG   bool   // waiting for the second g of gg
	statusHint string

	width, height int
	ready         bool // false until newCodeViewer() ran
}

func newCodeViewer(title, code string) codeViewer {
	v := codeViewer{keys: newCodeViewerKeys()}

	in := textinput.New()
	in.Prompt = "/"
	in.Placeholder = "pattern (enter to jump, esc to cancel)"
	in.CharLimit = 256
	v.search = in

	v.vp = viewport.New()
	v.vp.SetHorizontalStep(2)
	v.vp.HighlightStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("214")).
		Foreground(lipgloss.Color("16"))

	v.rollIn(title, code)
	v.ready = true
	return v
}

// rollIn loads (or reloads) a code buffer, reset to the top.
func (v *codeViewer) rollIn(title, code string) {
	v.title = title
	code = strings.ReplaceAll(code, "\r\n", "\n")

	v.vp.SetYOffset(0)
	v.vp.SetXOffset(0)
	v.vp.SetContentLines(strings.Split(code, "\n"))
	v.vp.LeftGutterFunc = codeGutter

	v.query = ""
	v.matches = nil
	v.matchIdx = 0
	v.countBuf = ""
	v.pendingG = false
	v.searching = false
	v.search.Blur()
	v.search.SetValue("")
	v.statusHint = ""
}

func codeGutter(c viewport.GutterContext) string {
	if c.Soft {
		return "      │ "
	}
	if c.Index >= c.TotalLines {
		return "      │ "
	}
	return fmt.Sprintf("%5d │ ", c.Index+1)
}

func (v *codeViewer) resize(w, h int) {
	if !v.ready {
		return
	}
	v.width, v.height = w, h
	v.vp.SetWidth(max(20, w))
	v.vp.SetHeight(max(3, h-2))
	v.search.SetWidth(max(10, w-6))
}

// update consumes messages; the bool reports "go back to the main screen".
func (v codeViewer) update(msg tea.Msg) (codeViewer, bool) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		return v.handleKey(keyMsg)
	}
	return v, false
}

// handleKey implements a pragmatic slice of neovim's normal mode:
//
//	j/k g/G gg 0 $ ^ /search n N ctrl+d/u ctrl+f/b h/l counts  q/Esc exit
func (v codeViewer) handleKey(msg tea.KeyPressMsg) (codeViewer, bool) {
	if v.searching {
		switch msg.String() {
		case "enter":
			v.acceptSearch()
		case "esc", "ctrl+c":
			v.searching = false
			v.search.Blur()
			v.search.SetValue("")
			v.statusHint = "search cancelled"
		default:
			in, _ := v.search.Update(msg)
			v.search = in
		}
		return v, false
	}

	s := msg.String()

	// 0 only becomes zero motion when digits are already queued.
	switch {
	case s >= "1" && s <= "9", s == "0" && v.countBuf != "":
		if len(v.countBuf) < 5 {
			v.countBuf += s
			v.statusHint = ""
		}
		return v, false

	case keyMatches(msg, v.keys.search):
		v.searching = true
		v.search.SetValue("")
		v.search.Focus()
		v.statusHint = ""
		return v, false

	case keyMatches(msg, v.keys.nextMatch):
		v.jumpMatch(+1)

	case keyMatches(msg, v.keys.first):
		v.jumpMatch(-1)

	case keyMatches(msg, v.keys.lineDown):
		n := v.takeCount()
		v.vp.ScrollDown(n)

	case keyMatches(msg, v.keys.lineUp):
		v.vp.ScrollUp(v.takeCount())

	case keyMatches(msg, v.keys.left):
		v.vp.ScrollLeft(v.takeCount() * 4)

	case keyMatches(msg, v.keys.right):
		v.vp.ScrollRight(v.takeCount() * 4)

	case keyMatches(msg, v.keys.halfDown):
		v.halfScroll(v.takeCount())

	case keyMatches(msg, v.keys.halfUp):
		v.halfScroll(-v.takeCount())

	case keyMatches(msg, v.keys.pageDown):
		n := v.takeCount()
		for range n {
			v.vp.PageDown()
		}

	case keyMatches(msg, v.keys.pageUp):
		n := v.takeCount()
		for range n {
			v.vp.PageUp()
		}

	case s == "g" && v.pendingG:
		v.takeCount()
		v.pendingG = false
		v.vp.GotoTop()

	case s == "g":
		v.pendingG = true // wait for the closing g of gg

	case s == "G":
		v.takeCount()
		v.vp.GotoBottom()

	case keyMatches(msg, v.keys.lineStart):
		v.takeCount()
		v.vp.SetXOffset(0)

	case keyMatches(msg, v.keys.lineEnd):
		v.takeCount()
		v.vp.SetXOffset(1_000_000) // viewport clamps internally

	case s == "esc" || s == "q":
		v.takeCount()
		v.pendingG = false
		return v, true

	default:
		// any other key clears pending state
		v.takeCount()
	}

	if s != "g" && v.pendingG {
		// pending g swallowed by another motion
		v.pendingG = false
	}
	return v, false
}

// takeCount converts pending digits into a rep count (default 1) and clears
// them, mirroring how vim consumes a count before a motion.
func (v *codeViewer) takeCount() int {
	if v.countBuf == "" {
		return 1
	}
	n, err := strconv.Atoi(v.countBuf)
	v.countBuf = ""
	if err != nil || n < 1 {
		return 1
	}
	return min(n, 10_000)
}

// halfScroll emulates ctrl+d / ctrl+u: half a viewport per repeat.
func (v *codeViewer) halfScroll(n int) {
	half := max(1, v.vp.Height()/2)
	v.vp.SetYOffset(v.vp.YOffset() + n*half)
}

// jumpMatch cycles through the accepted /search hits, wrapping around.
func (v *codeViewer) jumpMatch(step int) {
	if v.query != "" && len(v.matches) == 0 {
		// query set but produced nothing: re-run cheaply, still nothing.
		v.runSearch()
	}
	if len(v.matches) == 0 {
		v.statusHint = "/" + v.query + " · no matches"
		return
	}
	v.matchIdx = (v.matchIdx + step + len(v.matches)) % len(v.matches)
	if step > 0 {
		v.vp.HighlightNext()
	} else {
		v.vp.HighlightPrevious()
	}
	v.statusHint = fmt.Sprintf("/%s · match %d/%d", v.query, v.matchIdx+1, len(v.matches))
}

// acceptSearch commits the typed pattern (like pressing enter on /).
func (v *codeViewer) acceptSearch() {
	q := v.search.Value()
	v.searching = false
	v.search.Blur()
	v.search.SetValue("")
	if strings.TrimSpace(q) == "" {
		v.statusHint = "search cancelled"
		return
	}
	v.query = q
	v.runSearch()
}

// runSearch rebuilds highlight ranges for v.query and jumps to the first hit
// at or below the current view (same rule the viewport uses internally).
func (v *codeViewer) runSearch() {
	if v.query == "" {
		return
	}
	content := v.vp.GetContent()
	lowerContent := strings.ToLower(content)
	pattern := strings.ToLower(v.query)

	var matches []vpMatch
	ranges := make([][]int, 0, 64)
	off := 0
	for {
		idx := strings.Index(lowerContent[off:], pattern)
		if idx < 0 {
			break
		}
		byteStart := off + idx
		byteEnd := byteStart + len(pattern)
		line := strings.Count(content[:byteStart], "\n")
		lineStart := strings.LastIndexByte(content[:byteStart], '\n') + 1
		matches = append(matches, vpMatch{
			line:     line,
			colStart: byteStart - lineStart,
			colEnd:   byteEnd - lineStart,
		})
		ranges = append(ranges, []int{byteStart, byteEnd})
		off = byteEnd
	}

	v.matches = matches
	v.vp.ClearHighlights()
	if len(ranges) == 0 {
		v.matchIdx = 0
		v.statusHint = "/" + v.query + " · no matches"
		return
	}

	v.vp.SetHighlights(ranges)

	// Mirror viewport.findNearestMatch: the first hit at/after YOffset.
	found := -1
	for i, m := range matches {
		if m.line >= v.vp.YOffset() {
			found = i
			break
		}
	}
	if found == -1 {
		found = len(matches) - 1
	}
	v.matchIdx = found
	v.statusHint = fmt.Sprintf("/%s · match %d/%d", v.query, v.matchIdx+1, len(matches))
}

func (v codeViewer) view() string {
	header := headerBarStyle.Render(fmt.Sprintf(" ■ %s ", strings.TrimSpace(v.title)))
	body := v.vp.View()
	return strings.Join([]string{header, body, v.statusLine()}, "\n")
}

func (v codeViewer) statusLine() string {
	if v.searching {
		return searchPromptStyle.Render(v.search.View())
	}

	helpLine := dimStyle.Render(
		"  ·  j/k scroll · ^d/^u ½ page · gg/G goto · /search · n matches · q back")

	if v.statusHint != "" {
		return statusOKStyle.Render("  "+v.statusHint) + helpLine
	}

	lineNo := v.vp.YOffset() + 1
	total := v.totalLines()
	percent := 100
	if total > 0 {
		percent = (lineNo * 100) / max(1, total)
	}
	if v.vp.AtBottom() {
		percent = 100
	}
	return statusOKStyle.Render(fmt.Sprintf("  L%d · %d lines · %d%%", lineNo, total, percent)) + helpLine
}

func (v codeViewer) totalLines() int {
	return v.vp.TotalLineCount()
}
