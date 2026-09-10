#!/usr/bin/env python3
"""Collect dependency license files without copying local paths or module source."""
import json
from pathlib import Path
import subprocess

root = Path(__file__).resolve().parents[1]
raw = subprocess.check_output(["go", "list", "-m", "-json", "all"], cwd=root, text=True)
decoder = json.JSONDecoder()
modules = []
while raw.strip():
    module, end = decoder.raw_decode(raw.lstrip())
    modules.append(module)
    raw = raw.lstrip()[end:]
lines = ["# Third-party notices", "", "Public upstream license and copyright attribution is preserved below.",
         "Versions are pinned in go.mod/go.sum. This inventory includes transitive", "and test dependencies, not only the binary's runtime dependency graph.", ""]
missing = []
for module in modules:
    if module.get("Main"):
        continue
    directory = Path(module.get("Dir", ""))
    licenses = sorted(p for p in directory.iterdir() if p.is_file() and p.name.lower().startswith(("license", "licence", "copying", "notice"))) if module.get("Dir") else []
    if not licenses:
        missing.append(module["Path"])
        continue
    lines += [f"## {module['Path']} {module['Version']}", ""]
    for license in licenses:
        lines += [f"### {license.name}", "", "```text", license.read_text(errors="replace").strip(), "```", ""]
if missing:
    raise SystemExit("Missing upstream license files; review: " + ", ".join(missing))
(root / "THIRD_PARTY_NOTICES.md").write_text("\n".join(lines))
print(f"Collected license notices for {len(modules)-1} dependencies.")
