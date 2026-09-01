# Data provenance — `transcript-mining`

**All data in this scenario is synthetic.** This assertion carries more weight here than in any
other scenario in this repository, so it is worth being explicit about what is being asserted
and why the risk is higher.

`transcripts/chat.jsonl` and `transcripts/tickets.csv` look exactly like a real support export.
That is the point of them — a scenario about reading production transcripts is worthless if its
transcripts do not have the shape of production transcripts. It is also precisely the condition
under which somebody, at some point, pastes in a real one.

**None of these threads happened.** No message here is a real employee's words, a real support
engineer's reply, a real ticket, or a paraphrase of any of those. There is no company whose
helpdesk this is. Every fact asserted in them — the fifteen-minute lockout, the 400-day drive
retention, the fifty-gigabyte mailbox cap, the self-service certificate portal — was invented for
this file. No personal data, no confidential data, no scraped data.

Author: the kno maintainers.

## Why this is a human gate and not a lint

No checker can tell a plausible invented helpdesk thread from a real one. That is stated as
policy in [CONTRIBUTING.md](../../CONTRIBUTING.md) and it applies with full force here: a pull
request that adds transcripts to this directory is asserting authorship of them, and there is no
mechanism behind that assertion but review. A scenario whose whole subject is "point this at
your logs" is the most likely place in this repository for the rule to be broken by accident,
and the accident is unrecoverable — a published transcript stays published whether or not the
commit is later removed.

If you are adding transcripts here: write them. Do not redact real ones. Redaction is not
synthesis, and a redacted thread still carries its structure, its timing, and its subject.

## The mined Cases

`evals/cases.jsonl` is not a hand-authored file. It is the **output** of `kno mine` over
`transcripts/`, committed so that a change in what mining produces is a visible diff rather than
a silent one, and asserted byte-for-byte by
[a test](../../cmd/verify/scenarios_test.go). Its provenance is therefore exactly the provenance
of the transcripts above, plus a deterministic transformation.

Every record in it carries `derived: true`, a `derivation_note` naming the mode, and a
`source_ref` naming the file it came from. That is Kno's own weak-label provenance, not
something this repository adds.

## The Pool

`pool/pool.jsonl` holds three synthetic candidate Assets — an access runbook, a device runbook,
and a policy digest — written by hand for this scenario, describing the same invented
organisation. No real internal document was copied, paraphrased, or summarised.

Author: the kno maintainers.

## Licence

Apache-2.0, the same as the rest of this repository.
