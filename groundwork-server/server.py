from fastapi import FastAPI, HTTPException, Request
from slowapi import Limiter, _rate_limit_exceeded_handler
from slowapi.util import get_remote_address
from slowapi.errors import RateLimitExceeded
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from services.fetcher import get_live_scenario
from services.evaluator import evaluate_student_fix
from dotenv import load_dotenv
import time
import json
import uuid
import redis
import uvicorn

try:
    redis_manager = redis.Redis(host="localhost", port=6379, decode_responses=True)

    if redis_manager.ping():
        print("Connected to redis successfully.")

except redis.ConnectionError as e:
    print(f"Error connecting to redis: {e}")

load_dotenv()

limiter = Limiter(key_func=get_remote_address)

app = FastAPI(
    title="Groundwork API",
    description="Real CVE training for developers",
    version="0.1.0",
)

app.state.limiter = limiter
app.add_exception_handler(RateLimitExceeded, _rate_limit_exceeded_handler)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


class EvaluateRequest(BaseModel):
    session_id: str
    student_code: str


@app.get("/")
@limiter.limit("10/minute")
async def root(request: Request):
    return {
        "name": "Groundwork API",
        "version": "0.1.0",
        "status": "running",
    }


@app.get("/ticket")
@limiter.limit("8/minute")
async def get_ticket(request: Request, language: str = "python"):

    valid_languages = ["python", "javascript", "java", "c++"]

    if language not in valid_languages:
        raise HTTPException(
            status_code=400, detail=f"language must be one of {valid_languages}"
        )

    scenario = await get_live_scenario(language)

    session = str(uuid.uuid4())
    session_detail = {
        "scenario": scenario,
        "start_time": time.time(),
        "hint_index": 0,
        "attempts": 0,
    }

    redis_manager.set(session, json.dumps(session_detail), ex=7200)

    return {
        "session_id": session,
        "ticket": scenario["ticket"],
        "buggy_code": scenario["buggy_function"],
        "cve_id": scenario["cve_id"],
        "package": scenario["real_package"],
        "language": language,
    }


@app.post("/evaluate")
@limiter.limit("8/minute")
async def evaluate(request: Request, req: EvaluateRequest):
    """
    Evaluate the student's fix against the real GitHub patch.
    """

    raw = redis_manager.get(req.session_id)

    if not raw:
        raise HTTPException(status_code=404, detail="Session not found.")

    session = json.loads(raw)
    session["attempts"] += 1

    redis_manager.set(req.session_id, json.dumps(session), ex=7200)

    result = await evaluate_student_fix(req.student_code, session["scenario"])

    result["attempts"] = session["attempts"]
    result["time_elapsed"] = int(time.time()) - session["start_time"]
    return result


@app.get("/hint")
@limiter.limit("8/minute")
async def get_hint(request: Request, session_id: str):
    """
    Get the next progressive hint for the current session.
    """

    raw = redis_manager.get(session_id)
    if not raw:
        raise HTTPException(status_code=404, detail="Session not found.")

    session = json.loads(raw)

    hints = session["scenario"]["hints"]
    idx = session["hint_index"]

    if idx >= len(hints):
        return {"hint": "No more hints. Commit to your approach.", "exhausted": True}

    session["hint_index"] += 1
    redis_manager.set(session_id, json.dumps(session), ex=7200)

    return {
        "hint": hints[idx],
        "index": idx + 1,
        "total": len(hints),
        "exhausted": False,
    }


@app.get("/solution")
@limiter.limit("8/minute")
async def get_solution(request: Request, session_id):
    """
    Reveal the real fix from the GitHub commit.
    Call after evaluate.
    """

    raw = redis_manager.get(session_id)
    if not raw:
        raise HTTPException(status_code=404, detail="Session not found.")

    session = json.loads(raw)

    scenario = session["scenario"]
    return {
        "fixed_code": scenario["fixed_function"],
        "cve_id": scenario["cve_id"],
        "package": scenario["real_package"],
    }


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
