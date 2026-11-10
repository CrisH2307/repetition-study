#!/usr/bin/env python3
"""Build perturbations.jsonl and split.json from content/unit.json. Deterministic; rerun after claims change.

    python3 eval/data/make_perturbations.py

Each rule changes one claim in one way and is tagged with its error type. A perturbed
claim is no longer true, so the checker should flag it (anything but "supported").
Split: claims are hashed into dev (about 30%) or test, so the two never share a claim.
Tuning prompts or models is allowed on dev only (review 4.6).
"""
import hashlib
import json
import os

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..")
OUT = os.path.join(ROOT, "eval", "data")


def swap(a, b):
    return lambda s: s.replace(a, "\0").replace(b, a).replace("\0", b)


def swap_tail(a, b, sep):
    """Swap only after the last sep, so a symmetric restatement does not stay true."""
    def f(s):
        i = s.rfind(sep)
        return s if i < 0 else s[:i] + swap(a, b)(s[i:])
    return f


def sub(a, b):
    return lambda s: s.replace(a, b, 1)


def drop_independence(s):
    return s.replace("an independent ", "a ", 1) if "an independent " in s else s.replace("independent ", "", 1)


# ponytail: plain string rules, each reviewed by hand to leave a claim that is false.
# Symmetric swaps (that stay true) were removed. Claims no rule touches get no perturbation.
D, S, G, F = "dropped_condition", "swapped_parameter", "wrong_sign", "wrong_formula"
RULES = [
    (D, "drop independence", drop_independence),
    (D, "drop normalisation", sub("; π must also be normalised so its entries sum to 1", ", so any solution is the distribution")),
    (D, "drop both", sub("If that sum and the sum", "If the sum")),
    (D, "drop the sale probability", sub(" and each buys with probability 0.30", "")),
    (D, "drop positive", sub("is positive recurrent", "is recurrent")),
    (D, "drop split by independence", sub(", split it by independence,", ",")),
    (D, "drop finiteness premise", sub("When that sum S is finite, π0", "For any birth-death process, π0")),
    (D, "drop the waiting-room limit", sub(" with room for two customers", "")),
    (D, "drop the faster barber", sub("If the barber works twice as fast, the", "The")),
    (D, "drop t from the exponent", sub("P(t) = e^{tG}", "P(t) = e^{G}")),
    (S, "swap λ1 and λ2 in the result", swap_tail("λ1", "λ2", " = ")),
    (S, "swap the exponent rate", sub("e^{-λ2 t}, and", "e^{-λ1 t}, and")),
    (S, "swap birth and death rates", lambda s: s if s.startswith("If that sum") else swap("λ_", "β_")(s)),
    (S, "swap p00 and p01", swap("p00", "p01")),
    (S, "swap π1 and π2 in the mean", sub("π1 + 2π2", "π2 + 2π1")),
    (S, "total rate becomes one rate", sub("divide by the Poisson(λ1+λ2)", "divide by the Poisson(λ1)")),
    (S, "uses 1-p for the sale probability", sub("rate 20·0.3", "rate 20·0.7")),
    (S, "arrival rate for sale rate", sub("with mean 1/6 hour", "with mean 1/20 hour")),
    (S, "arrival rate for sale rate in count", sub("Poisson of rate 6 per hour", "Poisson of rate 20 per hour")),
    (S, "mean used as rate", sub("q10 = q21 = 4", "q10 = q21 = 0.25")),
    (S, "minimum becomes maximum", sub("The minimum min(T1, T2)", "The maximum max(T1, T2)")),
    (G, "drop minus in exponent", sub("e^{-", "e^{")),
    (G, "minus becomes plus", sub(" - ", " + ")),
    (G, "plus becomes minus", sub(" + ", " - ")),
    (G, "drop minus on diagonal", lambda s: s.replace("-g_ii", "g_ii")),
    (G, "less than becomes greater than", sub(" < ", " > ")),
    (G, "survival becomes distribution", sub("P(T2 > t)", "P(T2 < t)")),
    (G, ">= becomes <=", sub(">= 10", "<= 10")),
    (G, "finite becomes infinite", sub("is finite", "is infinite")),
    (G, "diverge becomes converge", sub("diverge", "converge")),
    (G, "negative coefficient flipped", sub("-(n+1)", "(n+1)")),
    (G, "at least becomes at most", sub("is at least 3", "is at most 3")),
    (G, "grows becomes shrinks", sub("business grows by", "business shrinks by")),
    (G, "sum becomes difference", sub("then X(t)+Y(t) is", "then X(t)-Y(t) is")),
    (F, "wrong denominator", sub("λ1/(λ1+λ2)", "λ1/λ2")),
    (F, "race is a coin flip", sub("P(T1 < T2) = λ1/(λ1+λ2)", "P(T1 < T2) = 1/2")),
    (F, "rates multiply", sub("Exp(λ1+λ2)", "Exp(λ1·λ2)")),
    (F, "rates average", sub("is Exp(λ1+λ2)", "is Exp((λ1+λ2)/2)")),
    (F, "density without rate", sub("density λ1 e^{-λ1 t}", "density e^{-λ1 t}")),
    (F, "mean equals rate", sub("has mean 1/λ", "has mean λ")),
    (F, "rate equals mean", sub("converts to the rate 1/m", "converts to the rate m")),
    (F, "superposed rate is the max", sub("rate λ1+λ2.", "rate max(λ1, λ2).")),
    (F, "superposed rate multiplies", sub("rate λ1+λ2.", "rate λ1λ2.")),
    (F, "Poisson count without t", sub("(λt)^k", "λ^k")),
    (F, "Poisson mean without t", sub("is Poisson(λt)", "is Poisson(λ)")),
    (F, "k factorial dropped", sub("(λt)^k/k!", "(λt)^k/k")),
    (F, "both counts equal k", sub("{X(1)=k, Y(1)=0}", "{X(1)=k, Y(1)=k}")),
    (F, "integrate becomes plug in mean", sub("integrating P(X(t)=k) against the density of T", "plugging E[T] into P(X(t)=k)")),
    (F, "density becomes CDF", sub("against the density of T", "against the distribution function of T")),
    (F, "conditioning removed", sub("divide by the Poisson(λ1+λ2) probability of the total", "stop there, without dividing")),
    (F, "column-vector balance", sub("πG = 0", "Gπ = 0")),
    (F, "discrete-time balance", sub("row vector solving πG = 0", "row vector solving πG = π")),
    (F, "normalise to zero", sub("entries sum to 1", "entries sum to 0")),
    (F, "long-run share inverted", sub("in state i is π_i", "in state i is 1/π_i")),
    (F, "sojourn is uniform", sub("is exponential with rate -g_ii", "is uniform with rate -g_ii")),
    (F, "mean sojourn equals rate", sub("so its mean is 1/(-g_ii)", "so its mean is -g_ii")),
    (F, "π0 without the 1", sub("π0 = 1/(1+S)", "π0 = 1/S")),
    (F, "π_n adds instead of multiplies", sub("π_n = π0 (λ", "π_n = π0 + (λ")),
    (F, "null becomes transient", sub("the process is null recurrent", "the process is transient")),
    (F, "products not simplified", sub("simplify the rate products (they often telescope) and test the resulting series for convergence", "check only that λ_n < β_n for large n")),
    (F, "terms to zero is enough", sub("test the resulting series for convergence", "check that the terms of the series go to zero")),
    (F, "transition matrix is tG", sub("P(t) = e^{tG}", "P(t) = tG")),
    (F, "wrong decay rate", lambda s: s.replace("e^{-5t}", "e^{-4t}")),
    (F, "thinning adds", sub("20·0.3 = 6", "20+0.3 = 20.3")),
    (F, "expected sales equal the rate", sub("E[N(8)] = 48", "E[N(8)] = 6")),
    (F, "mean is the rate", sub("with mean 1/6 hour", "with mean 6 hours")),
    (F, "tail sum off by one", sub("sum_{n=0}^{9}", "sum_{n=0}^{10}")),
    (F, "memoryless step wrong", sub("= P(N(1) <= 2)", "= P(N(2) <= 8)")),
    (F, "sojourn rate inverted 3", sub("is Exp(3)", "is Exp(1/3)")),
    (F, "sojourn rate inverted 2", sub("is Exp(2)", "is Exp(1/2)")),
    (F, "mean sojourn inverted 3", sub("E[U1] = 1/3", "E[U1] = 3")),
    (F, "mean sojourn inverted 2", sub("E[U1] = 1/2", "E[U1] = 2")),
    (F, "hitting time without removing state", sub("(-G_{-4})^{-1} 1", "(-G)^{-1} 1")),
    (F, "Euler step uses h squared", sub("0.1·0.05", "0.01·0.05")),
    (F, "Euler square becomes double", sub("0.994^2", "2·0.994")),
    (F, "recursion ratio inverted", sub("(n+1)/(2(n+2))", "(n+2)/(2(n+1))")),
    (F, "mean count misses weight", sub("π1 + 2π2", "π1 + π2")),
    (F, "entering states wrong", sub("0 or 1 customers, π0 + π1", "1 or 2 customers, π1 + π2")),
    (F, "long-run share depends on start", sub("in state i is π_i.", "in state i is π_i, but only when the chain starts in state i.")),
    (F, "hitting vector wrong", sub("(1, 1, 1)^T", "(1/3, 1/3, 1/3)^T")),
    (F, "speed-up ignored", sub("(64, 24, 9)/97", "(16, 12, 9)/37")),
    (F, "minimum not exponential", sub("is Exp(λ1+λ2)", "is not exponential")),
    (F, "stationary weights inverted", sub("π = (16, 12, 9)/37", "π = (9, 12, 16)/37")),
]


