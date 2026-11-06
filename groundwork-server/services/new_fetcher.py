"""
New fetcher that generates training scenarios from real CVE metadata using AI.
Does NOT require GitHub commit URLs or diffs — works with any CVE that has
a description, CWE, and affected package.

Optimizations:
- Parallel OSV detail fetching via asyncio.gather
- Redis caching of generated scenarios (24h TTL)
"""

import asyncio
import httpx
import random
import json
import os
import redis
from groq import Groq
from dotenv import load_dotenv
from language_conf import LANGUAGE_CONF

load_dotenv()
groq_client = Groq(api_key=os.getenv("GROQ_API_KEY"))

# Redis for caching scenarios
redis_url = os.getenv("UPSTASH_REDIS_URL")
try:
    redis_manager = redis.Redis.from_url(redis_url, decode_responses=True)
    redis_manager.ping()
    REDIS_AVAILABLE = True
except Exception:
    REDIS_AVAILABLE = False
    redis_manager = None

CACHE_TTL = 86400  # 24 hours
CONCURRENCY_LIMIT = 15  # Max concurrent OSV detail requests
OSV_TIMEOUT = 30.0  # Increased timeout for slower packages (Java/JS)
MAX_RETRIES = 3


async def fetch_cve_metadata(language: str = "python", limit: int = 100) -> list:
    """
    Queries OSV.dev for CVEs across packages for the given language.
    Returns CVEs with metadata (description, CWE, affected packages) but
    does NOT require GitHub commit URLs.
    """
    config = LANGUAGE_CONF.get(language, LANGUAGE_CONF["python"])

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

    all_vulns = []
    for result in results:
        all_vulns.extend(result.get("vulns", []))

    random.shuffle(all_vulns)

    # Fetch full details for each vuln IN PARALLEL
    async def fetch_one(vuln: dict) -> dict | None:
        vuln_id = vuln.get("id")
        if not vuln_id:
            return None

        # Retry with exponential backoff
        for attempt in range(MAX_RETRIES):
            try:
                async with httpx.AsyncClient(timeout=OSV_TIMEOUT) as client:
                    detail_resp = await client.get(f"https://api.osv.dev/v1/vulns/{vuln_id}")
                    if detail_resp.status_code != 200:
                        if attempt < MAX_RETRIES - 1:
                            await asyncio.sleep(2 ** attempt)  # Exponential backoff
                            continue
                        return None

                    full_vuln = detail_resp.json()

                    affected = full_vuln.get("affected", [])
                    pkg_name = ""
                    if affected:
                        pkg_name = affected[0].get("package", {}).get("name", "unknown")

                    severity = "HIGH"
                    db_specific = full_vuln.get("database_specific", {})
                    if db_specific.get("severity"):
                        severity = db_specific["severity"]

                    cwe = ""
                    for ref in full_vuln.get("references", []):
                        if "cwe" in ref.get("url", "").lower():
                            cwe = ref["url"].split("/")[-1]
                            break

                    return {
                        "id": vuln_id,
                        "summary": full_vuln.get("summary", "Security vulnerability"),
                        "details": full_vuln.get("details", ""),
                        "severity": severity,
                        "package": pkg_name,
                        "cwe": cwe,
                        "references": full_vuln.get("references", []),
                    }

            except (httpx.TimeoutException, httpx.ConnectError) as e:
                if attempt < MAX_RETRIES - 1:
                    await asyncio.sleep(2 ** attempt)  # Exponential backoff
                    continue
                print(f"[DEBUG] {vuln_id}: timeout/connect error after {MAX_RETRIES} attempts")
                return None
            except Exception as e:
                print(f"[DEBUG] {vuln_id}: unexpected error — {e}")
                return None

        return None

    # Run with bounded concurrency
    sem = asyncio.Semaphore(CONCURRENCY_LIMIT)

    async def fetch_with_sem(v):
        async with sem:
            return await fetch_one(v)

    tasks = [fetch_with_sem(v) for v in all_vulns[:limit]]
    results = await asyncio.gather(*tasks, return_exceptions=True)

    enriched = [r for r in results if isinstance(r, dict) and r]
    return enriched[:50]  # Cap at 50 usable CVEs


