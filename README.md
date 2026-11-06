# GroundWork TUI

> **Real CVE Training for Developers** — A terminal UI for practicing real CVE fixes in your terminal.

![GroundWork TUI Demo](demo.gif)

## 🚀 One-Line Install

### Linux / macOS
```bash
curl -fsSL https://raw.githubusercontent.com/kanishkp/groundwork/main/install.sh | bash
```

### Windows (PowerShell)
```powershell
iwr -useb https://raw.githubusercontent.com/kanishkp/groundwork/main/install.ps1 | iex
```

### Manual Download
Download the latest release for your platform from [Releases](https://github.com/kanishkp/groundwork/releases).

---

## 🎯 What is GroundWork?

GroundWork is a terminal-based training platform where you fix real CVEs (Common Vulnerabilities and Exposures) in a realistic terminal environment. 

- **Real vulnerabilities** — Based on actual CVEs from OSV.dev
- **Real code** — Buggy code patterns from real GitHub commits
- **Real feedback** — AI-powered code review with detailed explanations
- **Multiple languages** — Python, JavaScript, Java, C++

---

## 🎮 How It Works

1. **Pick a language** — Choose Python, JavaScript, Java, or C++
2. **Get a ticket** — Receive a realistic Jira-style ticket with a real CVE
3. **Read the code** — View the buggy code with syntax highlighting
4. **Write your fix** — Edit in a full-featured editor (Ctrl+S to submit)
5. **Get reviewed** — AI senior engineer reviews your fix and explains what you got right/missed

---

## ⌨️ Key Bindings

### Main Screen
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

### Code Viewer (Vim keys)
| Key | Action |
|-----|--------|
| `j`/`k` | Scroll down/up |
| `h`/`l` | Scroll left/right |
| `G`/`gg` | Go to bottom/top |
| `/` | Search |
| `n`/`N` | Next/previous match |
| `y` | Yank code |
| `q`/`Esc` | Back |

### Editor (Alt Screen)
| Key | Action |
|-----|--------|
| `Ctrl+S` | Submit fix |
| `Esc` | Back (fix kept) |
| `Tab` | Indent |

---

## ⚙️ Configuration

### Environment Variables
```bash
# Required: Point to your GroundWork server
export GROUNDWORK_SERVER_URL=http://localhost:8000

# Optional: Override default server
groundwork -server https://your-server.com
```

### Server Setup
You need a running GroundWork server. The easiest way:

```bash
# Using Docker (recommended)
docker run -d -p 8000:8000 -e UPSTASH_REDIS_URL=your_redis_url \
  ghcr.io/kanishkp/groundwork:latest

# Or run locally
cd groundwork-server
pip install -r requirements.txt
python server.py
```

---

## 🏗️ Building from Source

### Prerequisites
- Go 1.21+
- Groq API key (for AI generation/evaluation)

```bash
# Clone and build
git clone https://github.com/kanishkp/groundwork.git
cd groundwork/go_client
go build -o groundwork .
./groundwork
```

### Cross-compile for all platforms
```bash
# Requires goreleaser
goreleaser build --snapshot --clean
```

---

## 🔑 Required API Keys

Create `.env` in the server directory:
```bash
GROQ_API_KEY=your_groq_api_key
UPSTASH_REDIS_URL=your_upstash_redis_url
NVIDIA_API_KEY=your_nvidia_api_key  # Optional, for Nemotron evaluator
```

Get keys from:
- [Groq Console](https://console.groq.com/) — Fast, free LLM access
- [Upstash](https://upstash.com/) — Free Redis for caching
- [NVIDIA NGC](https://ngc.nvidia.com/) — Optional, for Nemotron evaluator

---

## 📦 Distribution

### Homebrew (macOS/Linux)
```bash
brew tap kanishkp/tap
brew install groundwork
```

### Scoop (Windows)
```powershell
scoop bucket add kanishkp https://github.com/kanishkp/scoop-bucket
scoop install groundwork
```

### Chocolatey (Windows)
```powershell
choco install groundwork
```

### Docker
```bash
docker run -it --rm ghcr.io/kanishkp/groundwork:latest
```

### AUR (Arch Linux)
```bash
yay -S groundwork
```

---

## 🛠️ Development

### Project Structure
```
groundwork/
├── go_client/           # TUI client (Go + Bubble Tea)
│   ├── main.go          # Entry point
│   ├── app.go           # Main app logic
│   ├── client.go        # HTTP client
│   ├── editor.go        # Editor component
│   ├── codeview.go      # Vim-style code viewer
│   ├── select.go        # Language selector
│   └── ...
├── groundwork-server/   # Python FastAPI server
│   ├── server.py        # FastAPI app
│   ├── services/
│   │   ├── new_fetcher.py   # CVE fetching + scenario generation
│   │   └── evaluator.py     # Code evaluation
│   └── ...
├── .goreleaser.yaml     # Release config
├── install.sh           # Linux/macOS installer
└── install.ps1          # Windows installer
```

### Running Tests
```bash
# Go client
cd go_client && go test ./...

# Server
cd groundwork-server && python -m pytest
```

---

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `go test ./...` and `python -m pytest`
5. Submit a PR

---

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Groq](https://groq.com/) — Fast LLM inference
- [OSV.dev](https://osv.dev/) — CVE database
- [Upstash](https://upstash.com/) — Redis caching

---

**GroundWork** — Making CVE fixing a daily habit, not a panic response. 🛡️