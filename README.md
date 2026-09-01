# kno-examples

Committed **scenarios** for [Kno](https://github.com/uknoAI/kno) — real Cases, a real Pool,
committed expected output — and the machinery that runs them against the released binary every
night.

The point is narrow and worth stating plainly: **before this repository existed, no CI job
anywhere had ever run a command from Kno's documentation.** The docs were checked for godoc
coverage and for relative-link integrity, both real gates, neither of which executes anything.
The same twelve-Case scenario was maintained by hand in three places and had already drifted —
`README.md` said refunds are "processed within 5 business days" while `tapes/quickstart.tape`
said "issued". Nothing could tell you which was right.

Now something runs them.

```sh
git clone https://github.com/uknoAI/kno-examples
cd kno-examples
make install-kno                    # skip if you already have one on your PATH
PATH="$PWD/bin:$PATH" sh scenarios/support-refunds/run.sh /tmp/sr
```

That needs a released `kno` and nothing else — no key, no network after the install, no
environment variable. `make install-kno` fetches one into `./bin` through Kno's own
`install.sh`, signature and all, so the quickstart exercises the path a user exercises.

It runs baseline → value → select → export → report → purge against the built-in `fake:` agent
in about a second, and every number it prints is asserted against a committed expectation.

## What is here

```
recipes/           task-shaped pages, each declaring how far it has been verified
scenarios/         committed Cases, Pool, run.sh, and expected output
cmd/verify         the runner: the lint, the flag check, the scenario executor, the fixture detector
.github/workflows  PR checks, the nightly two-job drift detector, vendor-smoke
VERIFICATION.md    exactly what each badge claims and does not claim
```

## The three tiers

Every recipe declares one, in front matter, and no recipe may omit it. The full contract is in
[VERIFICATION.md](VERIFICATION.md); the short version:

| Tier | What CI does | What the page shows |
|---|---|---|
| `executed` | Runs the commands end-to-end against the released binary and compares the result to committed expectations | ✔ the only tick, the only green, the only use of the word "Verified" |
| `flags-only` | Checks every command shape against the released binary's own `--help` and `kno doctor --json` | • neutral. Identical in icon and colour to `manual` |
| `manual` | Nothing | • neutral. Identical in icon and colour to `flags-only` |

**`flags-only` looks exactly as unverified as `manual`, on purpose.** With respect to the
question a reader is actually asking — *does this recipe work* — it is. The difference between
the two is a claim about what was checked, which is a sentence, not a colour. Wording is not a
control: a reader forms a belief from a badge's shape before reading the sentence under it. So
[a test](internal/render/render_test.go) asserts the two tiers emit the identical icon and the
identical colour token and differ only in their text, and fails if a future stylesheet gives
`flags-only` a colour of its own.

Every claim also expires. A CI-written verification older than 30 days, or a hand-check older
than 180, renders a staleness banner on the page itself. Nobody has to remember; the page tells
on itself.

## Scenarios

A scenario is the unit of execution, and that choice does most of the work. `select`,
`export`, `report`, and `retention` all read a SQLite store an earlier stage wrote — so if the
*recipe* were the unit, none of them could ever be executed. Because `run.sh` performs all six
stages against one store, each recipe asserts against its own stage and four pages move from
"cannot be checked" to "checked".

**But `executed` does not mean `standalone`.** A page verified as stage 3 says so, right next to
its badge, and names the script to run first — generated from a declared `requires-stages:`
field rather than remembered. A reader who pastes stage three into a fresh shell gets an empty
store, and a green tick over that failure is exactly the false confidence this repository exists
to destroy.

Today there are seven, in three groups — a pair contrasting on verdict, a pair contrasting on
shape, and three written to answer a question a reader actually arrives with.
[`support-refunds`](scenarios/support-refunds/README.md) has twelve Cases across refunds,
shipping, account, and billing, and three candidate Assets. All three are rejected. That is the
interesting part — an empty Portfolio is the tool refusing to recommend something, which is the
one screen an ordinary eval harness does not have.

[`underpowered-eval`](scenarios/underpowered-eval/README.md) is the same Cases with three
removed, and it also rejects all three Assets — for a different reason. `support-refunds` says
`no-effect`: an interval was formed and it contained zero. `underpowered-eval` says
`underpowered`: too few Cases survived into the reserve for any interval to form, so there is
nothing to report. One is a measurement; the other is the refusal to pretend there was one. They
look alike on screen, they mean opposite things, and a reader who conflates them will read every
`Rejected` as "measured and found wanting".

The second pair varies the *shape* rather than the verdict, because the vendor recipes needed
more than one thing to be "the same shape as".
[`coding-agent`](scenarios/coding-agent/README.md) answers questions about a codebase's
conventions, and its three Assets are demonstrations rather than documents — `kind: behavior`,
the only kind that faces the fine-tuning bridge, which until it landed no committed Asset
exercised at all. [`eval-platform`](scenarios/eval-platform/README.md) is an LLM-as-judge: each
Case input is *another model's answer* and the expected output is a grade, which is the shape a
Braintrust dataset or a Langfuse trace actually has. Its Pool is mixed, so routing has to decide
rather than send everything one way, and it exports to `context` where `coding-agent` exports to
`tuning_set`.

`support-refunds`'s six stages carry six recipes, one each: `ci-gate` and `first-baseline` on
`baseline`, then `value-a-pool`, `select-a-portfolio`, `export-a-tuning-set`,
`read-the-whole-story`, and `retention` on the five that follow.

The third group exists because of a gap the first two could not close. The four pairs above
demonstrate that the loop runs and reports honestly; none of them answers the questions a reader
turns up with, which — asked in almost this order — are *why not script this myself*, *how many
Cases do I need*, *what will it cost*, and *I don't have an eval set at all*. Each of the three
new scenarios is built so that one of those has an answer CI runs nightly rather than an answer
a page asserts.

[`diy-ablation`](scenarios/diy-ablation/README.md) commits the hundred-line context ablation an
engineer writes in an afternoon and **executes it in CI**, over the same 24 Cases, the same 3
Assets, the same agent and the same scorer as the three `kno` stages beside it. Every delta is
zero in both — `fake:` guarantees that — and the script still prints `winner: auth-guide`,
because `max()` returns something. The identity of that winner is decided by the order of lines
in `pool.jsonl`. Kno prints `Rejected 3 … crosses zero`. Same data, same numbers, opposite
conclusions, and the difference is method.

[`power-analysis`](scenarios/power-analysis/README.md) reads one eval set at 12, 40 and 160
Cases and reports what each size could have detected: a separable effect of 6.35, then 0.51,
then 0.25. A score lives in [0, 1], so 6.35 means *nothing is detectable at all*. Two of the
five checks flag at twelve and none flags at 160, and they clear at different sizes — an eval
set can be big enough to tell you which behaviour is failing and still too small to tell you
whether you fixed it. It costs nothing: `kno eval inspect` makes no LLM call and constructs no
agent.

[`transcript-mining`](scenarios/transcript-mining/README.md) starts one step earlier than every
other scenario here — with transcripts rather than with an eval set. One `kno mine` reads a
directory holding a JSONL chat export and a CSV ticket export, auto-sniffs both, and writes 18
Cases each carrying `derived`, a derivation note and a source ref. Then `kno eval inspect` flags
three of five checks on what came back, because mining does not invent behaviour tags and
eighteen Cases is not enough. That is the honest shape of a first day, and the tool says so
rather than the documentation.

## How verification stays honest

- **Assertions are on `--json`, never on rendered text — and on a projection, not a full
  golden.** `expected/<stage>.json` holds only the fields a recipe's prose makes a claim about.
  An additive CLI change passes; a removed or renamed field fails. A full-document golden would
  churn on every run id and timestamp, so it would be regenerated reflexively and rubber-stamped
  — a golden file that has stopped being a test.
- **Where prose quotes CLI output, the quotation itself is the assertion.** If Kno reformats a
  line, the page goes red and the *prose* is what changes. That is the correct direction of
  blame.
- **Recipes never re-type commands.** A `kno-run` block quotes a marked region of `run.sh` and a
  lint asserts byte-identity. There is exactly one copy of every flag in this repository. That is
  the mechanism that stops `processed`/`issued` from happening again — not a review convention,
  a failing build.
- **Determinism is bought with flags, not with luck.** `run.sh` pins `--run-id`, `--seed`,
  `--routing-seed`, `--concurrency`, and `--holdout-frac`, and CI runs it twice and
  byte-compares. Anything that varies is a Kno bug, and the scenario is where we find out.
- **Nothing here spends money automatically.** Every nightly command runs against `fake:`,
  offline. Vendor recipes get flag-shape checks, which are free. What exercises a vendor API is
  `vendor-smoke`, which is `workflow_dispatch`-only, gated on a GitHub environment with required
  reviewers, and capped three independent ways.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). In short: DCO sign-off (`git commit -s`), no CLA;
