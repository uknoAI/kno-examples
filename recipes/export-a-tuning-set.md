---
verification: executed
scenario: support-refunds
stage: export
requires-stages: [baseline, value, select]
last-verified: 2026-08-31
verified-against: kno v0.1.2
---
# Export a tuning set

What you're asking: **turn the Portfolio into files a downstream pipeline can consume.** `kno select` chose the assets; `kno export` renders them into the destination grammar you name, atomically and idempotently.

## Before you start

You need one recorded run and the pool:

- **A Select run** — from `kno select` (same `--db`), whose Portfolio holds the selected assets and their measurements.
- **A pool** — the same `assets.jsonl` you valued and selected. The Portfolio carries measurements, not content; the pool is the only place content lives, so export cannot render without it.
- **A destination** — one of the three the design ships:
  - `context` — a context-pack manifest plus the rendered pack
  - `knowledge_base` — a manifest plus a human-readable instruction list (writable knowledge-base adapters arrive with v0.2; the manifest says so)
  - `tuning_set` — **OpenAI chat format JSONL**, the exact shape the Tuner adapters will parse

This page is verified as the fourth stage of the `support-refunds` scenario, and the transcript
below is that scenario's real output. The command reads a Portfolio that the baseline, value and
select stages wrote, so `sh scenarios/support-refunds/run.sh` is what produces it — pasted into a
fresh shell against an empty store, there is no Select run to export and it will say so.

## Run it

```bash kno-run scenario=support-refunds stage=export
kno export --select-run-id sr-select --pool pool.jsonl \
  --destination tuning_set --out tuning.jsonl \
  --db kno.db --run-id sr-export
```

`sr-select` is the run id the previous stage recorded.

## Read the report

```
Export run sr-export (completed)
  destination  tuning_set
  wrote        tuning.jsonl (0 assets, 117 bytes)
  manifest     tuning.jsonl.manifest.md

The artifact is a pure function of the Portfolio and the pool: re-exporting is byte-identical, and export never mutates a destination.
```

That is real output, asserted against the binary by
`scenarios/support-refunds/expected/quotations.json`.

## What the file contains

One JSON object per line, in the chat format every hosted fine-tuning API
accepts, in selection order:

```json
{"messages":[{"role":"assistant","content":"Refunds are issued within 5 business days."}]}
```

**Every example carries an `assistant` turn.** That is not a stylistic choice —
it is the requirement. Without an assistant message there is no target to train
on, and a provider's file-validation step rejects the upload. If your Assets are
plain demonstration text, Kno wraps each one as a single assistant message. If an
Asset is *already* chat JSONL, it passes through re-marshaled to one line, and is
**refused** if it carries no assistant turn — rather than shipped as a line that
would fail validation later, further from the cause.

Empty or whitespace-only content is refused for the same reason: a zero-example
demonstration is not trainable at any price, and paying a provider to discover
that is a paid no-op.

> Kno versions before **v0.1.6** emitted `{"role":"user"}` and no assistant turn,
> so every tuning set they wrote was rejected by every provider. If you have an
> artifact from an earlier version, re-export it rather than editing it — the
> manifest beside it records the Portfolio it came from.

**Zero assets is a legal export, and the file is not empty.** The scenario's Portfolio selected
nothing — every Asset's corrected interval crossed zero — so the tuning set carries no examples,
and the 117 bytes are the manifest header that records *which* Portfolio produced it and that it
chose nothing. An export that silently wrote a zero-byte file would lose the difference between
"this Portfolio is empty" and "this export never ran".

## The contract

- **Refusal, not overwrite.** An existing file at `--out` is refused unless you pass `--force`. Nothing is ever silently mutated.
- **Atomic writes.** The artifact (and the manifest beside it at `<out>.manifest.md`) is written temp-then-rename, so a crash mid-export leaves no partial file.
- **Idempotent.** Re-exporting the same Portfolio produces byte-identical files — safe to rerun in a scheduled job, and safe to diff.
- **Derived, never authoritative.** The artifact is a pure function of the Portfolio and the pool; export never changes the Portfolio. The record is the source.

## Machine-readable

```sh
kno export --select-run-id <id> --pool assets.jsonl \
  --destination context --out pack.md --json
```

prints `run_id`, `status`, `select_run_id`, `destination`, `asset_count`, `bytes_written`,
`path`, and `manifest_path` — the fields a CI step or a `jq` pipeline branches on.

**`select_run_id` comes back empty in kno v0.1.2** even when `--select-run-id` was given — the
field was declared in the contract and never populated. It is fixed on `uknoAI/kno@main` and
carries the Select run's id from the next release. It is deliberately absent from
`expected/export.json`: pinning a field whose value is already known to be about to change would
schedule a red build for release day and teach everyone to regenerate the expectation without
reading it. Branch on `run_id` and `status` against v0.1.2.

## Next

- [Choose a portfolio under budget](select-a-portfolio.md) — the step before this one.
- [Read the whole story with `kno report`](read-the-whole-story.md) — the export page folds the gaps record into the report.
- Validate (upcoming) measures the Portfolio as a set against the untouched holdout — the honest number that selection-time estimates point at.
