import httpx
import random
import json
import os
from google import genai
from groq import Groq
from dotenv import load_dotenv
import redis
from language_conf import LANGUAGE_CONF

load_dotenv()
client = genai.Client(api_key=os.getenv("GEMINI_API_KEY"))
groq_client = Groq(api_key=os.getenv("GROQ_API_KEY"))

try:
    redis_manager = redis.Redis(host="localhost", port=6379, decode_responses=True)

    if redis_manager.ping():
        print("Connected to redis successfully.")

except redis.ConnectionError as e:
    print(f"Error connecting to redis: {e}")


async def fetch_cves(language: str = "python", limit: int = 200) -> list:
    """
    Queries OSV.dev for CVEs across multiple well-known Python packages
    using the querybatch endpoint. Returns only CVEs that have a real
    GitHub fix commit URL we can extract code from.
    """

    config = LANGUAGE_CONF.get(language, LANGUAGE_CONF["python"])

    # Build a batch query — one entry per package
    queries = [
        {"package": {"name": pkg, "ecosystem": config["ecosystem"]}}
        for pkg in config["packages"]
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

    random.shuffle(all_vulns)

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
                if "github.com" in r.get("url", "")
                and "/commit/" in r.get("url", "")
                and "github.com/pypa/advisory-database" not in r.get("url", "")
            ]

            for affected in full_vuln.get("affected", []):
                for rng in affected.get("ranges", []):
                    if rng.get("type") == "GIT":
                        repo = rng.get("repo", "")

                        if "github.com" in repo:
                            repo_url = repo.rstrip("/")
                            if repo_url.endswith(".git"):
                                repo_url = repo_url[:-4]

                            for event in rng.get("events", []):
                                if "fixed" in event:
                                    commit_url = f"{repo_url}/commit/{event['fixed']}"
                                    fix_commits.append(commit_url)

            if fix_commits:
                full_vuln["fix_commit_url"] = fix_commits[0]
                usable.append(full_vuln)

            if len(usable) >= 50:
                break  # enough to work with

    return usable


async def extract_code_from_commit(
    commit_url: str, github_token: str, language: str = "python"
) -> dict:
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
    return parse_diff(diff_text, language=language)


