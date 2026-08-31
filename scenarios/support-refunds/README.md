# `support-refunds`

A small support agent that answers questions about refunds, shipping, accounts, and billing,
and three candidate Assets someone might feed it: a refund policy, a shipping promise, and a
brand style guide.

The question the scenario asks is the one the whole product asks: **do those three Assets earn
their place?** The answer here is no, for all three, and that is the interesting part.

```
scenarios/support-refunds/
  evals/cases.jsonl     12 Cases across four tags
  pool/pool.jsonl        3 candidate Assets
  run.sh                 six stages: baseline, value, select, export, report, purge
  expected/              one projected JSON document per stage, plus quotations.json
  DATA-PROVENANCE.md     who wrote this data, and the assertion that it is synthetic
```

Run it:

```sh
sh scenarios/support-refunds/run.sh /tmp/sr
```

It needs a released `kno` on `PATH` and nothing else — no credential, no network after the
install, no environment variable. It finishes in about a second.

## What to look for

**The score reads `1.000` and that is honest rather than flattering.** The agent is `fake:`,
which answers every Case with exactly what the Case expects. The run is showing that a run
*happened*, was scored against a declared Goal, cost nothing, and sealed a holdout — not that
anybody's agent is perfect.

**Every delta is `+0.0000`, and it arrives with an interval.** Because `fake:` is
deterministic, injecting an Asset cannot move the answer, so no choice of Asset content could
change this. The interval is the point: it is what lets `select` say `no-effect` — "we measured
this and it did nothing" — rather than the much weaker `underpowered`, which would mean "we
could not tell".

**The Portfolio comes back empty, and that is the product working.** `select` rejects all three
Assets with a reason on the record. An empty Portfolio is a legal, first-class outcome: the tool
refusing to recommend something is the one screen an ordinary eval harness does not have.

## The Case count is load-bearing

**Twelve Cases. Not nine, not eleven.**

The dev/holdout split is keyed on the Case id, so twelve Cases put eight in dev and hold four
back. `value` then reserves 0.3 of dev for the control arm — `int(8 * 0.3)` = **two** control
Cases, which is the minimum an interval can be computed from.

At nine Cases the arithmetic collapses: six dev, `int(6 * 0.3)` = **one** control Case, every
interval comes back `nil`, the value table renders "sample too small or ragged to form an
interval" for every row, and `select` rejects on `underpowered` rather than `no-effect`. The
three paragraphs above would then be claims the screen never showed.

Twelve also keeps the honest warning on the first screen: at a 0.2 holdout fraction the holdout
is four Cases, below the minimum for a meaningful interval, so the baseline renderer prints the
too-small-holdout caveat — and `expected/quotations.json` asserts it does.

Do not shrink these files. `expected/` will fail, which is the intended behaviour, but read what
the output actually says before regenerating it.

## Do not "improve" the numbers

Picking flattering Asset content would buy a prettier screen and zero additional truth, and it
would turn a demonstration into a promise the product cannot keep on the reader's own data. The
same prohibition is written into `uknoAI/kno`'s `cli/demodata/README.md` and
`tapes/quickstart.tape`, for the same reason.

## This data is synthetic, and that is a limitation

Every Case and every Asset here was written by hand for this repository (see
`DATA-PROVENANCE.md`). A synthetic support corpus is cleaner than any real one: no duplicate
tickets, no contradictory policies, no half-answered threads, no HTML. A reader may
over-generalize from how smoothly this runs. Real pools are messier and the intervals on real
data are wider.

## Relationship to `uknoAI/kno`

The same twelve Cases and three Assets are embedded in the kno binary, at `cli/demodata/`, and
recorded in `tapes/quickstart.tape`. That duplication is deliberate — `kno demo` must work on a
plane, on an air-gapped box, and behind a corporate proxy, so it cannot fetch a sibling
repository at runtime — and the cost of duplication is paid by a detector rather than by
vigilance. See the roadmap in this repository's `README.md`.

## A cross-platform float difference, worth reporting upstream

The scenario is bit-for-bit reproducible on one machine: two runs on the same binary produce
byte-identical output, and CI asserts that with `--repeat 2`. Across architectures it is not.
`value`'s interval bounds agree to eleven significant digits and then diverge:

```
darwin/arm64   -0.39597252156206514
linux/amd64    -0.39597252156200174
```

That is a floating-point difference in an iterative computation, not a docs finding — and it is
the reason `expected/value.json` declares the bound to four decimal places, which is what the
CLI renders (`[-0.3960, +0.3960]`) and what the recipe quotes. The projection asserts to the
precision it declares and no further; a bound that moves at the fourth decimal still fails.

Asserting all seventeen digits would make the expectation a statement about the runner's libm,
so the scenario would be red on one architecture forever and the file would end up regenerated
per-platform — a golden file that has stopped being a test.

## An observation worth reporting upstream

`kno export --json` at v0.1.2 prints `"select_run_id": ""` even though `--select-run-id` was
supplied and the export plainly found the Portfolio. It is not in `expected/export.json`,
because committing a projection over a field whose value looks wrong would cement the bug as
the contract. The rendered output and every other field are correct.
