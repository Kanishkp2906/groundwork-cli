import asyncio
from services.fetcher import get_live_scenario
from services.evaluator import evaluate_student_fix
import time

async def main():
    print("🚀 Starting GroundWork session...")

    # Fetch ticket
    scenario = await get_live_scenario("medium")
    sessions = {
        "scenario": scenario,
        "start_time": time.time(),
        "hint_index": 0,
        "attempts": 0
    }

    ticket = scenario["ticket"]
    print(f"""
╔══════════════════════════════════════════╗
  🎫 {ticket['company']} — {ticket['ticket_id']}
  🔴 Severity: {ticket['severity']}

  {ticket['title']}
  {ticket['description']}
╚══════════════════════════════════════════╝

📋 CVE: {scenario['cve_id']} | Package: {scenario['real_package']}

🐛 BUGGY CODE — fix this in TRAE editor:
""")
    print(scenario["buggy_function"])
    print("\n" + "="*50)
    print("Write your fix in TRAE editor, then come back here.")
    print("="*50)

    while True:
        print("\nOptions: [h]int | [s]ubmit fix | [q]uit")
        choice = input("→ ").strip().lower()

        if choice == "h":
            hints = scenario["hints"]
            idx = sessions["hint_index"]
            if idx >= len(hints):
                print("Arjun 🔥: You've used all hints. Commit to an approach.")
            else:
                print(f"\nArjun 🔥 [{idx+1}/3]: {hints[idx]}")
                sessions["hint_index"] += 1

        elif choice == "s":
            print("\nPaste your complete fixed function below.")
            print("When done, type END on a new line and press Enter:\n")

            lines = []
            while True:
                line = input()
                if line.strip() == "END":
                    break
                lines.append(line)

            student_code = "\n".join(lines)
            sessions["attempts"] += 1

            print("\n⏳ Evaluating your fix...")
            result = await evaluate_student_fix(student_code, scenario)

            verdict_emoji = {"PASS": "✅", "PARTIAL": "⚡", "FAIL": "❌"}
            emoji = verdict_emoji[result["verdict"]]

            elapsed = int(time.time() - sessions["start_time"])
            mins, secs = divmod(elapsed, 60)

            print(f"""
{emoji} {result['verdict']} — {result['score']}/100
⏱  Time: {mins}m {secs}s | Attempts: {sessions['attempts']}

Arjun 🔥: {result['arjun_message']}

✅ Got right: {result['what_they_got_right']}
❌ Missed: {result.get('what_they_missed', 'Nothing — solid fix.')}

🧠 Mental model:
{result['mental_model']}

🔗 Real CVE: {scenario['cve_id']}
""")

            if result["verdict"] == "PASS":
                print("🎉 Session complete! Type 'q' to quit or start another.")

        elif choice == "q":
            print("Session ended. Good work.")
            break

if __name__ == "__main__":
    asyncio.run(main())