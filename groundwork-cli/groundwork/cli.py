import asyncio
import httpx
import sys
import tty
import termios
import itertools
from slowapi.errors import RateLimitExceeded

SERVER_URL = "http://0.0.0.0:8000"
CLI_SECRET_KEY = "my_groundwork_secret_key"


def main():
    asyncio.run(run())


async def spinner(message: str, done_event: asyncio.Event):
    """Shows a live spinning indicator with status message."""
    frames = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"]
    stages = [
        "Querying OSV.dev for CVEs...",
        "Scanning GitHub commit history...",
        "Extracting vulnerable code...",
        "Processing with AI...",
        "Validating scenario...",
    ]
    stage_cycle = itertools.cycle(stages)
    current_stage = next(stage_cycle)
    stage_timer = 0

    for frame in itertools.cycle(frames):
        if done_event.is_set():
            # Clear the line on completion
            sys.stdout.write("\r\033[K")
            sys.stdout.flush()
            break

        sys.stdout.write(f"\r{frame} {current_stage}\033[K")
        sys.stdout.flush()

        await asyncio.sleep(0.1)
        stage_timer += 1

        # Rotate status message every 3 seconds
        if stage_timer >= 30:
            current_stage = next(stage_cycle)
            stage_timer = 0


def select_menu(options: list, title: str) -> int:
    selected = 0
    total = len(options)

    def render(first=False):
        if not first:
            # Move cursor up past all options + title
            sys.stdout.write(f"\033[{total + 1}A\033[J")
        sys.stdout.write(f"{title}\r\n")
        for i, option in enumerate(options):
            if i == selected:
                sys.stdout.write(f"  \033[42m\033[30m › {option} \033[0m\r\n")
            else:
                sys.stdout.write(f"    {option}\r\n")
        sys.stdout.flush()

    render(first=True)

    fd = sys.stdin.fileno()
    old_settings = termios.tcgetattr(fd)

    try:
        tty.setraw(fd)
        while True:
            ch = sys.stdin.read(1)

            if ch == "\x1b":
                seq = sys.stdin.read(2)
                if seq == "[A":  # up arrow
                    selected = (selected - 1) % total
                    render()
                elif seq == "[B":  # down arrow
                    selected = (selected + 1) % total
                    render()

            elif ch in ("\r", "\n"):  # enter
                break

            elif ch == "\x03":  # Ctrl+C
                raise KeyboardInterrupt

    finally:
        termios.tcsetattr(fd, termios.TCSADRAIN, old_settings)

    # Final clean render showing selection
    sys.stdout.write(f"\033[{total + 1}A\033[J")
    sys.stdout.write(f"{title}\n")
    for i, option in enumerate(options):
        if i == selected:
            sys.stdout.write(f"   ● {option}\n")
        else:
            sys.stdout.write(f"     {option}\n")
    sys.stdout.flush()
    print()

    return selected


