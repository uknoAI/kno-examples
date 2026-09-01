#!/usr/bin/env python3
"""The ablation script you would write in an afternoon.

    python3 naive_ablation.py --evals cases.jsonl --pool pool.jsonl [--json]

This is not a straw man and it is not a joke. It is the honest shape of the
first thing a competent engineer writes when asked "which of these documents
actually helps the agent": for each candidate, score the eval set with it in
context and without it, take the difference of the means, and keep the best
one. Roughly a hundred lines, no dependencies, done before lunch.

It is committed here, in full, run by CI, and asserted against committed
expectations, for one reason: the case for a measurement tool cannot be made
against a description of the alternative. It has to be made against the
alternative, running, on the same data, with its output printed next to the
tool's.

WHAT IT SHARES WITH KNO

The agent and the scorer, exactly. The agent is the same echo agent `kno`'s
`fake:` implements — it returns each Case's expected answer — and the scorer is
the same exact-match comparison. That is what makes the comparison fair: every
difference in the output below is a difference of METHOD, because there is no
difference of model, data, or metric left to explain it.

It also means every delta this script reports is exactly 0.0000, and so is
every delta `kno` reports. Neither program is being flattered. What differs is
what each one does with a table of zeros.

WHAT IT DOES NOT DO, AND WHY EACH OMISSION IS INVISIBLE

  1. No holdout. It scores all N Cases and reports on all N. The number it
     prints is the number it selected on, so it cannot be evidence about
     anything else. `kno baseline` seals a fraction before the first call and
     tells you how many it sealed.

  2. No interval. A difference of means is a point, and a point cannot be
     compared to zero. Nothing here can distinguish "no effect" from "an effect
     I have too few Cases to see" — the distinction two whole scenarios in this
     repository exist to keep visible.

  3. No multiplicity correction. Asking about three assets and keeping the best
     is three chances to be fooled, and the fix (widen every interval by the
     number of questions asked) presupposes an interval, which is #2.

  4. No control arm. Nothing measures whether injecting an asset made some
     other behavior worse, so a document that fixes one tag and breaks another
     shows up as an improvement.

  5. `max()` always returns something. On the table below every delta is
     identical, and this script still names a winner — the first line of
     pool.jsonl, because that is what `max` does with a tie. Add a `> 0`
     threshold and the tie is rejected; on real, noisy data the largest of
     three noise draws is reliably above zero, and the threshold accepts it.
     That is the winner's curse, and it is not fixed by a threshold.

None of the five is hard. Each is a normal afternoon's work. The claim this
script exists to support is not that they are hard; it is that they are five,
that each one is silent when you get it wrong, and that a table like the one
below looks exactly as convincing with all five missing as with none.
"""

import argparse
import json
import sys


def read_jsonl(path):
    records = []
    with open(path) as handle:
        for number, line in enumerate(handle, start=1):
            line = line.strip()
            if not line:
                continue
            try:
                records.append(json.loads(line))
            except ValueError as err:
                raise SystemExit("{}:{}: {}".format(path, number, err))
    return records


def agent(case, context=None):
    """The same echo agent kno's `fake:` implements.

    `context` is accepted and ignored, which is not a shortcut: an agent that
    returns the expected answer cannot be improved by anything put in front of
    it. `fake:` has the same property, which is why no scenario in this
    repository can show an asset earning its place.
    """
    del context
    return case["expected"]


def score(answer, expected):
    """Exact match, the same goal kno is run with here."""
    return 1.0 if answer == expected else 0.0


def mean(values):
    return sum(values) / len(values) if values else 0.0


def ablate(cases, assets):
    without = mean([score(agent(c), c["expected"]) for c in cases])
    rows = []
    for asset in assets:
        with_asset = mean(
            [score(agent(c, context=asset["content"]), c["expected"]) for c in cases]
        )
        rows.append(
            {
                "asset_id": asset["id"],
                "with_asset": with_asset,
                "without_asset": without,
                "delta": with_asset - without,
            }
        )
    return rows


def main(argv):
    parser = argparse.ArgumentParser(description="a naive context ablation")
    parser.add_argument("--evals", required=True, help="Cases, one JSON object per line")
    parser.add_argument("--pool", required=True, help="candidate assets, one JSON object per line")
    parser.add_argument("--json", action="store_true", help="machine-readable output")
    args = parser.parse_args(argv)

    cases = read_jsonl(args.evals)
    assets = read_jsonl(args.pool)
    if not cases or not assets:
        raise SystemExit("need at least one case and one asset")

    rows = ablate(cases, assets)
    # The line this whole scenario is about. `max` over a tie returns the first
    # element, so on a table of identical deltas the winner is decided by the
    # order of the lines in pool.jsonl.
    best = max(rows, key=lambda row: row["delta"])

    if args.json:
        json.dump(
            {
                "cases_scored": len(cases),
                "cases_held_back": 0,
                "assets": rows,
                "winner": {"asset_id": best["asset_id"], "delta": best["delta"]},
            },
            sys.stdout,
            indent=2,
        )
        sys.stdout.write("\n")
        return 0

    print("naive ablation over {} cases, {} assets".format(len(cases), len(assets)))
    print("  agent   echo (returns each case's expected answer)")
    print("  scorer  exact match")
    print("")
    print("{:<18}{:>8}{:>10}{:>9}".format("ASSET", "WITH", "WITHOUT", "DELTA"))
    for row in rows:
        print(
            "{:<18}{:>8.4f}{:>10.4f}{:>+9.4f}".format(
                row["asset_id"], row["with_asset"], row["without_asset"], row["delta"]
            )
        )
    print("")
    print("winner: {} ({:+.4f})".format(best["asset_id"], best["delta"]))
    print("scored {} of {} cases; 0 held back".format(len(cases), len(cases)))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
