import httpx

def extract_code_from_commit(commit_url: str, github_token: str) -> dict:
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

    with httpx.Client(timeout=20) as client:
        response = client.get(
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
    print(diff_text)

if __name__ == "__main__":
    extract_code_from_commit()