---
title: GroundWork Server
emoji: 🛠️
colorFrom: blue
colorTo: green
sdk: docker
pinned: false
license: other
---

# GroundWork API Server

This is the FastAPI backend that powers the [GroundWork CLI](https://github.com/yourusername/groundwork-cli). 

It dynamically fetches real-world Common Vulnerabilities and Exposures (CVEs) from GitHub Advisories and OSS-Fuzz, securely formats them, and delivers them to the CLI for interactive developer training.

## Architecture
* **Framework:** FastAPI
* **Deployment:** Hugging Face Docker Space (Port 7860)
* **Security:** IP Rate Limiting (`slowapi`) and hardcoded CLI handshake headers.

## API Usage
This server is not intended to be browsed directly. To interact with this API, please install the GroundWork CLI:

```bash
pip install groundwork-cli
groundwork
