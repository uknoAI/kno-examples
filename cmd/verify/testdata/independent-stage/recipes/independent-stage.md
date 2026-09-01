---
verification: executed
scenario: toy
stage: reads-a-file
requires-stages: []
last-verified: 2026-08-31
verified-against: kno v0.1.4
---

# A page that opens on a stage which reads no store

This block is not the scenario's first stage, and it declares no
`requires-stages:`. It must NOT be a finding: `run.sh` declares the stage
independent, and demanding a prior-stage sentence here would make the page
state something false about itself.

```bash kno-run scenario=toy stage=reads-a-file
kno eval inspect --evals cases.jsonl --holdout-frac 0.2
```