async def generate_scenario_with_ai(
    cve_data: dict,
    language: str = "python",
) -> dict:
    """
    Uses AI to generate a complete training scenario from CVE metadata.
    The AI invents realistic buggy/fixed code that demonstrates the vulnerability
    pattern described by the CVE.
    """
    config = LANGUAGE_CONF.get(language, LANGUAGE_CONF["python"])
    ext = config["extensions"][0] if config["extensions"] else ".py"

    prompt = f"""
You are a security training expert creating a realistic coding exercise for developers learning to fix vulnerabilities.

REAL CVE DATA:
- CVE ID: {cve_data["id"]}
- Summary: {cve_data["summary"]}
- Details: {cve_data.get("details", "Not available")}
- Severity: {cve_data["severity"]}
- Package: {cve_data["package"]}
- CWE: {cve_data.get("cwe", "Not specified")}
- Language: {language}

YOUR TASK:
Create a realistic but INVENTED code scenario that demonstrates this vulnerability class.
The code should:
1. Be plausible for the package ({cve_data["package"]}) and language ({language})
2. Demonstrate the vulnerability pattern described by the CVE/CWE
3. Have a clear, minimal fix
4. Be standalone and runnable

RETURN EXACTLY THREE SECTIONS separated by "===SPLIT===":

===SPLIT===
{{
  "ticket": {{
    "company": "A realistic Indian startup name (e.g., PayFlow, ShopNow, EduTrack)",
    "ticket_id": "A realistic ticket ID like SEC-104 or BUG-7732",
    "severity": "{cve_data["severity"]}",
    "title": "Short non-technical title a manager would write. Max 10 words. No CVE jargon.",
    "description": "2-3 sentences. Sound urgent. Describe the user-facing symptom. No CVE IDs or technical terms."
  }},
  "hints": [
    "Hint 1: Vague — points to the right area of the code without naming the fix",
    "Hint 2: Medium — names the concept involved without giving the solution",
    "Hint 3: Specific — almost gives it away, names the correct function or pattern to use"
  ],
  "mental_model": "One paragraph. The transferable lesson. What thinking pattern prevents this entire class of bug forever. Write for a junior developer.",
  "correct_pattern": "A SINGLE short string (max 3 words) that exists ONLY in the fixed code and NOT in the buggy code. If the fix is purely structural (e.g., changing an operator), pick a short unique snippet from the newly added lines."
}}
===SPLIT===
Make the VULNERABLE code a complete standalone {language} snippet ({ext}).
Add imports at the top and a simple test call at the bottom so it runs standalone.
Do NOT include the fix — this is the BUGGY version that a developer would need to fix.
===SPLIT===
Make the FIXED code — the same snippet with the vulnerability fixed.
Complete and standalone {language} code. Only the minimal fix applied.
"""

    response = groq_client.chat.completions.create(
        model="openai/gpt-oss-20b",
        messages=[{"role": "user", "content": prompt}],
        temperature=0.4,
        max_tokens=6000,
    )

    raw = response.choices[0].message.content.strip()
    parts = [p.strip() for p in raw.split("===SPLIT===") if p.strip()]

    if len(parts) < 3:
        raise ValueError(f"LLM did not return required 3 sections. Got: {len(parts)}")

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


def validate_scenario(scenario: dict) -> bool:
    """Validates a scenario is complete and has the right structure."""
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

    ticket_fields = ["company", "ticket_id", "severity", "title", "description"]
    for field in ticket_fields:
        if not scenario["ticket"].get(field):
            return False

    if len(scenario.get("hints", [])) < 3:
        return False

    pattern = scenario.get("correct_pattern", "")
    if len(pattern) < 2:
        return False
    if pattern not in scenario["fixed_function"]:
        return False
    if pattern in scenario["buggy_function"]:
        return False

    buggy = scenario["buggy_function"].strip()
    fixed = scenario["fixed_function"].strip()
    if buggy == fixed:
        return False

    if len(buggy) < 50 or len(fixed) < 50:
        return False

    return True


def _cache_key(language: str, cve_id: str) -> str:
    return f"scenario:{language}:{cve_id}"


async def get_live_scenario(language: str = "python") -> dict:
    """
    Main entry point — replaces the old get_live_scenario.
    Fetches real CVE metadata, then uses AI to generate the scenario.
    Uses Redis cache (24h TTL) to avoid repeated LLM calls.
    """
    # Try cache first
    if REDIS_AVAILABLE:
        # Get list of cached scenario keys for this language
        # We'll try a few random cached scenarios first
        cache_prefix = f"scenario:{language}:"
        try:
            keys = redis_manager.keys(f"{cache_prefix}*")
            if keys:
                random.shuffle(keys)
                for key in keys[:5]:  # Try up to 5 cached scenarios
                    cached = redis_manager.get(key)
                    if cached:
                        print(f"[DEBUG] Cache hit: {key}")
                        scenario = json.loads(cached)
                        # Validate cached scenario still looks good
                        if validate_scenario(scenario):
                            return scenario
        except Exception as e:
            print(f"[DEBUG] Cache read error: {e}")

    # No valid cache — fetch fresh CVEs
    cves = await fetch_cve_metadata(language=language, limit=100)
    random.shuffle(cves)

    last_error = None
    for cve in cves:
        cve_id = cve.get("id", "unknown")
        try:
            print(f"[DEBUG] Trying CVE: {cve_id} ({cve['package']})")

            scenario = await generate_scenario_with_ai(cve, language=language)

            if not validate_scenario(scenario):
                print(f"[DEBUG] {cve_id}: validation failed")
                continue

            print(f"[DEBUG] {cve_id}: generated valid scenario ✅")

            scenario["cve_id"] = cve_id
            scenario["real_package"] = cve["package"]
            scenario["language"] = language
            scenario["cwe"] = cve.get("cwe", "")

            # Cache the valid scenario
            if REDIS_AVAILABLE:
                try:
                    key = _cache_key(language, cve_id)
                    redis_manager.setex(key, CACHE_TTL, json.dumps(scenario))
                    print(f"[DEBUG] Cached: {key}")
                except Exception as e:
                    print(f"[DEBUG] Cache write error: {e}")

            return scenario

        except Exception as e:
            last_error = e
            print(f"[DEBUG] {cve_id}: error — {e}")
            continue

    raise RuntimeError(
        f"No valid scenario generated for {language}. Last error: {last_error}"
    )


# Compatibility function for the evaluator
async def evaluate_student_fix(student_code: str, scenario: dict) -> dict:
    """
    Evaluates student's fix against the generated scenario.
    Uses the correct_pattern from the scenario to verify the fix.
    """
    from services.evaluator import evaluate_student_fix as original_evaluate

    return await original_evaluate(student_code, scenario)