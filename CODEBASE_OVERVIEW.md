# GroundWork - Codebase Overview

## Problem Statement

**GroundWork** is a developer training platform that teaches security best practices through **real-world CVE (Common Vulnerabilities and Exposures) scenarios**. Instead of artificial or contrived examples, developers learn by fixing actual production vulnerabilities that have affected well-known Python packages.

### Core Problem It Solves

1. **Security knowledge gap**: Junior (and even senior) developers often don't understand security vulnerabilities until they cause a production incident
2. **Abstract learning**: Most security training uses fake examples that don't reflect real-world code patterns
3. **No hands-on practice**: Traditional security training is passive - watching videos or reading docs without actually writing fixes
4. **Context-switching**: Developers need to learn security concepts within their actual workflow (IDE + terminal), not in a separate LMS platform

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         USER INTERFACES                         │
├─────────────────────────┬───────────────────────────────────────┤
│   CLI Mode (cli.py)     │   MCP Server Mode (server.py)         │
│   - Terminal-based      │   - IDE integration (Claude, Cursor)  │
│   - Interactive prompts │   - Tools: get_ticket, evaluate_fix   │
│   - Paste-based submit  │   - Session management                │
└───────────┬─────────────┴────────────────┬──────────────────────┘
            │                              │
            ▼                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      CORE SERVICES LAYER                        │
├─────────────────────────────────────────────────────────────────┤
│  fetcher.py                          evaluator.py               │
│  - Fetches live CVEs from OSV.dev   - Evaluates student fixes   │
│  - Extracts code from GitHub commits - Uses Gemini for semantic │
│  - Processes with Gemini AI          - No hardcoded tests       │
└─────────────────────────────────────────────────────────────────┘
            │                              │
            ▼                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      EXTERNAL APIS                              │
├─────────────────┬─────────────────────┬─────────────────────────┤
│   OSV.dev       │   GitHub API        │   Google Gemini AI      │
│ - CVE data      │   - Commit diffs    │   - Code generation     │
│ - Vulnerability │   - Fix commits     │   - Evaluation          │
│     details     │                     │                         │
└─────────────────┴─────────────────────┴─────────────────────────┘
```

---

## Data Flow Pipeline

### Phase 1: Fetching a Live CVE Scenario

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  get_ticket  │ ──► │  fetcher.py  │ ──► │   OSV.dev    │
│  (user call) │     │  get_live_   │     │   API        │
│              │     │  _scenario() │     │              │
└──────────────┘     └──────────────┘     └──────────────┘
                                                  │
                                                  ▼
     ┌──────────────────────────────────────────────────────┐
     │  Returns list of CVEs for popular Python packages:   │
     │  django, flask, requests, pillow, cryptography...    │
     └──────────────────────────────────────────────────────┘
                                                  │
                                                  ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Validate    │ ◄── │  extract_    │ ◄── │   GitHub     │
│  scenario    │     │  code_from_  │     │   API        │
│              │     │  commit()    │     │              │
└──────────────┘     └──────────────┘     └──────────────┘
                            │
                            ▼
                   Parse unified diff
                   Extract Python files
                   Skip tests/migrations
```

**Detailed Steps:**

1. **Query OSV.dev** (`fetch_python_cves()`):
   - Sends batch request for 15 popular Python packages
   - Gets list of CVEs with IDs
   - Fetches full vulnerability details for each

2. **Filter for GitHub commits** (`fetch_python_cves()`):
   - Only keeps CVEs that have a `fix_commit_url` pointing to GitHub
   - This ensures we can extract actual code changes

3. **Extract code from commit** (`extract_code_from_commit()`):
   - Converts browser URL → API URL:
     - `https://github.com/pallets/flask/commit/abc123`
     - `https://api.github.com/repos/pallets/flask/commits/abc123`
   - Fetches raw unified diff with `Accept: application/vnd.github.v3.diff`
   - Parses diff to extract:
     - Removed lines (buggy code, prefixed with `-`)
     - Added lines (fixed code, prefixed with `+`)
   - Filters to only `.py` files, skipping tests/migrations

4. **Process with Gemini** (`process_with_gemini()`):
   - Sends buggy + fixed code + CVE description to Gemini
   - Gemini returns structured JSON with:
     - `buggy_function`: Standalone runnable Python function with the vulnerability
     - `fixed_function`: Same function with the real fix applied
     - `ticket`: Simulated Jira ticket (company name, ID, severity, title, description)
     - `hints`: 3 progressive hints (vague → specific)
     - `mental_model`: Transferable lesson for the developer
     - `correct_pattern`: The specific function/pattern that must appear in fix

