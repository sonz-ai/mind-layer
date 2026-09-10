#!/usr/bin/env python3
"""Exercise the real demo executable on a new temporary database."""
import argparse
import json
from pathlib import Path
import subprocess
import tempfile

root = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--record", action="store_true", help="save authored scenario output for diagrams")
args = parser.parse_args()
with tempfile.TemporaryDirectory(prefix="mind-layer-character-") as directory:
    run = subprocess.run(["go", "run", "./cmd/character-demo", "-scenario", "-data", str(Path(directory) / "demo.db")], cwd=root, check=True, capture_output=True, text=True)
result = json.loads(run.stdout)
assert result["revision"] == 5
assert result["traits"] == {"trust": 44, "confidence": 48, "curiosity": 65}
receipt = {"fixture": "Authored fictional five-interaction scenario; deterministic policy, no model calls.", "command": "python3 scripts/run_character_scenario.py --record", "character": result}
if args.record:
    (root / "benchmarks/results/character-scenario.json").write_text(json.dumps(receipt, indent=2) + "\n")
print(json.dumps(receipt, indent=2))
