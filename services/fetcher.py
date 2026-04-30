import httpx
import asyncio
import json
import re
import os
import sys
import subprocess
from google import genai
from dotenv import load_dotenv

load_dotenv()
client = genai.Client(api_key=os.getenv("GEMINI_API_KEY"))

# Packages with lots of real, well-documented CVEs with GitHub fix commits
PYTHON_PACKAGES = [
    "fastapi",
    "django",
    "flask",
    "requests",
    "pillow",
    "cryptography",
    "sqlalchemy",
    "jinja2",
    "werkzeug",
    "aiohttp",
    "paramiko",
    "pyyaml",
    "lxml",
    "urllib3",
    "twisted",
    "httpx",
]


async def fetch_python_cves(limit: int = 50) -> list:
    """
    Queries OSV.dev for CVEs across multiple well-known Python packages
    using the querybatch endpoint. Returns only CVEs that have a real
    GitHub fix commit URL we can extract code from.
    """
    # Build a batch query — one entry per package
    queries = [
        {"package": {"name": pkg, "ecosystem": "PyPI"}} for pkg in PYTHON_PACKAGES
    ]

    async with httpx.AsyncClient(timeout=20) as client:
        response = await client.post(
            "https://api.osv.dev/v1/querybatch", json={"queries": queries}
        )
        response.raise_for_status()
        results = response.json().get("results", [])

    # Flatten all vulns from all packages into one list
    all_vulns = []
    for result in results:
        all_vulns.extend(result.get("vulns", []))

    # For each vuln we only get the ID from querybatch
    # We need to fetch the full details to get references + commit URLs
    usable = []
    async with httpx.AsyncClient(timeout=20) as client:
        for vuln in all_vulns[:limit]:
            vuln_id = vuln.get("id")
            if not vuln_id:
                continue

            # Fetch full vulnerability details by ID
            detail_resp = await client.get(f"https://api.osv.dev/v1/vulns/{vuln_id}")
            if detail_resp.status_code != 200:
                continue

            full_vuln = detail_resp.json()

            # Look for GitHub fix commit URLs in references
            refs = full_vuln.get("references", [])
            fix_commits = [
                r["url"]
                for r in refs
                if "github.com" in r.get("url", "") and "/commit/" in r.get("url", "")
            ]

            if fix_commits:
                full_vuln["fix_commit_url"] = fix_commits[0]
                usable.append(full_vuln)

            if len(usable) >= 20:
                break  # enough to work with

    return usable


async def extract_code_from_commit(commit_url: str, github_token: str) -> dict:
    """
    Takes a GitHub commit URL like:
    https://github.com/owner/repo/commit/abc123sha

    Fetches the raw diff of that commit from the GitHub API,
    then parses it to extract the buggy lines (removed, prefixed -)
    and the fixed lines (added, prefixed +) for Python files only.
    """

    # Step 1: Convert the browser URL to the GitHub API URL format
    # Browser:  https://github.com/pallets/flask/commit/abc123
    # API:      https://api.github.com/repos/pallets/flask/commits/abc123
    try:
        without_domain = commit_url.replace("https://github.com/", "")
        parts = without_domain.split("/commit/")
        repo_path = parts[0]  # e.g. "pallets/flask"
        sha = parts[1].split("/")[0]  # grab just the sha, ignore anything after
    except (IndexError, ValueError):
        return {}  # malformed URL — skip this CVE

    api_url = f"https://api.github.com/repos/{repo_path}/commits/{sha}"

    async with httpx.AsyncClient(timeout=20) as client:
        response = await client.get(
            api_url,
            headers={
                "Authorization": f"Bearer {github_token}",
                # This header tells GitHub to return the raw unified diff
                # instead of the default JSON commit object
                "Accept": "application/vnd.github.v3.diff",
            },
        )

        if response.status_code != 200:
            return {}  # commit not found or rate limited — skip

    diff_text = response.text
    return _parse_python_diff(diff_text)