5. **Validate scenario** (`validate_scenario()`):
   - Checks all required fields exist
   - Verifies 3 hints present
   - Confirms `correct_pattern` is in fixed code but NOT in buggy code
   - Ensures both functions are meaningfully different
   - If validation fails, tries next CVE (up to 3 more)

---

### Phase 2: Student Interaction

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Student    │ ──► │   CLI or     │ ──► │  Display     │
│   calls      │     │   MCP        │     │  ticket +    │
│   get_ticket │     │   server     │     │  buggy code  │
└──────────────┘     └──────────────┘     └──────────────┘
                                                  │
                                                  ▼
     ┌──────────────────────────────────────────────────────┐
     │  Student reads ticket, studies buggy code, writes    │
     │  fix in their IDE (TRAE, Cursor, VS Code, etc.)      │
     └──────────────────────────────────────────────────────┘
                                                  │
                                                  ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Display     │ ◄── │  evaluate_   │ ◄── │  Student     │
│  results +   │     │  student_fix │     │  submits fix │
│  feedback    │     │  ()          │     │              │
└──────────────┘     └──────────────┘     └──────────────┘
```

**User Journey:**

1. **Get ticket**: User calls `get_ticket(difficulty="medium")`
2. **Read scenario**: Sees company name, ticket ID, severity, description, and buggy code
3. **Optional hints**: Can call `get_hint()` up to 3 times for progressive guidance
4. **Write fix**: Implements fix in their IDE
5. **Submit**: Calls `evaluate_fix(student_code)` with their complete fixed function
6. **Receive feedback**: Gets score, verdict, and detailed explanation from "Arjun" (AI persona)

---

### Phase 3: Evaluation Pipeline

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  evaluate_   │ ──► │  Build       │ ──► │   Gemini     │
│  student_fix │     │  prompt with │     │   API        │
│  ()          │     │  3 code      │     │              │
│              │     │  versions    │     │              │
└──────────────┘     └──────────────┘     └──────────────┘
                            │
                            ▼
     ┌──────────────────────────────────────────────────────┐
     │  Prompt includes:                                    │
     │  1. Original buggy function                          │
     │  2. Real GitHub fix (ground truth)                   │
     │  3. Student's submitted fix                          │
     │  4. CVE context and description                      │
     └──────────────────────────────────────────────────────┘
                            │
                            ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Return      │ ◄── │  Parse JSON  │ ◄── │  AI returns  │
│  structured  │     │  response    │     │  semantic    │
│  feedback    │     │              │     │  evaluation  │
└──────────────┘     └──────────────┘     └──────────────┘
```

**Evaluation Criteria (handled by Gemini):**

- **Semantic matching**: Not string comparison - understands if approach is correct even with different syntax
- **Root cause analysis**: Did they fix the underlying vulnerability or just patch symptoms?
- **Security correctness**: Would their fix actually prevent the attack described in the CVE?
- **Approach validation**: Is their solution in the same ballpark as the real fix?

**Output:**
```json
{
  "score": 85,
  "verdict": "PASS",
  "understood_vulnerability": true,
  "fixed_root_cause": true,
  "approach": "correct",
  "what_they_got_right": "...",
  "what_they_missed": "...",
  "arjun_message": "...",
  "mental_model": "..."
}
```

---

## Assets Used

### External APIs

| Service | Purpose | Endpoint |
|---------|---------|----------|
| **OSV.dev** | CVE database for open-source vulnerabilities | `https://api.osv.dev/v1/querybatch`, `/v1/vulns/{id}` |
| **GitHub API** | Fetch commit diffs to extract buggy/fixed code | `https://api.github.com/repos/{owner}/{repo}/commits/{sha}` |
| **Google Gemini** | Generate scenarios and evaluate fixes | `gemini-2.5-flash-lite` model |

### Python Packages (from `pyproject.toml`)

| Package | Purpose |
|---------|---------|
| `fastmcp` | MCP (Model Context Protocol) server framework for IDE integration |
| `google-genai` | Google's Gemini AI SDK |
| `httpx` | Async HTTP client for API calls |
| `python-dotenv` | Load environment variables from `.env` |

### Environment Variables

| Variable | Source | Purpose |
|----------|--------|---------|
| `GEMINI_API_KEY` | Google AI Studio | Authenticate Gemini API requests |
| `GITHUB_TOKEN` | GitHub Personal Access Token | Increase rate limits for GitHub API |

---

## Component Breakdown

### `cli.py` - Terminal Interface

**Purpose**: Standalone CLI for users who prefer terminal workflow

**Key Features**:
- Fetches scenario with `get_live_scenario("medium")`
- Maintains session state (start time, hint index, attempt count)
- Interactive menu: hints, submit fix, quit
- Multi-line input for pasting code (terminates with `END`)
- Displays formatted results with timing and attempt tracking

