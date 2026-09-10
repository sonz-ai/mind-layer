#!/usr/bin/env python3
"""Local release scan: exact nonignored files plus generated docs; no matched values."""
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile


def inspect(path, text):
    hits = []
    rules = {
        "personal-home-path": re.compile(r"/(?:Users|home)/[A-Za-z0-9_.-]+/"),
        "private-key": re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"),
        "provider-token": re.compile(r"\b(?:AIza[A-Za-z0-9_-]{30,}|gh[pousr]_[A-Za-z0-9]{20,}|sk-[A-Za-z0-9_-]{25,})\b"),
    }
    for number, line in enumerate(text.splitlines(), 1):
        for rule, pattern in rules.items():
            if pattern.search(line):
                hits.append({"path": path, "line": number, "rule": rule})
        # Third-party public license contacts are required attribution, not private fixtures.
        if path not in {"THIRD_PARTY_NOTICES.md", "LICENSE"}:
            for email in re.findall(r"[\w.+-]+@([\w.-]+\.[A-Za-z]{2,})", line):
                if email not in {"example.com", "example.org", "example.net", "example.test"} and not email.endswith(".example.test"):
                    hits.append({"path": path, "line": number, "rule": "non-example-email"})
    return hits


def main():
    root = Path(__file__).resolve().parents[1]
    gitdir = Path(subprocess.check_output(["git", "rev-parse", "--absolute-git-dir"], cwd=root, text=True).strip())
    reportdir = gitdir / "release-audit"
    reportdir.mkdir(exist_ok=True, mode=0o700)
    raw = subprocess.check_output(["git", "ls-files", "--cached", "--others", "--exclude-standard", "-z"], cwd=root)
    names = sorted(set(raw.decode().split("\0")) - {""})
    if (root / "site").exists():
        names += [p.relative_to(root).as_posix() for p in (root / "site").rglob("*") if p.is_file()]
    hits = []
    forbidden = {".db", ".sqlite", ".pem", ".key", ".p12", ".log", ".zip", ".pdf", ".png", ".jpg", ".xlsx"}
    with tempfile.TemporaryDirectory(prefix="mind-layer-release-") as temp:
        snap = Path(temp) / "source"
        snap.mkdir()
        for name in names:
            path = root / name
            if path.is_symlink() or not path.is_file() or not path.resolve().is_relative_to(root):
                hits.append({"path": name, "rule": "unscannable-path"})
                continue
            if path.suffix in forbidden or (path.name.startswith(".env") and path.name != ".env.example"):
                hits.append({"path": name, "rule": "forbidden-release-artifact"})
                continue
            try:
                content = path.read_text(encoding="utf-8")
            except (UnicodeError, OSError):
                hits.append({"path": name, "rule": "unscannable-content"})
                continue
            hits += inspect(name, content)
            dest = snap / name
            dest.parent.mkdir(parents=True, exist_ok=True)
            dest.write_text(content)
        scanner = shutil.which("gitleaks")
        if not scanner:
            hits.append({"rule": "gitleaks-missing"})
        else:
            output = Path(temp) / "detector.json"
            run = subprocess.run([scanner, "dir", str(snap), "--redact=100", "--ignore-gitleaks-allow", "--report-format", "json", "--report-path", str(output), "--no-banner"], capture_output=True)
            if run.returncode not in (0, 1):
                hits.append({"rule": "gitleaks-error"})
            if output.exists():
                for hit in json.loads(output.read_text()):
                    path = Path(hit["File"])
                    if path.is_absolute():
                        path = path.relative_to(snap)
                    hits.append({"path": path.as_posix(), "line": hit["StartLine"], "rule": "gitleaks:" + hit["RuleID"]})
    report = {"status": "review_required" if hits else "no_automated_findings", "files_scanned": len(names), "findings": hits,
              "limitations": "No historical commits scanned. Heuristics do not detect every name, address, proprietary detail or encoded secret. License attribution contacts are permitted. A human still reviews the exact release diff."}
    output = reportdir / "report.json"
    output.write_text(json.dumps(report, indent=2) + "\n")
    os.chmod(output, 0o600)
    print(json.dumps({"status": report["status"], "files_scanned": len(names), "finding_count": len(hits)}))
    return 1 if hits else 0


if __name__ == "__main__":
    raise SystemExit(main())