Apache-2.0; every recipe declares a tier; every scenario ships a `DATA-PROVENANCE.md` asserting
its data is synthetic; a Tier A recipe ships committed expectations and passes the runner in PR
CI.

## Roadmap

This repository was scaffolded with one scenario end-to-end, to prove the machinery before
committing to it. In order:

- [x] **Migrate the cookbook entries.** All twenty-six migrated entries are here, each with
      front matter, a tier, and — for the vendor pages — every credential it requires named,
      including the `OPENAI_API_KEY` that `--agent openai:...` implies and that the old pages
      never mentioned. Two pages stayed behind, for one reason: each documented a command
      that was on `uknoAI/kno@main` and in no release, so no honest tier could be claimed for it
      against a binary that cannot run it. `check-your-evals` has since migrated — v0.1.4 ships
      `kno eval inspect` — leaving `calibrate-a-judge`, which documents `kno judge calibrate` and
      migrates with the release that ships it.

- [x] **Tombstone the old paths in `uknoAI/kno`.** Each migrated recipe leaves a one-line stub at
      its old path pointing here. Twenty-two branch-pinned links to
      `github.com/uknoAI/kno/blob/main/docs/cookbook/*.md` live in `uknoAI/kno-www` alone, and
      neither repository's CI checks external links — `make docs` skips `https://` targets and
      the site's Playwright crawl skips external hrefs. Those links would rot silently. The stubs
      are what keep them alive, and they are load-bearing precisely because nothing else is
      watching. A lint in `uknoAI/kno` pins each stub to one line and one link, so a stub cannot
      quietly regrow into a second copy of a page.

