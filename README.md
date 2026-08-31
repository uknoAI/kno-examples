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

Today there is one scenario, [`support-refunds`](scenarios/support-refunds/README.md): twelve
Cases across refunds, shipping, account, and billing, and three candidate Assets. All three are
rejected. That is the interesting part — an empty Portfolio is the tool refusing to recommend
something, which is the one screen an ordinary eval harness does not have.

Its six stages carry six recipes, one each: `ci-gate` and `first-baseline` on `baseline`, then
`value-a-pool`, `select-a-portfolio`, `export-a-tuning-set`, `read-the-whole-story`, and
`retention` on the five that follow.

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

- [x] **Migrate the cookbook entries.** All twenty-five migrated entries are here, each with
      front matter, a tier, and — for the vendor pages — every credential it requires named,
      including the `OPENAI_API_KEY` that `--agent openai:...` implies and that the old pages
      never mentioned. One page stayed behind: `check-your-evals` documents `kno eval inspect`,
      which is on `uknoAI/kno@main` and in no release, so no honest tier can be claimed for it
      against a binary that cannot run it. It migrates with the release that ships the command.

- [x] **Tombstone the old paths in `uknoAI/kno`.** Each migrated recipe leaves a one-line stub at
      its old path pointing here. Twenty-two branch-pinned links to
      `github.com/uknoAI/kno/blob/main/docs/cookbook/*.md` live in `uknoAI/kno-www` alone, and
      neither repository's CI checks external links — `make docs` skips `https://` targets and
      the site's Playwright crawl skips external hrefs. Those links would rot silently. The stubs
      are what keep them alive, and they are load-bearing precisely because nothing else is
      watching. A lint in `uknoAI/kno` pins each stub to one line and one link, so a stub cannot
      quietly regrow into a second copy of a page.

- [ ] **Re-point `uknoAI/kno-www`.** The twenty-two links resolve through the stubs today, which
      is a redirect rather than a destination. A link to a stub is worse prose than a link to the
      real page, so the site's references move here — a quality step, not a breakage fix.

- [x] **The fixture-drift detector.** `verify fixtures` compares
      `scenarios/support-refunds/evals/cases.jsonl` and `pool/pool.jsonl` against the copy
      embedded in the Kno binary (`cli/demodata/`) and the copy typed in
      `tapes/quickstart.tape`, and a nightly job runs it against the latest release tag. The
      duplication is deliberate — `kno demo` must work on a plane — and the cost of duplication
      is now paid by a detector rather than by vigilance. This is the job that would have caught
      `processed`/`issued`, and `cmd/verify/testdata/fixtures/drift-processed` is that exact bug,
      kept as a test.

- [ ] **More scenarios.** A coding-agent scenario and an eval-platform scenario, so the vendor
      recipes have more than one shape to be "the same shape as".

## License

Apache-2.0. Scenario data is synthetic and under the same license; see each scenario's
`DATA-PROVENANCE.md`.
