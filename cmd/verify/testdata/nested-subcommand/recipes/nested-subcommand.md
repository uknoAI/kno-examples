---
verification: flags-only
owner: "@nobody"
verified-against: kno v0.1.4
last-manual-verification: 2026-08-31
---

# A recipe using a two-level subcommand

`kno eval inspect` is the first command in the surface whose flags live on a
child rather than on the word after `kno`. Resolving one word would look
`--evals` up in `kno eval --help`, which lists only `-h`, and report every line
here as broken.

```bash
kno eval inspect --evals cases.jsonl --holdout-frac 0.2 --json
```

The second block is the finding this corpus exists to prove is still reachable:
a real parent, a real child, a flag the child does not have.

```bash
kno eval inspect --evals cases.jsonl --maximum-holdout-frac 0.2
```
