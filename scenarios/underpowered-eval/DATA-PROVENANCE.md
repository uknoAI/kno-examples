# Data provenance — `underpowered-eval`

**All data in this scenario is synthetic.** None of it is a real customer message, a real
support ticket, a real knowledge-base article, or a derivative of any of those. No personal
data, no confidential data, no scraped data.

## The Cases

`evals/cases.jsonl` holds nine Cases: the first nine of
[`support-refunds`](../support-refunds/evals/cases.jsonl), copied without modification.

They are the same synthetic support questions — refunds, shipping, and account — written by hand
for that scenario, and their provenance is stated in full in
[its DATA-PROVENANCE.md](../support-refunds/DATA-PROVENANCE.md).

They are duplicated here rather than referenced because a scenario is a self-contained unit that
`run.sh` copies into a working directory, and because the contract in `internal/scenario` requires
every scenario to carry its own `evals/cases.jsonl`. The duplication is paid for by a test
asserting these nine lines are exactly the first nine of the other scenario's twelve, so the two
cannot drift apart unnoticed — the same posture this repository takes toward the demo fixtures
that `kno demo` embeds.

## The Pool

`pool/pool.jsonl` holds the same three synthetic candidate Assets as `support-refunds`: a refund
policy, a shipping promise, and a brand style guide. Written by hand for that scenario, same
provenance, same assertion.

## Licence

Apache-2.0, the same as the rest of this repository.
