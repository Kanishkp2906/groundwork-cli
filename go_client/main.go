package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	// defaultServerURL points at the production GroundWork server on Hugging Face
	defaultServerURL = "https://kanishkp2906-groundwork-server.hf.space"
	serverURLEnvKey  = "GROUNDWORK_SERVER_URL"
)

// Version info (set via ldflags at build time)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	server := flag.String(
		"server",
		firstNonEmpty(os.Getenv(serverURLEnvKey), defaultServerURL),
		"GroundWork server URL (default: $GROUNDWORK_SERVER_URL or "+defaultServerURL+")",
	)
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("GroundWork TUI %s (commit: %s, built: %s)\n", version, commit, date)
		return
	}

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
