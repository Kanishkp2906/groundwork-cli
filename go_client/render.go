package main

import (
	"fmt"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
)

// glamourDocStyle frames the rendered markdown inside the viewport.
var glamourDocStyle = lipgloss.NewStyle().Padding(0, 1)

// glamourWrapWidth returns the word wrap target used for markdown rendering.
func glamourWrapWidth(termWidth int) int {
	if termWidth <= 0 {
		return 80
	}
	return max(60, min(termWidth-4, 110))
}

func newRenderer(termWidth int) (*glamour.TermRenderer, error) {
	return glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(glamourWrapWidth(termWidth)),
	)
}

// severityFlag picks the emoji groundwork-cli uses for a severity.
func severityFlag(severity string) string {
	switch strings.ToUpper(severity) {
	case "CRITICAL", "HIGH":
		return "🔴"
	case "MODERATE", "MEDIUM":
		return "🟠"
	case "LOW":
		return "🟡"
	default:
		return "🔴"
	}
}

// codeFence selects a fence that does not clash with the code's contents.
func codeFence(code string) string {
	if strings.Contains(code, "```") {
		return "~~~"
	}
	return "```"
}

// ticketMarkdown builds the main-screen document for the freshly fetched
// ticket. Hints and the review after a submit get appended further down.
func ticketMarkdown(t *Ticket) string {
	var b strings.Builder

	codeLang := strings.ToLower(t.Language)
	if codeLang == "c++" {
		codeLang = "cpp"
	}
	if codeLang == "" {
		codeLang = "text"
	}

	fmt.Fprintf(&b, "# 🎫 %s — %s\n\n", t.Ticket.Company, t.Ticket.TicketID)
	fmt.Fprintf(&b, "%s Severity: **%s**  ·  📦 CVE: **%s**  ·  Package: `%s`\n\n",
		severityFlag(t.Ticket.Severity), t.Ticket.Severity, t.CveID, t.Package)
	fmt.Fprintf(&b, "> **%s**\n\n", strings.TrimSpace(t.Ticket.Title))
	b.WriteString(strings.TrimSpace(t.Ticket.Description))
	b.WriteString("\n\n## 🐛 Buggy code\n\n")
	// Keep the code in a fenced block with its original indentation and line
	// breaks so glamour highlights it without mangling the source.
	fmt.Fprintf(&b, "%s%s\n%s\n%s\n", codeFence(t.BuggyCode), codeLang, t.BuggyCode, codeFence(t.BuggyCode))
	return b.String()
}

// hintMarkdown appends the progressive hints as an ordered list.
func hintMarkdown(b *strings.Builder, hints []Hint) {
	if len(hints) == 0 {
		return
	}
	b.WriteString("\n---\n\n## 💡 Arjun's hints\n\n")
	for _, h := range hints {
		if h.Exhausted {
			fmt.Fprintf(b, "* 🔥 *No more hints. You've got this.*\n")
			continue
		}
		fmt.Fprintf(b, "* **%d/%d** — %s\n", h.Index, h.Total, strings.TrimSpace(h.Hint))
	}
}

// resultMarkdown appends the review verdict after an evaluate call.
func resultMarkdown(b *strings.Builder, r Result) {
	emoji := "❌"
	switch strings.ToUpper(r.Verdict) {
	case "PASS":
		emoji = "✅"
	case "PARTIAL":
		emoji = "⚡"
	}
	totalSecs := int(r.TimeElapsed)
	mins, secs := totalSecs/60, totalSecs%60

	b.WriteString("\n---\n\n")
	fmt.Fprintf(b, "## %s Review — **%s** · **%d**/100\n\n", emoji, r.Verdict, r.Score)
	fmt.Fprintf(b, "⏱ %dm %ds · attempts: %d\n\n", mins, secs, max(1, r.Attempts))
	fmt.Fprintf(b, "> **Arjun 🔥:** %s\n\n", strings.TrimSpace(r.ArjunMessage))
	fmt.Fprintf(b, "**✅ Got right:** %s\n\n", strings.TrimSpace(r.WhatTheyGotRight))
	fmt.Fprintf(b, "**❌ Missed:** %s\n\n", strings.TrimSpace(r.WhatTheyMissed))
	fmt.Fprintf(b, "**🧠 Mental model:** %s\n", strings.TrimSpace(r.MentalModel))
}

// renderMarkdown turns markdown into an ANSI string through glamour.
func renderMarkdown(md string, termWidth int) (string, error) {
	r, err := newRenderer(termWidth)
	if err != nil {
		return "", err
	}
	out, err := r.Render(md)
	if err != nil {
		return "", err
	}
	return glamourDocStyle.Render(strings.TrimRight(out, "\n")), nil
}
