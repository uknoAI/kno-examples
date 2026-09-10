---
verification: flags-only
owner: "@kno-maintainers"
verified-against: kno v0.2.2
last-manual-verification: 2026-09-10
credentials: [TOGETHER_API_KEY, OPENAI_API_KEY]
---

# Bridge a tuning set: does the behavior actually transfer?

`kno select` can put a `behavior` Asset in the Portfolio with a `tuning_set` destination, and
`kno export` will render the training file. Neither of them has fine-tuned anything. The Asset's
delta was measured by putting it **in the prompt**, and "helps in context" is not the same claim
as "helps when trained on".

`kno bridge` is the stage that closes that gap. It fine-tunes a small proxy model on the
tuning-set Assets a Select run chose and measures leave-one-group-out: does each failure
cluster's Assets actually transfer under fine-tuning, and does tuning on them regress anything
else.

**It is the most expensive command in the tool**, and the only one that can spend money you
cannot get back. Read this page before you arm it.

## Read the plan first — it is free

Without `--bridge`, the command plans and submits nothing:

```bash
kno bridge --select-run-id nightly-select --pool pool.jsonl --db kno.db \
  --tuner together:meta-llama/Llama-3-8b \
  --price-train-per-mtok 0.20 --price-serve-per-minute 0.01
```

It reads the Portfolio, forms the ablation groups from the source Value run's failure clusters,
renders every group's training file **with the exact bytes `kno export --destination tuning_set`
would write**, counts tokens, and prices them — locally, with zero network calls and zero dollars
spent.

That last detail is the one to lean on. The un-armed plan is not an estimate of a different
thing; it renders the same bytes the armed run would upload. What you price is what you would
send.

`--price-train-per-mtok` and `--price-serve-per-minute` are required until a pricing table row
exists for your `--tuner`. That is deliberate: an unpriced model would produce a plan with no
number on it, and a plan with no number is not a thing you can approve.

## What arming costs

With `--bridge`, the plan is the same and a job is submitted for every group in it, once
confirmed.

- **Each job is charged when it is submitted and cannot be un-submitted.** There is no cancel
  that gets the money back.
- **Hosting a tuned model for its eval passes is charged per minute per endpoint, including
  while idle.**

The caps are separate flags because they bound different things, and the defaults are
conservative:

| Flag | Default | Bounds |
|---|---:|---|
| `--max-cost-usd` | 0 (unlimited) | total spend across every job |
| `--bridge-max-groups` | 6 | refuses the run rather than merging groups beyond this many leave-one-out jobs |
| `--bridge-max-live-endpoints` | 1 | endpoints live at once — each bills per minute per replica, idle included |
| `--bridge-max-serve-minutes` | 30 | per endpoint; beyond it the endpoint is torn down and its group reported `unknown` |
| `--bridge-timeout` | 1h | how long to wait for one job to reach a terminal status |

`--bridge-cancel-on-timeout` cancels a job that outlives `--bridge-timeout` instead of leaving it
running. Without it the job is never cancelled and `--resume` keeps polling it — which is the
right default for a job you have already paid for, and the wrong one for a job that is silently
holding an endpoint open.

Set `--max-cost-usd` on every armed run. Nothing else here bounds the total.

## The two refusals you will hit first

Both are free, both are reachable from a normal store, and both were checked by hand against
v0.2.2.

**No tuner:**

```
error: required flag(s) "tuner" not set
```

`--tuner` is `scheme:model` — `together:meta-llama/Llama-3-8b`, `openai:gpt-5.6-terra`. There is
no default, because there is no model it would be safe to guess.

**No failure clusters:**

```
error: the command cannot use the input it was given: run <id> recorded no clusters
  fix: re-run `kno value`; no clusters to group by means nothing routed to failure tags
```

The ablation groups *are* the failure clusters from the Value run. A run where nothing failed has
nothing to group by, and therefore nothing to ablate. This is the refusal every scenario in this
repository hits, for a reason worth stating plainly below.

## Why this page is `flags-only`

Every `kno` command on it is checked nightly against the released binary's own flag surface. None
of it is executed, and unlike most `flags-only` pages that is not because a vendor credential is
missing — it is because **this command cannot be exercised for free, by construction, twice
over**:

1. **No failure clusters.** Every committed scenario runs against `fake:`, which answers each
   Case with exactly what the Case expects. Nothing fails, so `kno value` records no clusters, and
   `kno bridge` refuses with the error above. Getting clusters needs an agent that is sometimes
   wrong, which means a paid provider.
2. **`exec:` cannot stand in.** The obvious free workaround — a local script that answers some
   Cases wrongly — does not work: `exec:` declares `ContextInject: false` and the Value stage
   refuses `exec` arms for injected measurement. So the Value run that would produce the clusters
   cannot be made with it.

And even past those, arming the command submits fine-tuning jobs that charge on submission. A
nightly that could do that is a nightly that should not exist.

So the honest ceiling here is the flag surface plus two hand-checked refusals, and this page
claims exactly that. See [VERIFICATION.md](../VERIFICATION.md) for what the tier does and does
not license you to believe.

## Next

- [Export a tuning set](export-a-tuning-set.md) — the bytes this command re-renders and prices.
- [What a run costs](what-it-costs.md) — the arithmetic for the stages that come before it.
- [Validate a portfolio against the untouched holdout](validate-the-portfolio.md) — the other
  stage that measures a Portfolio rather than an Asset.
