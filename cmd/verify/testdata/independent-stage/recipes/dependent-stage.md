---
verification: executed
scenario: toy
stage: reads-a-store
requires-stages: []
last-verified: 2026-08-31
verified-against: kno v0.1.4
---

# A page that opens on a stage which does read a store

The same shape as its neighbour, on a stage `run.sh` does NOT declare
independent. This one must still be a finding, or the relaxation has removed
the rule instead of narrowing it.

```bash kno-run scenario=toy stage=reads-a-store
kno report --value-run-id toy-value --db kno.db
```
