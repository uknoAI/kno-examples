---
verification: manual
owner: "@nobody"
last-manual-verification: 2026-08-31
---

# A recipe pointing at a page that is not there

The link below is the finding. [Value a pool of assets](value-a-pool.md) resolves relative to
this file, and there is no such file in this corpus.

These must NOT be findings: [the project](https://github.com/uknoAI/kno), an
[anchor](#a-recipe-pointing-at-a-page-that-is-not-there), and this one, which does exist:
[itself](broken-link.md).

A fenced block is not scanned, because a code sample may contain bracket-paren text that
nobody can click:

```python
things = [a for a in items](not_a_link.md)
```