- [x] **Re-point `uknoAI/kno-www`.** All twenty-two references now point at `recipes/` here
      rather than resolving through a stub in `uknoAI/kno`. A link to a stub is worse prose than
      a link to the real page, so this was a quality step rather than a breakage fix. The site's
      Playwright crawl skips external hrefs, so nothing on either side checks these links — each
      of the ten distinct targets was fetched by hand instead.

- [x] **The fixture-drift detector.** `verify fixtures` compares
      `scenarios/support-refunds/evals/cases.jsonl` and `pool/pool.jsonl` against the copy
      embedded in the Kno binary (`cli/demodata/`) and the copy typed in
      `tapes/quickstart.tape`, and a nightly job runs it against the latest release tag. The
      duplication is deliberate — `kno demo` must work on a plane — and the cost of duplication
      is now paid by a detector rather than by vigilance. This is the job that would have caught
      `processed`/`issued`, and `cmd/verify/testdata/fixtures/drift-processed` is that exact bug,
      kept as a test.

- [x] **More scenarios.** Seven now. Four in two pairs:
      [`underpowered-eval`](scenarios/underpowered-eval/README.md) contrasts with
      `support-refunds` on *verdict* — the same Cases, three fewer of them, ending in
      `underpowered` rather than `no-effect`: a measurement refused next to a measurement made.
      [`coding-agent`](scenarios/coding-agent/README.md) and
      [`eval-platform`](scenarios/eval-platform/README.md) contrast on *shape*: a Pool of
      `behavior` demonstrations exported to the tuning bridge, and an LLM-as-judge whose Case
      inputs are another model's answers, carrying a mixed Pool exported to context. Before
      them every committed Asset was `knowledge` and every vendor page deferred to a
      customer-support recipe for its flow, including the ones about source code and eval
      datasets.

      **What a new scenario can and cannot show.** Nothing committed here can demonstrate an
      Asset earning its place, and that is a property of the tool rather than of the scenarios.
      `fake:` answers every Case with exactly what the Case expects, so injected context cannot
      move its score; `exec:` declares `ContextInject: false` and the Value stage refuses exec
      arms for injected measurement; and every adapter that does accept injected context spends
      money, which the nightly may not. So an `executed` scenario always ends in an empty
      Portfolio, and a non-empty one would have to be a `manual` page against a paid provider —
      a different tier making a different promise. Design new scenarios around which *verdict*
      they reach, not around whether an Asset wins.