async def run():
    print("""
╔══════════════════════════════════════════════╗
║              GroundWork                      ║
║     Real CVE Training for Developers         ║
╚══════════════════════════════════════════════╝
""")
    while True:
        languages = ["Python", "Javascript", "Java", "C++"]
        lang_idx = select_menu(languages, "Select a language:")
        language = languages[lang_idx].lower()

        print()
        done_event = asyncio.Event()

        # Run spinner and HTTP request concurrently
        spinner_task = asyncio.create_task(spinner("Fetching ticket...", done_event))

        headers = {"X-Cli-Secret": CLI_SECRET_KEY}

        async with httpx.AsyncClient(timeout=60) as client:
            # ── Fetch ticket ──────────────────────────────────────
            try:
                resp = await client.get(
                    f"{SERVER_URL}/ticket",
                    params={"language": language},
                    headers=headers,
                )
                resp.raise_for_status()
                data = resp.json()
            except httpx.ConnectError:
                done_event.set()
                await spinner_task
                print("❌ Cannot reach the GroundWork server.")
                print("   Check your internet connection and try again.")
                return
            except httpx.TimeoutException:
                done_event.set()
                await spinner_task
                print("❌ Server took too long to respond.")
                print("   The CVE pipeline can take up to 60s. Try again.")
                return
            except httpx.HTTPStatusError as e:
                done_event.set()
                await spinner_task
                print(f"❌ Server returned an error ({e.response.status_code}).")
                print("   Please try again in a moment.")
                return
            except RateLimitExceeded:
                done_event.set()
                await spinner_task
                print("❌ Rate limit exceed. Please try again after 2 minutes.")
            except Exception:
                done_event.set()
                await spinner_task
                print("❌ Something went wrong fetching your ticket.")
                print("   Please try again.")
                return
            finally:
                done_event.set()
                await spinner_task

            print("✅ Ticket ready!\n")

            session_id = data["session_id"]
            ticket = data["ticket"]
            buggy_code = data["buggy_code"]

            print(f"""
╔══════════════════════════════════════════════════════╗
  🎫  {ticket["company"]} — {ticket["ticket_id"]}
  🔴  Severity: {ticket["severity"]}

  {ticket["title"]}
╚══════════════════════════════════════════════════════╝

📄 DESCRIPTION: {ticket["description"]}

📦 CVE: {data["cve_id"]} | Package: {data["package"]}

🐛 BUGGY CODE:
""")
            print(buggy_code)
            print("\n" + "═" * 54)
            print("Read the code above. Write your fix in your editor.")
            print("═" * 54)

            # ── Main loop ─────────────────────────────────────────
            while True:
                print(
                    "\n[h] hint  [s] submit fix  [r] reveal solution  [n] new session  [q] quit"
                )
                choice = input("→ ").strip().lower()

                # ── Hint ──────────────────────────────────────────
                if choice == "h":
                    try:
                        resp = await client.get(
                            f"{SERVER_URL}/hint", params={"session_id": session_id}
                        )
                        resp.raise_for_status()
                        data = resp.json()
                        if data.get("exhausted"):
                            print("\nArjun 🔥: No more hints. You've got this.")
                        else:
                            print(
                                f"\nArjun 🔥 [{data['index']}/{data['total']}]: {data['hint']}"
                            )
                    except Exception:
                        print("❌ Could not fetch hint. Try again.")

                # ── Submit ────────────────────────────────────────
                elif choice == "s":
                    print("\nPaste your complete fixed function.")
                    print("Type END on a new line when done:\n")
                    lines = []
                    while True:
                        line = input()
                        if line.strip() == "END":
                            break
                        lines.append(line)

                    student_code = "\n".join(lines)
                    if not student_code.strip():
                        print("❌ No code entered. Try again.")
                        continue

                    print("\n⏳ Evaluating your fix...")

                    try:
                        resp = await client.post(
                            f"{SERVER_URL}/evaluate",
                            json={
                                "session_id": session_id,
                                "student_code": student_code,
                            },
                            timeout=60,
                        )
                        resp.raise_for_status()
                        result = resp.json()
                    except httpx.TimeoutException:
                        print("❌ Evaluation timed out. Try again.")
                        continue
                    except Exception:
                        print("❌ Could not evaluate your fix. Try again.")
                        continue

                    emoji = {"PASS": "✅", "PARTIAL": "⚡", "FAIL": "❌"}.get(
                        result.get("verdict", "FAIL"), "❌"
                    )
                    mins, secs = divmod(int(result.get("time_elapsed", 0)), 60)

                    print(f"""
    {emoji} {result.get("verdict", "FAIL")} — {result.get("score", 0)}/100
    ⏱  {mins}m {secs}s | Attempts: {result.get("attempts", 1)}

    Arjun 🔥: {result.get("arjun_message", "No feedback available.")}

    ✅ Got right : {result.get("what_they_got_right", "N/A")}
    ❌ Missed    : {result.get("what_they_missed", "Nothing — solid fix.")}

    🧠 Mental model:
    {result.get("mental_model", "N/A")}
    """)
                    if result.get("verdict") == "PASS":
                        print(
                            "🎉 Clean fix. Type 'r' to see the real patch or 'q' to quit."
                        )

                # ── Reveal solution ───────────────────────────────
                elif choice == "r":
                    try:
                        resp = await client.get(
                            f"{SERVER_URL}/solution", params={"session_id": session_id}
                        )
                        resp.raise_for_status()
                        sol = resp.json()
                        print(f"\n🔍 Real fix from {sol['cve_id']}:\n")
                        print(sol["fixed_code"])
                    except Exception:
                        print("❌ Could not fetch solution. Try again.")

                # ── Quit ──────────────────────────────────────────
                elif choice == "q":
                    print("\nSession ended. Keep fixing bugs. 👋")
                    return

                elif choice == "n":
                    print("\n")
                    break

                else:
                    print("Invalid option. Use h, s, r, n or q.")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\nExiting GroundWork. Keep fixing bugs! 👋")
        sys.exit(0)