def bucket(s):
    return int(hashlib.sha256(s.encode()).hexdigest(), 16) % 10


def main():
    with open(os.path.join(ROOT, "content", "unit.json")) as f:
        unit = json.load(f)
    items, seen = [], set()
    for c in unit["concepts"]:
        for cl in c["claims"]:
            for etype, rule, fn in RULES:
                text = fn(cl["text"])
                if text == cl["text"] or text in seen:
                    continue
                seen.add(text)
                items.append({"id": f"p{len(items) + 1:03d}", "concept_id": c["id"], "claim_id": cl["id"],
                              "error_type": etype, "rule": rule, "text": text, "expect": "flagged"})
    with open(os.path.join(OUT, "perturbations.jsonl"), "w") as f:
        for it in items:
            f.write(json.dumps(it, ensure_ascii=False) + "\n")

    expl = [json.loads(l) for l in open(os.path.join(OUT, "explanations.jsonl")) if l.strip()]
    dev_expl = set()
    for cid in sorted({e["concept_id"] for e in expl}):  # one dev explanation per concept
        dev_expl.add(min((e["id"] for e in expl if e["concept_id"] == cid), key=lambda i: hashlib.sha256(i.encode()).hexdigest()))
    split = {
        "note": "Tune prompts and models on dev only. Report gate numbers on test only.",
        "explanations": {"dev": sorted(dev_expl), "test": sorted(e["id"] for e in expl if e["id"] not in dev_expl)},
        "perturbations": {"dev": [i["id"] for i in items if bucket(i["claim_id"]) < 3],
                          "test": [i["id"] for i in items if bucket(i["claim_id"]) >= 3]},
    }
    with open(os.path.join(OUT, "split.json"), "w") as f:
        json.dump(split, f, indent=2)
        f.write("\n")
    by = {}
    for it in items:
        by[it["error_type"]] = by.get(it["error_type"], 0) + 1
    print(len(items), "perturbations", by, "dev", len(split["perturbations"]["dev"]), "test", len(split["perturbations"]["test"]))
    assert len(items) >= 100, "need at least 100 perturbations"


if __name__ == "__main__":
    main()