def _parse_python_diff(diff: str) -> dict:
    """
    Parses unified diff, extracts Python code from source files only.
    Skips test files, migration files, and non-Python files.
    """
    # Files we don't want — tests and migrations don't teach real bugs
    SKIP_PATTERNS = [
        "test",
        "migration",
        "tests/",
        "migrations/",
        "setup.py",
        "conf.py",
        "settings",
    ]

    sections = []
    current_file = None
    current_removed = []
    current_added = []

    for line in diff.split("\n"):
        # New file section starts
        if line.startswith("diff --git"):
            # Save previous section if it had content
            if current_file and current_removed and current_added:
                sections.append(
                    {
                        "filename": current_file,
                        "removed": current_removed,
                        "added": current_added,
                    }
                )
            current_file = None
            current_removed = []
            current_added = []

            # Extract filename from "diff --git a/path/file.py b/path/file.py"
            try:
                filename = line.split(" b/")[-1].strip()
            except IndexError:
                continue

            # Only keep .py files that aren't tests or migrations
            if not filename.endswith(".py"):
                continue
            if any(skip in filename.lower() for skip in SKIP_PATTERNS):
                continue

            current_file = filename
            continue

        if not current_file:
            continue

        # Skip the --- and +++ header lines exactly
        if line.startswith("--- ") or line.startswith("+++ "):
            continue

        # Skip chunk position markers
        if line.startswith("@@"):
            continue

        # Removed lines = buggy code
        if line.startswith("-"):
            stripped = line[1:]  # remove the leading -
            if stripped.strip():  # skip blank removed lines
                current_removed.append(stripped)

        # Added lines = fixed code
        elif line.startswith("+"):
            stripped = line[1:]  # remove the leading +
            if stripped.strip():  # skip blank added lines
                current_added.append(stripped)

    # Don't forget the last section
    if current_file and current_removed and current_added:
        sections.append(
            {
                "filename": current_file,
                "removed": current_removed,
                "added": current_added,
            }
        )

    if not sections:
        return {}

    # Pick the section with the most changed lines — most likely the core fix
    best = max(sections, key=lambda s: len(s["removed"]) + len(s["added"]))

    buggy_code = "\n".join(best["removed"]).strip()
    fixed_code = "\n".join(best["added"]).strip()

    # Minimum size check — ignore trivial one-line changes
    if len(buggy_code) < 40 or len(fixed_code) < 40:
        return {}

    return {
        "buggy_code": buggy_code,
        "fixed_code": fixed_code,
        "filename": best["filename"],
    }


async def process_with_gemini(
    buggy_code: str,
    fixed_code: str,
    cve_description: str,
    severity: str,
    package_name: str,
) -> dict:
    """
    Gemini receives the real CVE code and description.
    Its only jobs:
      1. Make the buggy/fixed code standalone and runnable
      2. Write a realistic Jira ticket from the CVE description
      3. Generate 3 progressive hints based on the actual fix
      4. Extract the mental model the fix teaches
    It is NOT inventing bugs — it's formatting real data.
    """

    prompt = f"""
You are processing a real security vulnerability for a developer training platform.
You will receive real code from a real GitHub commit. Your job is formatting only.

REAL CVE DESCRIPTION:
{cve_description}

SEVERITY: {severity}
PACKAGE: {package_name}

VULNERABLE CODE (removed lines from the real GitHub commit diff):
{buggy_code}

FIXED CODE (added lines from the real GitHub commit diff):
{fixed_code}

YOUR TASKS — return ONLY valid JSON, no markdown fences, no explanation:

{{
  "buggy_function": "Take the vulnerable code above and make it a complete standalone Python function. Add realistic imports at the top and a simple test call at the bottom so it runs standalone. Do NOT change the core vulnerable logic — only wrap it so it executes.",

  "fixed_function": "Same function with the real fix from the diff applied. Must also run standalone.",

  "ticket": {{
    "company": "A realistic Indian startup name relevant to what this code does. E.g. PayFlow, ShopNow, EduTrack.",
    "ticket_id": "A realistic ticket ID like CRIT-2891 or SEC-104 or BUG-7732",
    "severity": "{severity}",
    "title": "Short non-technical title a manager would write. Max 10 words. No CVE jargon.",
    "description": "2-3 sentences. Sound urgent. Describe the user-facing symptom. No CVE IDs or technical terms."
  }},

  "hints": [
    "Hint 1: Vague — points to the right area of the code without naming the fix",
    "Hint 2: Medium — names the concept involved without giving the solution",
    "Hint 3: Specific — almost gives it away, names the correct function or pattern to use"
  ],

  "mental_model": "One paragraph. The transferable lesson. What thinking pattern prevents this entire class of bug forever. Write for a junior developer.",

  "correct_pattern": "The specific function name or string that must appear in a correct fix. E.g. bcrypt.checkpw or parameterized or Decimal. Single short string only."
}}
"""
    response = client.models.generate_content(
        model="gemini-2.5-flash-lite", contents=prompt
    )
    raw = response.text.strip()

    # Strip markdown fences if Gemini wraps the JSON anyway
    if raw.startswith("```"):
        raw = raw.split("```")[1]
        if raw.startswith("json"):
            raw = raw[4:]

    return json.loads(raw.strip())


