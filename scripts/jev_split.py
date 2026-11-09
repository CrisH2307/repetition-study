#!/usr/bin/env python3
"""Regulation 3: score each unit's effort and skill with Jev. Run on Cris's Mac.
Writes rpi/work-split.md. Use --dry-run to see what is sent without calling Jev."""
import json, os, sys, urllib.request

UNITS = {
    "U1": "Parse one math PDF, list concepts and claims, build a hand-labelled test set",
    "U2": "Go streaming teach engine with a code-owned knowledge state for an AI student",
    "U3": "LLM claim checker with code-verified citations and a model comparison tool",
    "U4": "Go scheduler wrapping an FSRS library, append-only review log, manual overrides",
    "U5": "Deterministic greedy weekly study planner and per-concept analytics in Go",
    "U6": "Daily per-subject digest with suppression and time zone handling",
    "U7": "React demo UI with browser speech input, LaTeX preview and five tabs",
    "INT": "Wire all Go units together and prove the full loop with one end-to-end test",
    "QA": "Run tests, load tests and security checks, and report without fixing",
}
QUESTIONS = {
    "effort": {"type": "score", "instructions": "How much focused work does `item` take for an experienced engineer using a coding agent?",
               "criteria": ["Under 30 minutes of routine work", "About one hour of straightforward work", "About two hours with some design decisions", "Half a day with hard design decisions"]},
    "skill": {"type": "choice", "instructions": "Which skill matters most to finish `item` well?",
              "criteria": {"backend": "Go services, data and concurrency", "ai": "LLM prompting, evaluation and grounding", "frontend": "Browser UI", "human_judgment": "Subject expertise that a person must supply", "qa": "Testing and verification", "other": "None of these"}},
}

def ask(key, item):
    body = json.dumps({"model": "typesafe/jev-1.13", "state": {"item": item}, "questions": QUESTIONS}).encode()
    req = urllib.request.Request("https://openrouter.ai/api/alpha/decisions", body,
                                 {"Authorization": f"Bearer {key}", "Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.load(r)

if "--dry-run" in sys.argv:
    print(json.dumps({"units": UNITS, "questions": QUESTIONS}, indent=2)); sys.exit(0)
key = os.environ.get("OPENROUTER_API_KEY")
if not key:
    with open(os.path.expanduser("~/.config/jev/openrouter_key")) as f:
        key = f.read().strip()
rows, cost = [], 0.0
for unit, item in UNITS.items():
    out = ask(key, item); a = out["answers"]; cost += out.get("usage", {}).get("cost", 0) or 0
    conf = min(a["effort"].get("confidence", 0), a["skill"].get("confidence", 0))
    rows.append(f"| {unit} | {a['effort']['score']:.2f} of 3 | {a['skill']['choice']} | {conf:.2f} | {'Claude or Cris decides' if conf < 0.7 else 'Jev'} |")
text = "# Work split from Jev\n\n| Unit | Effort | Skill | Confidence | Decided by |\n| --- | --- | --- | --- | --- |\n" + "\n".join(rows) + f"\n\nCost: ${cost:.4f}\n"
open("rpi/work-split.md", "w").write(text); print(text)
