# What the badges mean

The epistemics page for the documentation, written in the register of Kno's
[*What the numbers mean*](https://github.com/uknoAI/kno/blob/main/docs/what-the-numbers-mean.md).
Kno refuses to report a delta without its interval; this page is the same discipline applied to
the claim a documentation page makes about itself.

There are three tiers. Every recipe declares exactly one, in front matter, and a lint refuses a
recipe that omits it. There is no default and no fourth value.

---

## ✔ `executed`

**What CI did.** Installed the latest released `kno` through the project's own `install.sh`
(with `cosign` present, so the signature was verified and not merely the checksum), ran the
scenario's `run.sh` end-to-end, and compared what came back to the committed expectations. Twice,
byte-comparing the two runs.

**What that licenses you to believe.** These commands run, in this order, on a machine with
nothing but a released `kno`, and produce the output shown. If the binary stops doing that, this
page goes red the same night.

**What it does not license you to believe.**

- *That the numbers mean anything about your data.* The scenario runs against `fake:`, which
  answers every Case with what the Case expects. The score reads `1.000` by construction. What
  is verified is that the loop runs and reports honestly, not that any agent is good.
- *That you can paste one block in isolation.* An `executed` recipe may be verified as stage 3
  of a scenario, reading a store the earlier stages wrote. When it is, the page says so next to
  the badge and names the script to run first. That sentence is generated from the recipe's
  `requires-stages:` field, so it cannot be forgotten.
- *That the page's prose is correct.* Verification covers commands and output, not the
  explanations around them. A wrong sentence next to a right command still passes.

**Expiry.** 30 days. The nightly rewrites the date after a green run, so a banner on an
`executed` page means the nightly stopped running — which is itself the finding.

---

## • `flags-only`

**What CI did.** Extracted every `kno` invocation on the page and checked it against the released
binary's own surface: the subcommand exists, every long flag appears in `kno <subcommand>
--help`, and every `--agent` / `--evals` / `--pool` scheme prefix appears in `kno doctor --json`.
No key, no network, no vendor contacted.

**What that licenses you to believe.** The commands are spelled correctly for this build. That
catches the overwhelmingly most common form of documentation rot — a renamed or removed flag —
and it catches it the night it happens.

**What it does not license you to believe.** Anything else. It proves nothing about the vendor
call, nothing about whether the `curl` still returns the shape the `jq` expects, nothing about
whether the vendor's API still exists. **The page renders in exactly the same neutral register as
`manual`**, with the same icon and the same colour token, because with respect to "does this
recipe work" it is exactly as unverified.

That is a deliberate design decision and it has a test behind it
([`internal/render/render_test.go`](internal/render/render_test.go)). The reasoning: wording is
not a control. A reader skimming a page sees a badge's shape and colour and has finished forming
a belief before reading the sentence underneath it. If `flags-only` had a colour of its own —
amber, say — it would read as "partly verified", and "partly verified" is not a thing this tier
can claim. So the tier is denied any positive affordance and the difference is carried entirely
by the sentence, which is where a claim about *what* was checked belongs.

**Why not just call both of them "unverified", then?** Because "the flags are right and the
vendor steps are unchecked" and "none of this is checked" are different claims, and collapsing
them would be its own dishonesty — the documentation equivalent of reporting a point estimate
without saying which population it came from.

**Expiry.** 180 days on the hand-check date. Refreshed only by an actual `vendor-smoke` run, so
the date is evidence of a run rather than of a memory.

---

## • `manual`

**What CI did.** Nothing.

**What that licenses you to believe.** That a named human walked through this page on the date
shown. That is all, and vendor consoles change.

**Expiry.** 180 days.

---

## The tiers, side by side

| | `executed` | `flags-only` | `manual` |
|---|---|---|---|
| Icon | ✔ | • | • |
| Colour token | `kno-verify-verified` | `kno-verify-neutral` | `kno-verify-neutral` |
| Says "Verified" | yes | no | no |
| Commands run | yes | no | no |
| Command shapes checked | yes | yes | no |
| Vendor call exercised | n/a — there isn't one | no | no |
| Date written by | CI | CI (build) + a human (hand-check) | a human |
| Expires after | 30 days | 180 days | 180 days |

## Where the honesty runs out

Stated rather than glossed:

- **Vendor recipes are verified by a human, on a cadence nothing enforces.** The 180-day banner
  reports staleness; it does not prevent it. The alternative — nightly runs against eight
  third-party SaaS products with long-lived credentials in a public repository's CI — would spend
  money unattended and redden on every vendor's outage, training everyone to ignore the signal.
  We took the honest ceiling over the impressive-looking one.
- **A reader who ignores both the colour and the text is unreachable.** No mechanism a document
  has can fix that. What we can do is make sure the colour never says more than the text does,
  and that is what the renderer test enforces.
- **A green tick is a claim about a machine, not about a reader's situation.** The scenario runs
  on `fake:` with synthetic data. Your pool is messier, your intervals are wider, and your score
  is not 1.000.
