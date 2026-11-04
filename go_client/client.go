package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// The CLI handshake header. groundwork-cli sends this on every request, so the
// Go client sends it as well to be treated as an official client.
const (
	cliSecretHeader = "X-Cli-Secret"
	cliSecretKey    = "my_groundwork_secret_key"
)

// TicketInfo is the fake company ticket that wraps the CVE scenario.
type TicketInfo struct {
	Company     string `json:"company"`
	TicketID    string `json:"ticket_id"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Ticket is the payload returned by GET /{language}/ticket.
type Ticket struct {
	SessionID string     `json:"session_id"`
	Ticket    TicketInfo `json:"ticket"`
	BuggyCode string     `json:"buggy_code"`
	CveID     string     `json:"cve_id"`
	Package   string     `json:"package"`
	Language  string     `json:"language"`
}

// Hint is a progressive hint from GET /{session_id}/hint.
type Hint struct {
	Hint      string `json:"hint"`
	Index     int    `json:"index"`
	Total     int    `json:"total"`
	Exhausted bool   `json:"exhausted"` // all hints already given
}

// EvaluateRequest is the body of POST /evaluate.
type EvaluateRequest struct {
	SessionID   string `json:"session_id"`
	StudentCode string `json:"student_code"`
}

// Result is the review returned by POST /evaluate.
type Result struct {
	Score            int     `json:"score"`
	Verdict          string  `json:"verdict"`
	WhatTheyGotRight string  `json:"what_they_got_right"`
	WhatTheyMissed   string  `json:"what_they_missed"`
	ArjunMessage     string  `json:"arjun_message"`
	MentalModel      string  `json:"mental_model"`
	Attempts         int     `json:"attempts"`
	TimeElapsed      float64 `json:"time_elapsed"`
}

// Solution is the real patch from GET /{session_id}/solution.
type Solution struct {
	FixedCode string `json:"fixed_code"`
	CveID     string `json:"cve_id"`
	Package   string `json:"package"`
}

type apiClient struct {
	base string
	http *http.Client
}

func newAPIClient(base string) *apiClient {
	return &apiClient{
		base: strings.TrimRight(base, "/"),
		http: &http.Client{},
	}
}

// getJSON issues a GET and decodes the reply into out.
func (c *apiClient) getJSON(ctx context.Context, path string, query url.Values, timeout time.Duration, out any) error {
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set(cliSecretHeader, cliSecretKey)

	return c.do(req, timeout, out)
}

// postJSON issues a POST with a JSON body and decodes the reply into out.
func (c *apiClient) postJSON(ctx context.Context, path string, body any, timeout time.Duration, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(cliSecretHeader, cliSecretKey)

	return c.do(req, timeout, out)
}

func (c *apiClient) do(req *http.Request, timeout time.Duration, out any) error {
	client := c.http
	if timeout > 0 {
		client = &http.Client{Timeout: timeout}
	}

	res, err := client.Do(req)
	if err != nil {
		return apiError{reason: requestError(err)}
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return apiError{reason: fmt.Sprintf("Could not read the server response. (%v)", err)}
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		detail := strings.TrimSpace(string(body))
		if len(detail) > 240 {
			detail = detail[:240] + "…"
		}
		return apiError{reason: fmt.Sprintf("Server returned %s. %s", res.Status, detail)}
	}

	if err := json.Unmarshal(body, out); err != nil {
		return apiError{reason: fmt.Sprintf("Server sent unexpected data. (%v)", err)}
	}
	return nil
}

// apiError carries a short, user-facing explanation of a failed request.
type apiError struct {
	reason string
}

func (e apiError) Error() string { return e.reason }

func requestError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "context deadline exceeded"),
		strings.Contains(msg, "Timeout"),
		strings.Contains(msg, "Client.Timeout"):
		return "Server took too long to respond. The CVE pipeline can take up to a minute; try again."
	default:
		return "Cannot reach the GroundWork server. Check your internet connection and the server URL, then try again."
	}
}

func (c *apiClient) ticket(ctx context.Context, slug string) (Ticket, error) {
	var t Ticket
	q := url.Values{"language": {slug}}
	err := c.getJSON(ctx, "/"+slug+"/ticket", q, 75*time.Second, &t)
	if err == nil && strings.TrimSpace(t.SessionID) == "" {
		// The endpoint answered, but not with a ticket — surface it instead
		// of rendering a blank card.
		err = apiError{reason: "The server did not send a ticket payload. Check the server URL (it must point at groundwork-server)."}
	}
	return t, err
}

func (c *apiClient) hint(ctx context.Context, sessionID string) (Hint, error) {
	var h Hint
	err := c.getJSON(ctx, "/"+url.PathEscape(sessionID)+"/hint", nil, 20*time.Second, &h)
	return h, err
}

func (c *apiClient) solution(ctx context.Context, sessionID string) (Solution, error) {
	var s Solution
	err := c.getJSON(ctx, "/"+url.PathEscape(sessionID)+"/solution", nil, 20*time.Second, &s)
	return s, err
}

func (c *apiClient) evaluate(ctx context.Context, sessionID, studentCode string) (Result, error) {
	var r Result
	body := EvaluateRequest{SessionID: sessionID, StudentCode: studentCode}
	err := c.postJSON(ctx, "/evaluate", body, 90*time.Second, &r)
	return r, err
}
