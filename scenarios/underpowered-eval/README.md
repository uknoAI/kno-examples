# `underpowered-eval`

The same support agent as [`support-refunds`](../support-refunds/README.md), the same three
candidate Assets, and three fewer Cases.

That is the entire difference, and it changes the answer:

| Scenario | Cases | Every Asset rejected, because |
|---|---:|---|
| `support-refunds` | 12 | `no-effect` — we measured it, and it did nothing |
| `underpowered-eval` | 9 | `underpowered` — we could not tell |

```
scenarios/underpowered-eval/
  evals/cases.jsonl      9 Cases — the first nine of support-refunds, unmodified
  pool/pool.jsonl        the same 3 candidate Assets
  run.sh                 three stages: baseline, value, select
  expected/              one projected JSON document per stage, plus quotations.json
  DATA-PROVENANCE.md     who wrote this data, and the assertion that it is synthetic
```

Run it:

```sh
sh scenarios/underpowered-eval/run.sh /tmp/ue
```

It needs a released `kno` on `PATH` and nothing else.

## Why this exists

Both scenarios reject all three Assets. From a distance the two runs look identical — `Selected
0`, `Rejected 3` — and they mean opposite things.

`no-effect` is a measurement. An interval was formed, it contained zero, and the tool is
reporting that injecting this Asset did not move the score. `underpowered` is the refusal to
pretend a measurement happened: too few Cases survived into the reserve for any interval to
form, so there is nothing to report and the tool says so instead of rounding its ignorance down
to a number.

A reader who cannot tell those apart will read `Rejected` as "measured and found wanting" every
time. That is the mistake this pair exists to make impossible, and the reason the distinction is
worth a whole second scenario rather than a sentence in the first one's prose.

It is also the distinction that is hardest to keep working by accident. Nothing about
`no-effect` looks broken if it silently starts appearing where `underpowered` belongs — the run
stays green, the report stays plausible, and the tool quietly starts overclaiming. This scenario
is the assertion that it has not.

## Where the boundary is

Measured against kno v0.1.3, not assumed:

| Cases | dev / holdout | `select` |
|---:|---:|---|
| 8 | 6 / 2 | `underpowered` |
| **9** | **6 / 3** | **`underpowered`** — this scenario |
| 10 | 7 / 3 | `no-effect` |
| 12 | 8 / 4 | `no-effect` — `support-refunds` |

The control arm reserves `int(dev * 0.3)`, and an interval needs two. Ten Cases is the smallest
count that clears it. Nine is the largest that fails, which is deliberately where this scenario
sits: one Case from the boundary is where a change in the reserve arithmetic shows up first.

## Why the difference is only the Case count

Same Pool, same Goal, same seeds, same concurrency, same holdout fraction, same flags. The nine
Cases are the first nine of `support-refunds`, byte for byte, and
[a test](../../cmd/verify/scenarios_test.go) asserts that — so the pair stays a controlled
comparison rather than two unrelated runs that happen to disagree.

If the two scenarios ever drift apart in anything but Case count, the comparison stops meaning
what this page says it means, and the test is what notices.

## What this does not show

The same caveat as `support-refunds`, and for the same reason: the agent is `fake:`, which
answers every Case with exactly what the Case expects. No choice of Asset content could move
this score, so neither scenario can demonstrate an Asset earning its place.

That is not a gap in the scenarios. Offline, it is not expressible: `fake:` cannot be moved by
injected context, `exec:` is refused by the Value stage for injected measurement, and every
adapter that does accept injected context spends money. A scenario showing a non-empty Portfolio
would have to be a `manual` page against a paid provider, which is a different tier and a
different promise.
