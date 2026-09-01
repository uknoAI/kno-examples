# Data provenance — `coding-agent`

**All data in this scenario is synthetic.** None of it is real source code, a real pull request,
a real review comment, or a derivative of any of those. No personal data, no confidential data,
no scraped data, and no content taken from any actual repository.

## The Cases

`evals/cases.jsonl` holds twelve Cases, written by hand for this scenario. They describe the
conventions of a fictional Go service — its review expectations, test layout, error-wrapping
style, and tooling. The conventions are ordinary Go practice stated plainly; they describe no
particular project and were not copied from one.

Four tags, three Cases each: `review`, `testing`, `style`, `tooling`.

## The Pool

`pool/pool.jsonl` holds three synthetic candidate Assets, all `kind: behavior`: a
review-comment voice, a commit-message shape, and a table-test shape. Each is a pattern to
imitate rather than a fact to know, which is what makes it behavior rather than knowledge.

The example fragments inside them — a wrapped `os.Open` error, a commit subject — were written
for this file and are not quotations.

## Licence

Apache-2.0, the same as the rest of this repository.