def parse_diff(diff: str, language: str = "python") -> dict:
    """
    Parses unified diff, extracts Python code from source files only.
    Skips test files, migration files, and non-Python files.
    """

    config = LANGUAGE_CONF.get(language, LANGUAGE_CONF["python"])
    extensions = config["extensions"]
    skip_patterns = config["skip"]

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
            if not any(filename.endswith(ext) for ext in extensions):
                continue
            if any(skip in filename.lower() for skip in skip_patterns):
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
    language: str = "python",
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

    #    prompt = rf"""
    # You are processing a real security vulnerability for a developer training platform.
    # You will receive real code from a real GitHub commit. Your job is formatting only.
    # The code is written in {language.upper()}.
    #
    # REAL CVE DESCRIPTION:
    # {cve_description}
    #
    # SEVERITY: {severity}
    # PACKAGE: {package_name}
    #
    # VULNERABLE CODE (removed lines from the real GitHub commit diff):
    # {buggy_code}
    #
    # FIXED CODE (added lines from the real GitHub commit diff):
    # {fixed_code}
    #
    # YOUR TASKS — return ONLY valid JSON, no markdown fences, no explanation:
    # CRITICAL: You must properly double-escape all backslashes inside the code strings (e.g., use \\n instead of \n, and \\d instead of \d) so the JSON parses correctly.
    #
    # {{
    #  "buggy_function": "Make this a complete standalone {language} snippet. Add imports/requires at the top and a simple test call at the bottom so it runs standalone. Do NOT change the core vulnerable logic — only wrap it so it executes.",
    #
    #  "fixed_function": "Same snippet with the real fix applied. Complete and standalone {language} code.",
    #
    #  "ticket": {{
    #    "company": "A realistic Indian startup name relevant to what this code does. E.g. PayFlow, ShopNow, EduTrack.",
    #    "ticket_id": "A realistic ticket ID like CRIT-2891 or SEC-104 or BUG-7732",
    #    "severity": "{severity}",
    #    "title": "Short non-technical title a manager would write. Max 10 words. No CVE jargon.",
    #    "description": "2-3 sentences. Sound urgent. Describe the user-facing symptom. No CVE IDs or technical terms."
    #  }},
    #
    #  "hints": [
    #    "Hint 1: Vague — points to the right area of the code without naming the fix",
    #    "Hint 2: Medium — names the concept involved without giving the solution",
    #    "Hint 3: Specific — almost gives it away, names the correct function or pattern to use"
    #  ],
    #
    #  "mental_model": "One paragraph. The transferable lesson. What thinking pattern prevents this entire class of bug forever. Write for a junior developer.",
    #
    #  "correct_pattern": "A SINGLE short string (max 3 words) that exists ONLY in the fixed_function and NOT in the buggy_function. Do not include variables, object instances, or full method calls. Good examples: 'escapeJava', 'PreparedStatement'. IF the fix is purely structural (e.g., changing a math operator or moving pointers), pick a short, unique snippet from the newly added lines (e.g., 'prev->next = n', 'size + 1')."
    # }}
    # """
    prompt = f"""
You are processing a real security vulnerability for a developer training platform.
You will receive real code from a real GitHub commit. Your job is formatting only.
The code is written in {language.upper()}.

REAL CVE DESCRIPTION:
{cve_description}

SEVERITY: {severity}
PACKAGE: {package_name}

VULNERABLE CODE (removed lines from the real GitHub commit diff):
{buggy_code}

FIXED CODE (added lines from the real GitHub commit diff):
{fixed_code}

YOUR TASKS:
Return EXACTLY THREE SECTIONS separated by the exact string "===SPLIT===". 
Do not add any extra text before or after the splits.

===SPLIT===
{{
  "ticket": {{
    "company": "A realistic Indian startup name.",
    "ticket_id": "A realistic ticket ID like SEC-104",
    "severity": "{severity}",
    "title": "Short non-technical title. Max 10 words.",
    "description": "2-3 sentences describing the user-facing symptom."
  }},
  "hints": [
    "Hint 1: Vague",
    "Hint 2: Medium",
    "Hint 3: Specific"
  ],
  "mental_model": "One paragraph transferable lesson.",
  "correct_pattern": "A SINGLE short string (max 3 words) that exists ONLY in the fixed code and NOT in the buggy code. IF the fix is purely structural, pick a short, unique snippet from the newly added lines. IF the fix is ONLY removing code (nothing new added), output the exact string 'REMOVAL_ONLY'."
}}
===SPLIT===
Make the vulnerable code a complete standalone {language} snippet. Add imports/requires at the top and a simple test call at the bottom. Do NOT change the core vulnerable logic.
Write the code normally. No JSON.
===SPLIT===
Same standalone snippet with the real fix applied. Complete and standalone {language} code.
Write the code normally. No JSON.
"""
    # response = client.models.generate_content(
    #    model="gemini-2.5-flash-lite", contents=prompt
    # )
    # raw = response.text.strip()

    response = groq_client.chat.completions.create(
        model="llama-3.3-70b-versatile",
        messages=[{"role": "user", "content": prompt}],
        temperature=0.3,
        max_tokens=6000,
        # response_format={"type": "json_object"},
    )

    raw = response.choices[0].message.content.strip()

    parts = raw.split("===SPLIT===")

    parts = [p.strip() for p in parts if p.strip()]

    if len(parts) < 3:
        raise ValueError("LLM did not return the required 3 split sections.")

    json_text = parts[0]
    buggy_text = parts[1]
    fixed_text = parts[2]

    if json_text.startswith("```"):
        json_text = json_text.split("```")[1]
        if json_text.startswith("json"):
            json_text = json_text[4:]

    scenario = json.loads(json_text.strip(), strict=False)

    def clean_code_block(code_str):
        if code_str.startswith("```"):
            lines = code_str.split("\n")
            if len(lines) >= 2 and lines[-1].startswith("```"):
                return "\n".join(lines[1:-1]).strip()
        return code_str.strip()

    scenario["buggy_function"] = clean_code_block(buggy_text)
    scenario["fixed_function"] = clean_code_block(fixed_text)

    return scenario


