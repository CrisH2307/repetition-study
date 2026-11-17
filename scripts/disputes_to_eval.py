#!/usr/bin/env python3
"""Turn upheld disputes into dev-split eval cases (D39). No model calls.

    DATABASE_URL=... go run ./cmd/export -disputes > disputes.csv   # then add the columns below
    python3 scripts/disputes_to_eval.py disputes.csv
    go run ./cmd/evalcheck --split dev --disputes content/private/eval/disputes.jsonl ...

Input is a CSV, or a JSON/JSONL list of the Go DisputeCase type, with:
  dispute_id, concept_id, claim_id, student_text, expected_label, outcome
Only rows with outcome "upheld" and a valid expected_label are kept. Extra columns are ignored.
Output goes to content/private/eval/disputes.jsonl by default: that folder is gitignored
because the student's words are user data. Course source text is never read or written.
Never writes to eval/data (the frozen test split lives there).
"""
import argparse, csv, json, os, sys

LABELS = {"supported", "partial", "contradicted", "absent"}


def rows(path):
    if path.endswith((".json", ".jsonl")):
        txt = open(path, encoding="utf-8").read().strip()
        data = json.loads(txt) if txt.startswith("[") else [json.loads(l) for l in txt.splitlines() if l.strip()]
        for d in data:
            d.setdefault("outcome", "upheld")  # UpheldDisputes only returns upheld ones
        return data
    return list(csv.DictReader(open(path, encoding="utf-8", newline="")))


def convert(src):
    out, skipped, seen = [], [], set()
    for r in src:
        did = (r.get("dispute_id") or "").strip()
        lab = (r.get("expected_label") or "").strip().lower()
        claim = r.get("claim_id") or ""
        why = (None if (r.get("outcome") or "").strip().lower() == "upheld" else "not upheld") \
            or (None if lab in LABELS else "no valid expected_label") \
            or (None if claim.strip() else "no claim_id (cannot label one claim)") \
            or (None if (r.get("concept_id") or "").strip() else "no concept_id") \
            or (None if (r.get("student_text") or "").strip() else "no student_text") \
            or (None if did and did not in seen else "missing or repeated dispute_id")
        if why:
            skipped.append((did or "?", why))
            continue
        seen.add(did)
        out.append({"id": "d-" + did, "concept_id": r["concept_id"].strip(), "claim_id": claim.strip(),
                    "text": r["student_text"], "expected": lab})
    return out, skipped


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("export")
    ap.add_argument("--out", default="content/private/eval/disputes.jsonl")
    a = ap.parse_args()
    if os.path.abspath(a.out).startswith(os.path.abspath("eval/data") + os.sep):
        sys.exit("refusing to write into eval/data (frozen test split)")
    cases, skipped = convert(rows(a.export))
    os.makedirs(os.path.dirname(os.path.abspath(a.out)), exist_ok=True)
    with open(a.out, "w", encoding="utf-8") as f:
        for c in cases:
            f.write(json.dumps(c, ensure_ascii=False) + "\n")
    print(f"wrote {len(cases)} cases to {a.out}; skipped {len(skipped)}")
    for i, w in skipped:
        print(f"  skipped {i}: {w}")


if __name__ == "__main__":
    main()
