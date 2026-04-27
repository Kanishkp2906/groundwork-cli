from google import genai
import json, os

client = genai.Client(api_key=os.getenv("GEMINI_API_KEY"))


async def evaluate_student_fix(student_code: str, scenario: dict) -> dict:
    """
    Pure Gemini semantic evaluation.
    No hardcoded tests. No string matching.
    Gemini reads all three versions and judges like a senior dev.
    """

    prompt = f"""
You are Arjun, a senior software engineer doing a live code review.

A junior developer was given a real production bug to fix.
Here is everything you know:

━━━ THE VULNERABILITY ━━━
CVE: {scenario["cve_id"]}
Package: {scenario["real_package"]}
What it does: {scenario["ticket"]["description"]}

━━━ THE ORIGINAL BUGGY CODE ━━━
{scenario["buggy_function"]}

━━━ THE REAL FIX (from the actual GitHub commit) ━━━
{scenario["fixed_function"]}

━━━ WHAT THE STUDENT SUBMITTED ━━━
{student_code}

━━━ YOUR JOB ━━━
Compare the student's code semantically against the real fix.
You are NOT doing character matching. You are asking:
- Did they understand what the vulnerability actually was?
- Did they fix the root cause or just patch the symptom?
- Is their approach in the right ballpark even if syntax differs?
- Does their fix hold up against the same attack the CVE describes?

Return ONLY valid JSON — no markdown, no explanation outside the JSON:

{{
  "score": <integer 0-100>,
  "verdict": "<PASS|PARTIAL|FAIL>",
  "understood_vulnerability": <true|false>,
  "fixed_root_cause": <true|false>,
  "approach": "<correct|partial|wrong>",
  "what_they_got_right": "<one sentence, honest>",
  "what_they_missed": "<one sentence, specific — null if PASS>",
  "arjun_message": "<Arjun's full debrief — 3-4 sentences, direct, teaching, real senior dev energy. Reference the actual CVE. Tell them what a real PR review would have said.>",
  "mental_model": "<one paragraph — the transferable lesson that prevents this entire class of bug forever>"
}}

Scoring guide:
- 80-100 = PASS: understood the vulnerability, fixed root cause, approach matches or improves on real fix
- 40-79  = PARTIAL: right area, partially fixed, missed edge cases or used weaker approach
- 0-39   = FAIL: wrong approach, patched symptom, or didn't understand what was vulnerable
"""

    response = client.models.generate_content(
        model="gemini-2.5-flash-lite", contents=prompt
    )

    # Strip any markdown fences if Gemini adds them
    raw = response.text.strip()
    if raw.startswith("```"):
        raw = raw.split("```")[1]
        if raw.startswith("json"):
            raw = raw[4:]

    return json.loads(raw.strip())
