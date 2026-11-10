#!/usr/bin/env python3
"""Compare Dino's and the classmate's claim labels, and freeze the final set.

    python3 scripts/check_labels.py            report agreement and list disagreements
    python3 scripts/check_labels.py --freeze   write labels_final.csv and record its SHA-256 in FROZEN.md
    python3 scripts/check_labels.py --selftest run the built-in test with fake label files

One row per (explanation id, claim id). Labels: supported, partial, contradicted, absent.
Disagreements are settled together in labels_resolved.csv (written for you on the first run).
Freezing is refused while kappa is below 0.6 or any disagreement is unsettled.
Output is plain language on purpose: Dino reads it.
"""
import csv
import datetime
import hashlib
import json
import os
import sys
import tempfile

LABELS = ("supported", "partial", "contradicted", "absent")
MIN_KAPPA = 0.6
WHO = (("Dino", "labels_dino.csv"), ("Classmate", "labels_classmate.csv"))


def read(path):
    with open(path, newline="") as f:
        return {(r["id"].strip(), r["claim_id"].strip()): r for r in csv.DictReader(f)}


def kappa(a, b, keys):
    """Cohen's kappa over keys. 1 is perfect, 0 is no better than luck. None if undefined."""
    n = len(keys)
    po = sum(a[k] == b[k] for k in keys) / n
    pe = sum((sum(a[k] == c for k in keys) / n) * (sum(b[k] == c for k in keys) / n) for c in set(a.values()) | set(b.values()))
    return None if pe == 1 else (po - pe) / (1 - pe)


def fmt(k):
    return "n/a (too few labels of this kind)" if k is None else f"{k:.2f}"


def run(d, freeze=False, out=print):
    texts = {}
    with open(os.path.join(d, "explanations.jsonl")) as f:
        for line in f:
            if line.strip():
                e = json.loads(line)
                texts[e["id"]] = e["text"]
    sheets = {who: read(os.path.join(d, name)) for who, name in WHO}
    keys = list(sheets["Dino"])
    problems = []
    for who, rows in sheets.items():
        if set(rows) != set(keys):
            problems.append(f"{who}: the rows do not match the other sheet. Do not add or delete rows.")
        empty = [k for k in rows if not rows[k]["label"].strip()]
        if empty:
            problems.append(f"{who}: {len(empty)} rows have no label yet (first one: {empty[0][0]}, claim {empty[0][1]})")
        for k, r in rows.items():
            lab = r["label"].strip().lower()
            if lab and lab not in LABELS:
                problems.append(f"{who}: {k[0]} claim {k[1]} has '{lab}', which is not one of {', '.join(LABELS)}")
            if k[0] not in texts:
                problems.append(f"{who}: {k[0]} is not in explanations.jsonl")
    if problems:
        out("Not ready to compare. Fix these first:")
        out("\n".join("  - " + p for p in problems))
        return 1

    a = {k: sheets["Dino"][k]["label"].strip().lower() for k in keys}
    b = {k: sheets["Classmate"][k]["label"].strip().lower() for k in keys}
    diff = [k for k in keys if a[k] != b[k]]
    k_all = kappa(a, b, keys)
    out(f"You agree on {len(keys) - len(diff)} of {len(keys)} claim labels.")
    out(f"Agreement beyond luck (Cohen's kappa), all labels: {fmt(k_all)}")
    for lab in LABELS:  # one label against the rest
        out(f"  {lab:<13} {fmt(kappa({k: a[k] == lab for k in keys}, {k: b[k] == lab for k in keys}, keys))}")

    res_path = os.path.join(d, "labels_resolved.csv")
    resolved = {k: r["label"].strip().lower() for k, r in read(res_path).items()} if os.path.exists(res_path) else {}
    if diff:
        out(f"\nYou disagree on {len(diff)}. Settle each one together:")
        for k in diff:
            out(f"\n{k[0]}, claim {k[1]}: Dino says {a[k]}, classmate says {b[k]}")
            out(f"  Explanation: {texts[k[0]]}")
            out(f"  Claim: {sheets['Dino'][k]['claim']}")
            for who, rows in sheets.items():
                if rows[k].get("note", "").strip():
                    out(f"  {who}'s note: {rows[k]['note'].strip()}")
        if not os.path.exists(res_path):
            with open(res_path, "w", newline="") as f:
                w = csv.writer(f)
                w.writerow(["id", "claim_id", "dino", "classmate", "label"])
                for k in diff:
                    w.writerow([k[0], k[1], a[k], b[k], ""])
            out(f"\nI wrote {res_path}. Type the label you both agree on in its label column.")

    if not freeze:
        return 0
    final = os.path.join(d, "labels_final.csv")
    if os.path.exists(final):
        out(f"\nNot frozen: {final} already exists. The set is frozen once; ask Cris.")
        return 1
    if k_all is None or k_all < MIN_KAPPA:
        out(f"\nNot frozen: kappa is {fmt(k_all)}, below {MIN_KAPPA}. Copy this output to Cris and stop.")
        return 1
    open_ = [k for k in diff if resolved.get(k) not in LABELS]
    if open_:
        out(f"\nNot frozen: {len(open_)} disagreements have no agreed label in {res_path} yet.")
        return 1
    with open(final, "w", newline="") as f:
        w = csv.writer(f)
        w.writerow(["id", "claim_id", "label"])
        for k in keys:
            w.writerow([k[0], k[1], resolved[k] if k in diff else a[k]])
    sha = hashlib.sha256(open(final, "rb").read()).hexdigest()
    with open(os.path.join(d, "FROZEN.md"), "w") as f:
        f.write("# Frozen labels\n\n"
                f"- File: labels_final.csv\n- SHA-256: {sha}\n- Rows: {len(keys)}\n"
                f"- Kappa (independent labels, before settling): {k_all:.3f}\n"
                f"- Disagreements settled: {len(diff)}\n- Frozen on: {datetime.date.today().isoformat()}\n\n"
                "Do not edit labels_final.csv. Any change needs a new freeze approved by Cris.\n")
    out(f"\nFrozen. labels_final.csv SHA-256 {sha} is recorded in FROZEN.md.")
    return 0


