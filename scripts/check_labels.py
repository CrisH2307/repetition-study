#!/usr/bin/env python3
"""Compare Cris's and Dino's labels and help them settle differences.

    python3 scripts/check_labels.py               report only
    python3 scripts/check_labels.py --init-final  also write eval/data/labels_final.csv
                                                  (agreed rows filled, disagreements left empty)

Labels: supported, partial, contradicted, absent. Plain-language output on purpose.
"""
import csv
import json
import os
import sys

D = "eval/data"
OK = {"supported", "partial", "contradicted", "absent"}


def read_labels(path):
    with open(path, newline="") as f:
        return {r["id"].strip(): (r["label"].strip().lower(), r.get("note", "")) for r in csv.DictReader(f)}


def main():
    texts = {}
    with open(f"{D}/explanations.jsonl") as f:
        for line in f:
            if line.strip():
                e = json.loads(line)
                texts[e["id"]] = e["text"]
    cris, dino = read_labels(f"{D}/labels_cris.csv"), read_labels(f"{D}/labels_dino.csv")
    problems = []
    for who, lab in (("Cris", cris), ("Dino", dino)):
        empty = [i for i in texts if i in lab and lab[i][0] == ""]
        if empty:
            problems.append(f"{who}: {len(empty)} rows have no label yet ({empty[0]} to {empty[-1]})")
        for i in texts:
            if i not in lab:
                problems.append(f"{who}: row {i} is missing")
            elif lab[i][0] and lab[i][0] not in OK:
                problems.append(f"{who}: {i} has '{lab[i][0]}', which is not one of {', '.join(sorted(OK))}")
    if problems:
        print("Not ready to compare. Fix these first:")
        print("\n".join("  - " + p for p in problems))
        return 1

    same = [i for i in texts if cris[i][0] == dino[i][0]]
    diff = [i for i in texts if i not in same]
    n = len(texts)
    print(f"You agree on {len(same)} of {n} explanations ({100 * len(same) // n}%).")
    # Cohen's kappa: agreement beyond what luck would give. 1 is perfect, 0 is no better than luck.
    pe = sum((sum(cris[i][0] == k for i in texts) / n) * (sum(dino[i][0] == k for i in texts) / n) for k in OK)
    if pe < 1:
        print(f"Agreement beyond luck (kappa): {(len(same) / n - pe) / (1 - pe):.2f}")
    if diff:
        print(f"\nPlease settle these {len(diff)} and type the answer into {D}/labels_final.csv:")
        for i in diff:
            print(f"\n{i}: Cris says {cris[i][0]}, Dino says {dino[i][0]}")
            print(f"  Text: {texts[i]}")
            for who, lab in (("Cris", cris), ("Dino", dino)):
                if lab[i][1].strip():
                    print(f"  {who}'s note: {lab[i][1].strip()}")

    final_path = f"{D}/labels_final.csv"
    if "--init-final" in sys.argv:
        if os.path.exists(final_path):
            print(f"\n{final_path} already exists, so I did not overwrite it.")
        else:
            with open(final_path, "w", newline="") as f:
                w = csv.writer(f)
                w.writerow(["id", "label"])
                for i in texts:
                    w.writerow([i, cris[i][0] if i in same else ""])
            print(f"\nWrote {final_path}. Agreed rows are filled in; fill the {len(diff)} empty ones together.")
    if os.path.exists(final_path):
        fin = read_labels(final_path)
        bad = [i for i in texts if fin.get(i, ("",))[0] not in OK]
        print(f"\nlabels_final.csv: {'COMPLETE' if not bad else 'NOT complete, still empty or invalid: ' + ', '.join(bad)}")
        return 1 if bad else 0
    return 0


if __name__ == "__main__":
    sys.exit(main())
