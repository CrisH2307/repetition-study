#!/usr/bin/env python3
"""Blocks actions the project rules forbid. Exit code 2 stops the tool call."""
import json, os, re, subprocess, sys

def block(reason):
    print(f"Blocked by project rules: {reason}", file=sys.stderr)
    sys.exit(2)

def git(cwd, *args):
    r = subprocess.run(["git", "-C", cwd, *args], capture_output=True, text=True)
    return r.stdout.strip()

data = json.load(sys.stdin)
tool, inp, cwd = data.get("tool_name"), data.get("tool_input", {}), data.get("cwd") or os.getcwd()
branch, root = git(cwd, "branch", "--show-current"), git(cwd, "rev-parse", "--show-toplevel")

if tool == "Bash":
    for part in re.split(r"&&|\|\||;|\n", inp.get("command", "")):
        if re.search(r"\bgit\b.*\bpush\b.*\bmain\b", part):
            block("never push to main. Open a PR from qa, and Cris merges it.")
        if re.search(r"\bgit\b.*\b(commit|merge)\b", part) and branch in ("main", "qa"):
            block(f"never commit or merge directly on {branch}. Use a feature, fix or chore branch.")
        if re.search(r"\bgh\s+pr\s+merge\b", part):
            block("Cris merges pull requests.")
        if re.search(r"\bgit\s+add\b.*(?<![\w.])\.env(?![\w.])", part):
            block("never commit .env.")
    sys.exit(0)

if tool in ("Edit", "Write") and root:
    path = inp.get("file_path", "")
    rel = os.path.relpath(os.path.normpath(os.path.join(cwd, path)), root)
    try:
        units = json.load(open(os.path.join(root, ".claude", "boundaries.json")))
    except OSError:
        sys.exit(0)
    unit = next((u for u, v in units.items() if v["branch"] == branch), None)
    if unit is None or rel.startswith(".."):
        sys.exit(0)
    rule = units[unit]
    match = lambda p: p == "*" or rel == p or (p.endswith("/") and rel.startswith(p))
    if any(match(p) for p in rule.get("deny", [])) or not any(match(p) for p in rule["allow"]):
        block(f"{rel} is outside {unit}'s files: {rule['allow']}. Stop and ask Cris if the contract must change.")
sys.exit(0)