def selftest():
    def make(d, dino, mate, resolved=None):
        with open(os.path.join(d, "explanations.jsonl"), "w") as f:
            for i in ("e01", "e02"):
                f.write(json.dumps({"id": i, "concept_id": "c", "text": "t " + i}) + "\n")
        keys = [(e, c) for e in ("e01", "e02") for c in ("c1", "c2", "c3", "c4", "c5")]
        for name, labs in (("labels_dino.csv", dino), ("labels_classmate.csv", mate)):
            with open(os.path.join(d, name), "w", newline="") as f:
                w = csv.writer(f)
                w.writerow(["id", "claim_id", "explanation", "claim", "label", "note"])
                for k, lab in zip(keys, labs):
                    w.writerow([k[0], k[1], "x", "claim " + k[1], lab, ""])
        if resolved:
            with open(os.path.join(d, "labels_resolved.csv"), "w", newline="") as f:
                w = csv.writer(f)
                w.writerow(["id", "claim_id", "dino", "classmate", "label"])
                w.writerows(resolved)

    s, p, c, a = LABELS
    base = [s, s, s, p, p, c, c, a, a, a]
    quiet = lambda *_: None
    # Same labels: kappa 1, freezes, hash recorded.
    with tempfile.TemporaryDirectory() as d:
        make(d, base, base)
        assert run(d, freeze=True, out=quiet) == 0
        sha = hashlib.sha256(open(os.path.join(d, "labels_final.csv"), "rb").read()).hexdigest()
        assert sha in open(os.path.join(d, "FROZEN.md")).read()
        assert run(d, freeze=True, out=quiet) == 1, "must not freeze twice"
    # Empty or invalid label: refuses to compare.
    with tempfile.TemporaryDirectory() as d:
        make(d, base, [""] + base[1:])
        assert run(d, out=quiet) == 1
        make(d, base, ["yes"] + base[1:])
        assert run(d, out=quiet) == 1
    # One disagreement: high kappa, but freeze waits for the settled label.
    one = [p] + base[1:]
    with tempfile.TemporaryDirectory() as d:
        make(d, base, one)
        log = []
        assert run(d, freeze=True, out=log.append) == 1
        assert any("Not frozen" in m for m in log)
        assert os.path.exists(os.path.join(d, "labels_resolved.csv"))
        make(d, base, one, resolved=[["e01", "c1", s, p, s]])
        assert run(d, freeze=True, out=quiet) == 0
        rows = list(csv.reader(open(os.path.join(d, "labels_final.csv"))))
        assert rows[1] == ["e01", "c1", s]
    # Low agreement: refuses to freeze.
    with tempfile.TemporaryDirectory() as d:
        make(d, base, [a, a, c, s, c, s, p, s, p, c])
        assert run(d, freeze=True, out=quiet) == 1
        assert not os.path.exists(os.path.join(d, "labels_final.csv"))
    # Kappa against a hand-computed value: 2x2 table [[2,1],[1,2]] gives po=2/3, pe=1/2, kappa=1/3.
    x = dict(enumerate([1, 1, 1, 0, 0, 0]))
    y = dict(enumerate([1, 1, 0, 1, 0, 0]))
    assert abs(kappa(x, y, list(x)) - 1 / 3) < 1e-9
    print("selftest passed")
    return 0


if __name__ == "__main__":
    if "--selftest" in sys.argv:
        sys.exit(selftest())
    sys.exit(run("eval/data", freeze="--freeze" in sys.argv))
