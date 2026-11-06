# GroundWork CLI

> **Real CVE Training for Developers** — Fix production bugs in your terminal.

A terminal-based training platform where you fix real CVEs (Common Vulnerabilities and Exposures) in a realistic environment.

## Features

- **Real vulnerabilities** — Based on actual CVEs from OSV.dev
- **Real code** — Buggy code patterns from real GitHub commits
- **Real feedback** — AI-powered code review with detailed explanations
- **Multiple languages** — Python, JavaScript, Java, C++

## Installation

### Quick Install (Linux/macOS)
```bash
curl -fsSL https://raw.githubusercontent.com/kanishkP2906/groundwork-cli/main/install.sh | bash
```

### Windows (PowerShell)
```powershell
iwr -useb https://raw.githubusercontent.com/kanishkp/groundwork/main/install.ps1 | iex
```

### Via pipx (recommended)
```bash
pipx install groundwork-cli
```

### Via pip
```bash
pip install groundwork-cli
```

### From Source
```bash
git clone https://github.com/kanishkP2906/groundwork-cli.git
cd groundwork-cli
pip install -e .
```

## Usage

```bash
# Run the TUI
groundwork

# With custom server URL
groundwork --server https://your-server.com

# Show version
groundwork --version
```

## How It Works

1. **Pick a language** — Choose Python, JavaScript, Java, or C++
2. **Get a ticket** — Receive a realistic Jira-style ticket with a real CVE
3. **Read the code** — View the buggy code with syntax highlighting
4. **Write your fix** — Edit in a full-featured editor (Ctrl+S to submit)
4. **Get reviewed** — AI senior engineer reviews your fix and explains what you got right/missed

## Key Bindings

| Key | Action |
|-----|--------|
| `v` | View code (vim keys) |
| `e` | Open editor (alt screen) |
| `y` | Yank exact code to clipboard |
| `h` | Get a hint |
| `s` | Submit fix |
| `n` | New ticket |
| `?` | Full help |
| `q` | Quit |

## Configuration

```bash
# Required: Point to your GroundWork server
export GROUNDWORK_SERVER_URL=https://your-server.com

# Or pass directly
groundwork --server https://your-server.com
```

## Requirements

- Python 3.10+
- A running GroundWork server (local or remote)

## Development

```bash
# Install in development mode
pip install -e .

# Run tests
pytest

# Format code
black .
ruff check .

# Build package
python -m build
```

## License

MIT License — see [LICENSE](LICENSE) for details.