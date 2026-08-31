---
verification: flags-only
owner: "@nobody"
verified-against: kno v0.1.2
last-manual-verification: 2026-08-31
credentials: [ZENDESK_API_KEY]
---

# A vendor recipe that bills OpenAI without saying so

```bash
export ZENDESK_API_KEY=...
kno baseline --evals cases.jsonl --agent openai:gpt-4.1 --max-cost-usd 2.00 --yes
```
