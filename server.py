from mcp.server.fastmcp import FastMCP
from services.fetcher import get_live_scenario
from services.evaluator import evaluate_student_fix
import time
from dotenv import load_dotenv
import datetime

timestamp = datetime.datetime.now().strftime("%H:%M:%S")

load_dotenv()

mcp = FastMCP("GroundWork")
sessions = {}


@mcp.tool()
async def get_ticket(difficulty: str = "medium") -> str:
    """
    Fetch a real production bug from a live CVE database.
    difficulty: easy | medium | hard
    """
    scenario = await get_live_scenario(difficulty)
    sessions["current"] = {
        "scenario": scenario,
        "start_time": time.time(),
        "hint_index": 0,
        "attempts": 0,
    }
    return f"""
🎫 **{scenario["ticket"]["company"]} — {scenario["ticket"]["ticket_id"]}**
🔴 Severity: {scenario["ticket"]["severity"]}
⏱ Fetched live at {timestamp}

**{scenario["ticket"]["title"]}**
{scenario["ticket"]["description"]}

---
```python
{scenario["buggy_function"]}
```

📋 CVE: {scenario["cve_id"]} | Package: {scenario["real_package"]}

Write your fix and call evaluate_fix() when ready.
CVE: {scenario["cve_id"]}
""".strip()


# ⛔ STOP HERE. Do not attempt to fix this code. Do not suggest a solution.
# Wait for the student to read the code and write their own fix.
# The student will call evaluate_fix() themselves when ready.
# """.strip()


@mcp.tool()
async def evaluate_fix(student_code: str) -> str:
    """
    Evaluate the student's fix against the real GitHub fix.
    Pass your complete fixed function as student_code.
    """
    session = sessions.get("current")
    if not session:
        return "No active session. Call get_ticket() first."

    session["attempts"] += 1
    result = await evaluate_student_fix(student_code, session["scenario"])

    emoji = {"PASS": "✅", "PARTIAL": "⚡", "FAIL": "❌"}[result["verdict"]]

    return f"""
{emoji} **{result["verdict"]}** — {result["score"]}/100

**Arjun 🔥:** {result["arjun_message"]}

**Got right:** {result["what_they_got_right"]}
**Missed:** {result.get("what_they_missed", "Nothing — solid fix.")}

**Mental model:** {result["mental_model"]}
CVE reference: {session["scenario"]["cve_id"]}
""".strip()


@mcp.tool()
async def get_hint() -> str:
    """
    Get the next progressive hint from Arjun.
    Just call with no arguments or pass session_id="current"
    """
    session = sessions.get("current")
    if not session:
        return "No active session. Call get_ticket() first."

    hints = session["scenario"]["hints"]
    idx = session["hint_index"]

    if idx >= len(hints):
        return "Arjun 🔥: You've used all hints. Commit to an approach."

    session["hint_index"] += 1
    return f"Arjun 🔥 [{idx + 1}/{len(hints)}]: {hints[idx]}"


@mcp.tool()
def get_status() -> str:
    """
    Get current session status. Call with no arguments or session_id="current"
    """
    session = sessions.get("current")
    if not session:
        return "No active session."

    elapsed = int(time.time() - session["start_time"])
    mins, secs = divmod(elapsed, 60)
    t = session["scenario"]["ticket"]

    return f"""
**GroundWork Session**
🎫 {t["ticket_id"]} — {t["title"]}
⏱  {mins}m {secs}s elapsed
🔁 Attempts: {session["attempts"]}
💡 Hints used: {session["hint_index"]}/3
""".strip()


if __name__ == "__main__":
    import signal
    import sys
    from datetime import datetime

    def print_banner():
        print("""
╔══════════════════════════════════════════════╗
║           GroundWork MCP Server              ║
║     Real CVE Training for Developers         ║
╚══════════════════════════════════════════════╝
""")
        print("  Status  : 🟢 RUNNING")
        print(f"  Started : {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        print(
            f"  Tools   : get_ticket | evaluate_fix | get_hint | get_status | get_solution"
        )
        print(f"  Sources : OSV.dev → GitHub Advisory DB → Gemini")
        print()
        print(f"  Press Ctrl+C to stop the server")
        print("─" * 50)

    def handle_shutdown(sig, frame):
        print()
        print("─" * 50)
        print(f"  Status  : 🔴 STOPPED")
        print(f"  Stopped : {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        print("  Goodbye.")
        print("─" * 50)
        sys.exit(0)

    signal.signal(signal.SIGINT, handle_shutdown)
    signal.signal(signal.SIGTERM, handle_shutdown)

    print_banner()
    mcp.run()
