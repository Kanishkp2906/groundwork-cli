package main

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
)

// chooseLanguageCmd tells the app the user picked a language on the spinnerless
// main screen.
func chooseLanguageCmd(lang languageOption) tea.Cmd {
	return func() tea.Msg { return chooseLanguageMsg{lang: lang} }
}

// fetchTicketCmd requests GET /{language}/ticket.
func fetchTicketCmd(c *apiClient, slug string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
		defer cancel()

		t, err := c.ticket(ctx, slug)
		if err != nil {
			return fetchErrMsg{reason: err.Error()}
		}
		return ticketLoadedMsg{t}
	}
}

// fetchHintCmd requests the next hint for the current session.
func fetchHintCmd(c *apiClient, sessionID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		h, err := c.hint(ctx, sessionID)
		if err != nil {
			return fetchErrMsg{reason: err.Error()}
		}
		return hintLoadedMsg{h}
	}
}

// evaluateFixCmd grades the fix with POST /evaluate.
func evaluateFixCmd(c *apiClient, sessionID, studentCode string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 95*time.Second)
		defer cancel()

		r, err := c.evaluate(ctx, sessionID, studentCode)
		if err != nil {
			return fetchErrMsg{reason: err.Error()}
		}
		return evalDoneMsg{r}
	}
}

// copyTicketCmd copies the buggy code byte-for-byte. It writes the system
// clipboard and additionally sends OSC52 so it also works over SSH.
func copyTicketCmd(code string) tea.Cmd {
	copyBytes := code
	if !strings.HasSuffix(copyBytes, "\n") {
		// preserve the exact bytes, but make the final line selectable
		copyBytes = copyBytes + "\n"
	}
	lines := strings.Count(copyBytes, "\n")

	return tea.Batch(
		func() tea.Msg {
			err := clipboard.WriteAll(copyBytes)
			return copiedMsg{bytes: len(copyBytes), lines: lines, clipboardErr: err}
		},
		tea.SetClipboard(copyBytes),
	)
}