#    # Strip markdown fences if Gemini wraps the JSON anyway
#    if raw.startswith("```"):
#        raw = raw.split("```")[1]
#        if raw.startswith("json"):
#            raw = raw[4:]
#
#    raw = raw.strip()
#
#    try:
#        return json.loads(raw, strict=False)
#    except json.JSONDecodeError:
#        clean_raw = raw.replce("\\", "\\\\").replace('\\\\"', '\\"')
#        return json.loads(clean_raw, strict=False)


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
    if pattern == "REMOVAL_ONLY":
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


async def get_live_scenario(language: str = "python") -> dict:
    # Try cache first
    # cache_key = f"scenario_cache:{language}"
    # cached_ids = redis_manager.lrange(cache_key, 0, -1)

    # if cached_ids:
    #    scenario_key = random.choice(cached_ids)
    #    cached = redis_manager.get(scenario_key)
    #    if cached:
    #        print(f"[DEBUG] Serving from cache: {scenario_key}")
    #        return json.loads(cached)

    github_token = os.getenv("GITHUB_TOKEN")
    print(f"[DEBUG] Language: {language}")

    cves = await fetch_cves(language=language, limit=200)
    print(f"[DEBUG] Got {len(cves)} usable CVEs")

    random.shuffle(cves)

    good_code = None
    good_cve = None

    # Step 1 — find a CVE matching the difficulty (no LLM calls yet)
    for cve in cves:
        cve_id = cve.get("id", "unknown")
        try:
            code = await extract_code_from_commit(
                cve["fix_commit_url"], github_token, language=language
            )
            if not code or not code.get("buggy_code"):
                print(f"[DEBUG] {cve_id}: no usable code")
                continue

            # Count non-empty, non-comment lines

            if not code or not code.get("buggy_code"):
                print(f"[DEBUG] {cve_id}: no usable code")
                continue

            buggy_len = len(code["buggy_code"])
            if buggy_len < 80:
                print(f"[DEBYG] {cve_id}: too short ({buggy_len} chars)")
                continue

            print(f"[DEBUG] {cve_id}: good code found ({buggy_len} chars) ✅")
            good_code = code
            good_cve = cve
            break

        except Exception as e:
            print(f"[DEBUG] {cve_id}: error — {e}")
            continue

    if not good_code:
        raise RuntimeError(f"No valud CVEs for {language} with usable code found")

    # Step 2 — call LLM once on the best candidate
    cve_id = good_cve.get("id", "unknown")
    print(f"[DEBUG] Calling LLM once for {cve_id}")

    scenario = await process_with_gemini(
        buggy_code=good_code["buggy_code"],
        fixed_code=good_code["fixed_code"],
        cve_description=good_cve.get("summary", "Security vulnerability"),
        severity="HIGH",
        package_name=good_cve.get("affected", [{}])[0]
        .get("package", {})
        .get("name", "unknown"),
        language=language,
    )

    #    print("\n[DEBUG] Raw LLM output:")
    #    print(json.dumps(scenario, indent=2))

    if not validate_scenario(scenario):
        raise RuntimeError("Could not produce a valid scenario")

    scenario["cve_id"] = good_cve.get("id", "Unknown")
    scenario["real_package"] = (
        good_cve.get("affected", [{}])[0].get("package", {}).get("name", "unknown")
    )
    scenario["language"] = language

    # scenario_key = f"scenario: {scenario['cve_id']}: {language}"
    # redis_manager.set(scenario_key, json.dumps(scenario), ex=86400 * 7)
    # redis_manager.lpush(cache_key, scenario_key)
    # redis_manager.ltrim(cache_key, 0, 49)

    return scenario
