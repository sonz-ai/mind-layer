#!/usr/bin/env python3
"""Require passing tests for every documented supported capability."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
from datetime import datetime, timezone

root = Path(__file__).resolve().parents[1]
caps = json.loads((root / "docs/capabilities.json").read_text())
env = dict(os.environ)
env.pop("MIND_LAYER_LIVE_TEST", None)
run = subprocess.run(["go", "test", "-race", "-count=1", "-json", "./..."], cwd=root, env=env, capture_output=True, text=True)
events = []
for line in run.stdout.splitlines():
    try:
        events.append(json.loads(line))
    except json.JSONDecodeError:
        pass
passed = {e["Test"] for e in events if e.get("Action") == "pass" and e.get("Package") == "github.com/sonz-ai/mind-layer" and "Test" in e}
passed.update(e["Test"] for e in events if e.get("Action") == "pass" and e.get("Package") == "github.com/sonz-ai/mind-layer/examples/character" and "Test" in e)
missing = {test for cap in caps for test in cap["tests"]} - passed
if run.returncode or missing:
    print(run.stdout)
    print(run.stderr, file=sys.stderr)
    raise SystemExit("Verification failed; missing passing capability tests: " + ", ".join(sorted(missing)))
subprocess.run(["go", "vet", "./..."], cwd=root, check=True)
source_hash = hashlib.sha256()
for path in sorted(root.rglob("*.go")):
    source_hash.update(path.relative_to(root).as_posix().encode() + b"\0" + path.read_bytes())
report = {
    "status": "pass", "measured_at": datetime.now(timezone.utc).isoformat(),
    "command": "go test -race -count=1 -json ./...; go vet ./...",
    "go_source_sha256": source_hash.hexdigest(),
    "capabilities": [{"id": cap["id"], "passing_tests": cap["tests"]} for cap in caps],
    "test_pass_events": sum(e.get("Action") == "pass" and "Test" in e for e in events),
    "note": "Includes subtests. Live provider tests skipped in this offline run. Passing provider mocks establish contracts, not model quality.",
}
dest = root / "benchmarks/results/capabilities.json"
dest.parent.mkdir(parents=True, exist_ok=True)
dest.write_text(json.dumps(report, indent=2) + "\n")
print(f"PASS: {len(caps)} documented capabilities; {report['test_pass_events']} passing test events; race detector and go vet passed.")
