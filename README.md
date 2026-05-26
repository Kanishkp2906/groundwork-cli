# GroundWork CLI

> Real CVE training for developers — fix production bugs directly in your terminal.

GroundWork is a terminal-based training tool that pulls real-world Common Vulnerabilities and Exposures (CVEs) from open-source projects and drops you into a simulated scenario. Read the Jira-style ticket, review the buggy code, write your fix, and test your security engineering skills against actual production patches.

## Features

* **Real-World Vulnerabilities:** Pulled directly from GitHub Advisories and Google's OSS-Fuzz database.
* **Simulated Environment:** Every scenario includes a realistic bug ticket, severity rating, and AI-generated contextual hints.
* **Instant Validation:** Submit your code to instantly verify if your logic matches the historical commit that fixed the CVE.
* **Frictionless:** No accounts, no logins, no web portals. Just install and run.

## Supported Languages

* Python
* JavaScript / Node.js
* Java
* C++

## Installation

GroundWork requires **macOS or Linux** (Windows is not natively supported due to strict Unix `tty` dependencies) and **Python 3.10+**.

```bash
pip install groundwork-cli

```

## Usage

Start a new training session by running:

```bash
groundwork

```

Select your target language from the interactive menu. Once a ticket is loaded, use the following interactive commands:

* `[h]` - Request a hint
* `[s]` - Submit your fixed code
* `[r]` - Reveal the real-world solution and mental model
* `[n]` - Start a new session
* `[q]` - Quit the application

## Configuration

By default, the CLI connects to the public GroundWork production server. If you are hosting your own GroundWork FastAPI backend for local development or an internal corporate network, you can override the target API using an environment variable:

```bash
export GROUNDWORK_URL="http://localhost:8000"
groundwork

```

## License

MIT
