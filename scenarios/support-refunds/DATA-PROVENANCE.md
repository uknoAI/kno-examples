# Data provenance — `support-refunds`

**Author:** @devarispbrown, for `uknoAI/kno-examples`.

**Assertion:** every line of `evals/cases.jsonl` and `pool/pool.jsonl` is **synthetic**. It was
written by hand to exercise the loop. No line is derived from a real support ticket, a real
help-center article, a real customer, a real company, or any production system. No line is
copied from a third party's corpus.

**Why this file is required.** Kno treats stored traces as customer data: a Case's `input` is an
end-user's words and a Case's `expected` is what somebody was told. A public repository of
"example support tickets" is precisely where that rule gets broken by accident, and the accident
is unrecoverable — a published conversation cannot be unpublished. So no scenario merges without
a provenance file naming an author and asserting synthesis, and the assertion is reviewed by a
human. This is a human gate and is stated as one; no linter can tell a plausible invented ticket
from a real one.

**License.** Apache-2.0, the same as the rest of this repository and the same as `uknoAI/kno`.
Synthesis is what makes that license grantable: we can only license data we have the right to
license.

**Relationship to `uknoAI/kno`.** These files are byte-identical to `cli/demodata/cases.jsonl`
and `cli/demodata/pool.jsonl` in `uknoAI/kno`, which `kno demo` embeds, and to the JSONL typed
in `tapes/quickstart.tape`. The duplication is deliberate and detector-guarded; see this
scenario's `README.md`.

**Secrets.** No line contains a credential, a token, an internal hostname, or an email address.
`gitleaks` runs over this repository's working tree and its full history in PR CI, because a
scenario repository full of `export VENDOR_API_KEY=` lines is the single most likely place in
the organization for a real key to be pasted.
