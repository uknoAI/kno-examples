---
verification: executed
scenario: toy
stage: doctor
requires-stages: []
last-verified: 2026-08-31
verified-against: kno v0.1.2
---

# A recipe whose quoted block drifted from run.sh

```bash kno-run scenario=toy stage=doctor
kno doctor --json --verbose
```
