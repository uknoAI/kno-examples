# Data provenance — `power-analysis`

**All data in this scenario is synthetic.** None of it is a real customer message, a real
support ticket, a real internal document, or a derivative of any of those. No personal data, no
confidential data, no scraped data. No real company's warehouse, mart, schedule or access policy
is described here; the forty table names are invented.

## The Cases

`evals/cases.jsonl` holds 160 Cases about an invented analytics warehouse — what a row in a mart
means, which SQL dialect it runs on, how often it lands, and who may read it.

They were **written by a program**, and the program is committed beside them at
[`evals/generate.py`](evals/generate.py). Running it reproduces `cases.jsonl` byte for byte, and
[a test](../../cmd/verify/scenarios_test.go) asserts that, so the file and its provenance cannot
drift apart.

That is a stronger provenance claim than a sentence, not a weaker one. "Synthetic" here is not
an assertion a reader has to take on trust about 160 lines they would have to read; it is a
forty-row table of invented mart names and four question templates, in one file, that anybody
can read in a minute and re-run.

The reason for generating rather than writing: nothing this scenario demonstrates depends on the
Cases' wording. `fake:` answers every Case with what the Case expects, so every score is 1.000
whatever the text says. What the numbers depend on is the count and the tag.

Author: the kno maintainers.

## The Pool

`pool/pool.jsonl` holds three synthetic candidate Assets — a metrics glossary note, a SQL
dialect note, and an access-policy note — written by hand for this scenario. They describe the
same invented warehouse. No real document was copied, paraphrased, or summarised.

Author: the kno maintainers.

## The smaller eval sets

`cases-12.jsonl` and `cases-40.jsonl` are not committed. `run.sh` creates them with `head -n`
from `cases.jsonl` at run time, so they are the same synthetic data and carry the same
provenance by construction.

## Licence

Apache-2.0, the same as the rest of this repository.