### `server.py` - MCP Server

**Purpose**: IDE integration via Model Context Protocol

**Tools Exposed**:
1. `get_ticket(difficulty)` - Fetch and display a new CVE scenario
2. `evaluate_fix(student_code)` - Evaluate a submitted fix
3. `get_hint(session_id)` - Get next progressive hint
4. `get_status(session_id)` - Show session stats (time, attempts, hints used)

**Session Management**:
- In-memory session storage (`sessions["current"]`)
- Tracks: scenario, start_time, hint_index, attempts

### `services/fetcher.py` - CVE Pipeline

**Purpose**: Fetch real CVEs, extract code, generate training scenarios

**Key Functions**:
- `fetch_python_cves(limit=50)` - Query OSV.dev for Python package CVEs
- `extract_code_from_commit(commit_url, github_token)` - Parse GitHub diffs
- `_parse_python_diff(diff)` - Extract Python code, skip tests/migrations
- `process_with_gemini(...)` - Generate standalone functions + hints + mental model
- `validate_scenario(scenario)` - Ensure scenario is complete and valid
- `get_live_scenario(difficulty)` - Orchestrates entire pipeline

### `services/evaluator.py` - Fix Evaluation

**Purpose**: Evaluate student fixes using AI (no hardcoded tests)

**Key Features**:
- Pure semantic evaluation via Gemini
- Compares student code against both buggy and fixed versions
- Returns structured feedback with score, verdict, and teaching points
- "Arjun" persona provides senior-dev-style code review feedback

---

## Design Decisions

### Why Real CVEs Instead of Artificial Examples?

- **Authenticity**: Developers learn from actual mistakes that affected production systems
- **Credibility**: "This CVE broke Django/Flask/requests" is more impactful than a toy example
- **Transferable learning**: Real fixes teach patterns that apply to other codebases

### Why AI Evaluation Instead of Hardcoded Tests?

- **Semantic understanding**: A test can't tell if you "almost got it" - AI can give partial credit
- **Flexible solutions**: Multiple valid approaches can be recognized as correct
- **Teaching feedback**: AI explains *why* something is wrong, not just that it failed

### Why Two Interfaces (CLI + MCP)?

- **CLI**: Works anywhere, no IDE setup required
- **MCP**: Integrates into developer's existing workflow (Cursor, Claude Code, etc.)
- **Same backend**: Both use `fetcher.py` and `evaluator.py` - just different frontends

### Why Skip Tests and Migrations in Diff Parsing?

- **Tests don't teach bugs**: Test changes show what to assert, not how to fix the vulnerability
- **Migrations are noise**: Database schema changes are orthogonal to the security fix
- **Focus on core logic**: The actual vulnerability fix is in the main source files

---

## Security Considerations

1. **No code execution**: Student code is never executed - evaluation is purely AI-based
2. **Subprocess isolation**: If code execution were needed, it would run in a subprocess with 5-second timeout
3. **API key management**: Keys stored in `.env`, loaded via `python-dotenv`
4. **No user data persistence**: Sessions are in-memory only, lost on server restart

---

## Future Extensions

1. **Difficulty tiers**: `easy`/`medium`/`hard` based on CVE severity or code complexity
2. **Session persistence**: Save sessions to disk for resuming later
3. **Leaderboards**: Track time-to-fix, hints-used, accuracy across users
4. **Multi-language support**: Expand beyond Python to JavaScript, Go, Rust CVEs
5. **Team mode**: Collaborative debugging sessions
6. **PR simulation**: Full code review workflow with comments and iterations

---

## File Structure

```
groundwork/
├── cli.py                 # Terminal-based interface
├── server.py              # MCP server for IDE integration
├── pyproject.toml         # Project dependencies
├── .env                   # Environment variables (API keys)
├── .gitignore             # Git ignore rules
├── README.md              # Project readme
└── services/
    ├── fetcher.py         # CVE fetching and scenario generation
    └── evaluator.py       # Student fix evaluation
```

---

## Summary

**GroundWork** is a security training tool that:

1. **Fetches real CVEs** from OSV.dev for popular Python packages
2. **Extracts actual buggy and fixed code** from GitHub commit diffs
3. **Generates training scenarios** using Gemini AI (ticket, hints, mental models)
4. **Evaluates student fixes** semantically via AI comparison against real fixes
5. **Provides two interfaces**: CLI for terminal users, MCP server for IDE integration

The pipeline ensures developers learn from **real-world vulnerabilities** with **authentic fixes** and receive **personalized, semantic feedback** - all within their existing development workflow.
