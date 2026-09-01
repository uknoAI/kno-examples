# Data provenance — `diy-ablation`

**All data in this scenario is synthetic.** None of it is a real developer question, a real
support ticket, a real documentation page, or a derivative of any of those. No personal data, no
confidential data, no scraped data.

## The Cases

`evals/cases.jsonl` holds 24 Cases: developer questions about the authentication, rate limiting,
webhooks and error semantics of an API that does not exist. Written by hand for this scenario by
the kno maintainers.

The API is invented, and so is every fact the Cases assert about it — the header names, the
limits, the retry schedule, the status codes. Nothing here was copied from, paraphrased from, or
checked against any real product's documentation. Any resemblance to a real API's conventions is
because REST has conventions.

Twenty-four Cases, six per behaviour, is a count chosen to land on a `no-effect` verdict rather
than an `underpowered` one — see [`underpowered-eval`](../underpowered-eval/README.md) for why
that distinction is worth a scenario of its own, and where the boundary sits. This scenario
needs `no-effect`, because "we measured it and it did nothing" is the answer the naive script's
`winner:` line has to be read against; `underpowered` would have made the contrast a different
and weaker one.

## The Pool

`pool/pool.jsonl` holds three synthetic candidate Assets: an authentication guide, a rate-limits
page, and a changelog. They describe the same invented API. Written by hand for this scenario.

The changelog is deliberately the third line of the file, and `auth-guide` deliberately the
first: the naive script's `max()` returns the first element of a tie, so the identity of its
"winner" is a fact about line order, which is exactly what this scenario is about. Reordering
the file changes the script's recommendation and changes nothing about Kno's.

Author: the kno maintainers.

## The script

`naive_ablation.py` was written by the kno maintainers for this scenario. It is not a copy or
adaptation of anyone's real ablation script, and it is not a caricature of one: its agent and
its scorer are the same agent and scorer the `kno` stages beside it run with, which is what
makes the comparison a comparison.

## Licence

Apache-2.0, the same as the rest of this repository, and the same for `naive_ablation.py`.
