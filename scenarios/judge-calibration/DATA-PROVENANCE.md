# Data provenance — `judge-calibration`

**All data in this scenario is synthetic.** None of it is a real library's policy, a real
enquiry, a real internal document, or a derivative of any of those. No personal data, no
confidential data, no scraped data.

## The Cases

`evals/cases.jsonl` holds 24 Cases: enquiries to an invented public library about opening hours,
borrowing, holds and fines. Written by hand for this scenario by the kno maintainers.

The library does not exist and neither do its rules. Every fact the Cases assert — the twenty-item
adult card, the ten-pence daily charge capped at five pounds, the seven-day hold shelf life, the
one-day grace period — was invented for this file. No real library service's published policy was
copied, paraphrased, or checked against.

## The Pool

`pool/pool.jsonl` holds three synthetic candidate Assets — an hours notice, a lending policy, and
a charges schedule — describing the same invented library. Written by hand for this scenario.

Author: the kno maintainers.

## The calibration set is not ours

This scenario is the only one here whose committed numbers come from data it does not contain.

`kno judge calibrate` reads `starter v1`, a 60-record calibration set **built into the released
binary**, and this scenario asserts figures computed from it: kappa 0.867, the interval
[0.731, 0.967], the four disagreeing records, and the set's own content hash
`8f3fcd006f6d89a1aca3904dc02f8d41c785c4d9faa856ec9ad0e4bf57d53b6e`.

That set is authored by the kno maintainers and lives at
[`judge/testdata/calibration/starter/`](https://github.com/uknoAI/kno/tree/main/judge/testdata/calibration)
in `uknoAI/kno`. Its own provenance rule is stricter than this repository's and is stated there:
every record carries `provenance: authored` or `synthetic`, and **there is no spelling for a
record harvested from a real deployment** — traces are customer data, and the set is public and
permanent.

Two consequences worth stating plainly:

- **If that set changes, this scenario goes red.** That is correct. A changed calibration set
  moves every number on the recipe page, and the committed `set_content_sha256` is what names the
  cause rather than leaving a reader to wonder why kappa moved.
- **A public calibration set is contaminated for training purposes the day it is published.** Any
  model released afterwards may have seen it. It is a regression instrument — it detects a prompt
  change making things worse — not evidence that a judge generalises. That caveat belongs to the
  set, and it travels with any page that quotes its numbers, including this one.

## Licence

Apache-2.0, the same as the rest of this repository.