def validate_scenario(scenario: dict) -> bool:
    """
    Validates a scenario is complete and has the right structure.
    Does NOT execute code — Gemini is trusted to produce valid Python.
    """

    # Check 1: All required fields present and non-empty
    required = [
        "buggy_function",
        "fixed_function",
        "ticket",
        "hints",
        "mental_model",
        "correct_pattern",
    ]
    for field in required:
        if not scenario.get(field):
            return False

    # Check 2: Ticket has all sub-fields
    ticket_fields = ["company", "ticket_id", "severity", "title", "description"]
    for field in ticket_fields:
        if not scenario["ticket"].get(field):
            return False

    # Check 3: Need all 3 hints
    if len(scenario.get("hints", [])) < 3:
        return False

    # Check 4: correct_pattern must exist in fixed code but NOT in buggy code
    pattern = scenario.get("correct_pattern", "")
    if len(pattern) < 2:
        return False
    if pattern not in scenario["fixed_function"]:
        return False
    if pattern in scenario["buggy_function"]:
        return False

    # Check 5: The two functions must be meaningfully different
    buggy = scenario["buggy_function"].strip()
    fixed = scenario["fixed_function"].strip()
    if buggy == fixed:
        return False

    # Check 6: Both functions must have reasonable length
    if len(buggy) < 50 or len(fixed) < 50:
        return False

    return True


def _run_code_safely(code: str) -> dict:
    """
    Executes Python code in a subprocess with a hard timeout.
    Returns stdout, stderr, returncode, and whether it timed out.
    """
    try:
        result = subprocess.run(
            [sys.executable, "-c", code],
            capture_output=True,
            text=True,
            timeout=5,  # hard kill after 5 seconds
        )
        return {
            "stdout": result.stdout,
            "stderr": result.stderr,
            "returncode": result.returncode,
            "timed_out": False,
        }
    except subprocess.TimeoutExpired:
        return {"stdout": "", "stderr": "Timed out", "returncode": 1, "timed_out": True}


async def get_live_scenario(difficulty: str = "medium") -> dict:
    github_token = os.getenv("GITHUB_TOKEN")

    cves = await fetch_python_cves(limit=50)
    print(f"[DEBUG] Got {len(cves)} usable CVEs")

    # ── Step 1: Find a CVE with good code WITHOUT calling Gemini ──
    # Loop through CVEs cheaply (just GitHub API calls)
    # until we find one with substantial code
    good_code = None
    good_cve = None

    for cve in cves:
        cve_id = cve.get("id", "unknown")
        try:
            code = await extract_code_from_commit(cve["fix_commit_url"], github_token)

            if not code or not code.get("buggy_code"):
                print(f"[DEBUG] {cve_id}: no usable code")
                continue

            buggy_len = len(code["buggy_code"])
            fixed_len = len(code["fixed_code"])

            # Only proceed if there's enough code to be interesting
            # Skips trivial one-liner changes
            if buggy_len < 100 or fixed_len < 100:
                print(f"[DEBUG] {cve_id}: code too short ({buggy_len}/{fixed_len})")
                continue

            # Found a good candidate — stop looping
            print(f"[DEBUG] {cve_id}: good code found ({buggy_len}/{fixed_len} chars)")
            good_code = code
            good_cve = cve
            break  # ← KEY: stop here, don't keep looping

        except Exception as e:
            print(f"[DEBUG] {cve_id}: GitHub error — {e}")
            continue

    if not good_code:
        raise RuntimeError("No CVEs with usable code found")

    # ── Step 2: Call Gemini ONCE on the best candidate ────────────
    cve_id = good_cve.get("id", "unknown")
    print(f"[DEBUG] Calling Gemini once for {cve_id}")

    scenario = await process_with_gemini(
        buggy_code=good_code["buggy_code"],
        fixed_code=good_code["fixed_code"],
        cve_description=good_cve.get("summary", "Security vulnerability"),
        severity="HIGH",
        package_name=good_cve.get("affected", [{}])[0]
        .get("package", {})
        .get("name", "unknown"),
    )

    # ── Step 3: Validate — if it fails, try the next CVE ──────────
    if not validate_scenario(scenario):
        print(f"[DEBUG] {cve_id}: failed validation, trying next CVE")

        # Try up to 3 more CVEs with one Gemini call each
        # with a gap between calls
        tried = 0
        for cve in cves:
            if cve.get("id") == good_cve.get("id"):
                continue  # skip the one we already tried
            if tried >= 3:
                break

            try:
                code = await extract_code_from_commit(
                    cve["fix_commit_url"], github_token
                )
                if not code or len(code.get("buggy_code", "")) < 100:
                    continue

                await asyncio.sleep(4)  # small gap between Gemini calls
                scenario = await process_with_gemini(
                    buggy_code=code["buggy_code"],
                    fixed_code=code["fixed_code"],
                    cve_description=cve.get("summary", ""),
                    severity="HIGH",
                    package_name=cve.get("affected", [{}])[0]
                    .get("package", {})
                    .get("name", "unknown"),
                )
                tried += 1

                if validate_scenario(scenario):
                    good_cve = cve
                    break

            except Exception:
                continue

    if not validate_scenario(scenario):
        raise RuntimeError("Could not produce a valid scenario")

    scenario["cve_id"] = good_cve.get("id", "Unknown")
    scenario["real_package"] = (
        good_cve.get("affected", [{}])[0].get("package", {}).get("name", "unknown")
    )
    return scenario
