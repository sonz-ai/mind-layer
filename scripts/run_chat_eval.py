#!/usr/bin/env python3
"""Synthetic chat smoke eval through the real HTTP server; never a model-quality benchmark."""
import argparse
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import socket
import subprocess
import tempfile
import time
import urllib.request
import urllib.error
import os

ROOT = Path(__file__).resolve().parents[1]
MESSAGES = [
    "I am building a greenhouse called Fern House. I want to grow basil in it.",
    "Can you help me think of a small first step?",
    "Actually, I renamed the greenhouse Cedar Room. Please use the new name.",
    "What did I name my greenhouse, and what do I want to grow there?",
    "What is my favorite music?",
]

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--live", action="store_true", help="explicitly allow configured provider calls and charges")
    parser.add_argument("--record", action="store_true", help="record synthetic results for publication")
    args = parser.parse_args()
    turns = []
    with tempfile.TemporaryDirectory(prefix="mind-layer-chat-eval-") as temp:
        binary = Path(temp) / "character-demo"
        subprocess.run(["go", "build", "-o", str(binary), "./cmd/character-demo"], cwd=ROOT, check=True)
        with socket.socket() as s:
            s.bind(("127.0.0.1", 0))
            port = s.getsockname()[1]
        base = f"http://127.0.0.1:{port}"
        command = [str(binary), "-addr", f"127.0.0.1:{port}", "-data", str(Path(temp) / "chat.db")]
        if args.live:
            command.append("-live")

        def request(path, body=None):
            req = urllib.request.Request(base + path, data=None if body is None else json.dumps(body).encode(), headers={"Content-Type": "application/json"})
            try:
                with urllib.request.urlopen(req, timeout=75) as response:
                    return json.load(response)
            except urllib.error.HTTPError as error:
                raise RuntimeError(f"{path}: HTTP {error.code}: {error.read().decode()}") from None

        def start():
            process = subprocess.Popen(command, cwd=ROOT, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            try:
                for _ in range(100):
                    if process.poll() is not None:
                        raise RuntimeError("demo exited before readiness")
                    try:
                        request("/api/state")
                        return process
                    except OSError:
                        time.sleep(.1)
                raise RuntimeError("demo readiness timeout")
            except BaseException:
                stop(process)
                raise

        def stop(process):
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()

        process = start()
        try:
            for i, message in enumerate(MESSAGES):
                started = time.monotonic()
                turn = request("/api/chat", {"query": message, "revision": i})
                turns.append({"turn": turn, "milliseconds": round((time.monotonic() - started) * 1000, 2)})
            before = request("/api/chat")
        finally:
            stop(process)
        process = start()
        try:
            after = request("/api/chat")
            state = request("/api/state")
        finally:
            stop(process)
    recalled = [h["memory"]["text"] for h in turns[3]["turn"]["context"]["memories"]]
    checks = {
        "five_complete_turns": len(before) == 5,
        "restart_preserves_exact_receipts": before == after,
        "retrieved_original_fact": any("basil" in text for text in recalled),
        "retrieved_correction": any("Cedar Room" in text for text in recalled),
        "free_chat_does_not_infer_traits": state["character"]["revision"] == 0,
        "mode_labeled": all(t["turn"]["mode"] == ("model" if args.live else "offline") for t in turns),
    }
    result = {
        "fixture": "authored greenhouse conversation v1; synthetic data only",
        "dataset_sha256": hashlib.sha256(json.dumps(MESSAGES).encode()).hexdigest(),
        "measured_at": datetime.now(timezone.utc).isoformat(),
        "mode": "live" if args.live else "offline",
        "model": os.environ.get("MIND_LAYER_CHAT_MODEL", "") if args.live else None,
        "checks": checks,
        "passed": all(checks.values()),
        "limits": "Five authored messages, one local process, one restart. Structural checks measure persistence and context selection, not answer quality. Read live replies manually for correction handling and unknown-fact honesty. Latencies include local HTTP and, in live mode, provider time; they are not a load benchmark.",
        "turns": turns,
    }
    if args.record:
        name = "chat-live.json" if args.live else "chat-offline.json"
        (ROOT / "benchmarks/results" / name).write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps({k: v for k, v in result.items() if k != "turns"}, indent=2))
    if not result["passed"]:
        raise SystemExit(1)

if __name__ == "__main__":
    main()