- [x] **Scenarios for the questions readers actually arrive with.** The four above prove the
      loop runs and reports honestly. None of them answers *why not script this myself*, *how
      many Cases do I need*, *what will it cost*, or *I have no eval set* — so
      [`diy-ablation`](scenarios/diy-ablation/README.md),
      [`power-analysis`](scenarios/power-analysis/README.md) and
      [`transcript-mining`](scenarios/transcript-mining/README.md) were written to give each of
      those an answer a machine re-checks nightly, and
      [`why-not-diy`](recipes/why-not-diy.md),
      [`power-and-sample-size`](recipes/power-and-sample-size.md),
      [`what-it-costs`](recipes/what-it-costs.md) and
      [`mine-your-transcripts`](recipes/mine-your-transcripts.md) are the `executed` pages over
      them. Three more pages —
      [`analyze-in-a-notebook`](recipes/analyze-in-a-notebook.md),
      [`from-your-warehouse`](recipes/from-your-warehouse.md) and
      [`orchestration`](recipes/orchestration.md) — cover the parts nothing here can execute,
      and say so in their tier rather than in a footnote.

      They obey the constraint above rather than working around it: every one of them ends in an
      empty Portfolio, and each demonstrates something that is a function of the Case count, the
      Asset count or the file format rather than of any model's behaviour — which is exactly the
      class of claim that survives the move to a paid provider unchanged.

      `diy-ablation` is the one that needed a new precedent. It executes a **non-`kno`** program
      in a stage — `naive_ablation.py`, committed in full — because the standing objection to a
      measurement tool cannot be answered by describing the alternative. It has to be answered by
      running it. That makes `python3` a prerequisite for one scenario out of seven, checked and
      named in its `run.sh` rather than discovered as a stack trace.

- [x] **A two-level subcommand in the checker.** `kno eval inspect` shipped in v0.1.4 and is the
      first command whose flags live on a child: `kno eval --help` lists only `-h`, so resolving
      one word after `kno` reported `--evals` as removed. `OpenBinary` now discovers children
      from the binary the same way it discovers the root list, and
      [`cmd/verify/testdata/nested-subcommand/`](cmd/verify/testdata/nested-subcommand/) holds
      both halves of the claim: the child's real flags must pass, and an invented one must still
      fail. A change that fixed only the first half would have passed a one-sided test while
      checking nothing.

      This also unblocked a migration. `check-your-evals` stayed in `uknoAI/kno` because no
      honest tier could be claimed for a command no release shipped; v0.1.4 ships it, and the
      page is now [`recipes/check-your-evals.md`](recipes/check-your-evals.md), `executed`
      against `power-analysis`, with a one-line tombstone stub left at its old path.

## License

Apache-2.0. Scenario data is synthetic and under the same license; see each scenario's
`DATA-PROVENANCE.md`.
