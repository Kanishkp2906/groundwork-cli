package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	// defaultServerURL points at a local server, per the existing go_client
	// main.go. The production endpoint used by groundwork-cli is
	// https://kanishkp2906-groundwork-server.hf.space
	defaultServerURL = "http://localhost:8000"
	serverURLEnvKey  = "GROUNDWORK_SERVER_URL"
)

func main() {
	server := flag.String(
		"server",
		firstNonEmpty(os.Getenv(serverURLEnvKey), defaultServerURL),
		"GroundWork server URL (default: $GROUNDWORK_SERVER_URL or "+defaultServerURL+")",
	)
	flag.Parse()

	if strings.TrimSpace(*server) == "" {
		*server = defaultServerURL
	}

	tui := newApp(newAPIClient(*server))
	p := tea.NewProgram(&tui)

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error running groundwork:", err)
		os.Exit(1)
	}

	fmt.Println("Exiting Groundwork. Keep fixing bugs! 👋")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
