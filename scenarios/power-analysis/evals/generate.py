#!/usr/bin/env python3
"""Write this scenario's Cases.

    python3 scenarios/power-analysis/evals/generate.py > scenarios/power-analysis/evals/cases.jsonl

`cases.jsonl` is committed; this file is committed beside it so the data's
provenance is a program rather than a claim. Re-running it must reproduce the
committed file byte for byte, and a `make check` that disagrees means one of
the two was edited without the other.

WHY THIS SET IS GENERATED AND THE OTHER SCENARIOS' ARE HAND-WRITTEN

Every other scenario here has a dozen Cases somebody wrote one at a time, and
the words matter: `support-refunds` exists partly to hold the exact refund
sentence that once drifted between two files. This scenario needs 160 Cases and
the words do not matter at all, because nothing it demonstrates depends on
them. `fake:` answers every Case with what the Case expects, so every score is
1.000 whatever the text says. What the numbers here depend on is the COUNT and
the TAG — how many Cases there are, and how they divide into behaviors.

Writing 160 Cases by hand would therefore have bought nothing but the
appearance of effort, and would have made the set harder to change: the whole
point of the scenario is that n moves, and n moving must not mean an afternoon
of typing.

The four tags are interleaved, one behavior after another, so `head -n 12`
takes three of each rather than twelve of one. run.sh takes its smaller eval
sets that way, which is what makes them honest subsets of the same set rather
than three separately-written files that could disagree.
"""

# name, grain, window, dialect, cadence, owning group.
MARTS = [
    ("fct_sessions", "session", "28 days", "Trino", "hourly at :20", "analytics"),
    ("dim_account", "account", "90 days", "Trino", "daily at 04:10", "analytics"),
    ("fct_orders", "order", "30 days", "Trino", "hourly at :05", "commerce"),
    ("agg_daily_active", "active account", "1 day", "Trino", "daily at 05:00", "analytics"),
    ("dim_plan", "plan", "365 days", "Trino", "daily at 03:40", "analytics"),
    ("fct_payments", "payment", "30 days", "Trino", "hourly at :35", "finance"),
    ("agg_weekly_retention", "retained account", "7 days", "Trino", "weekly on Monday 06:00", "analytics"),
    ("dim_region", "region", "365 days", "Trino", "daily at 03:15", "analytics"),
    ("fct_events", "event", "7 days", "Trino", "every 15 minutes", "analytics"),
    ("agg_arr", "ARR dollar", "30 days", "Trino", "daily at 06:30", "finance"),
    ("dim_device", "device", "180 days", "Trino", "daily at 03:20", "analytics"),
    ("fct_refunds", "refund", "30 days", "Trino", "hourly at :45", "finance"),
    ("agg_funnel", "funnel step", "14 days", "Trino", "daily at 05:30", "growth"),
    ("dim_channel", "channel", "365 days", "Trino", "daily at 03:25", "growth"),
    ("fct_tickets", "support ticket", "30 days", "Trino", "hourly at :50", "support"),
    ("agg_cohort", "cohort member", "28 days", "Trino", "weekly on Monday 07:00", "analytics"),
    ("dim_currency", "currency", "365 days", "Trino", "daily at 03:05", "finance"),
    ("fct_invoices", "invoice", "90 days", "Trino", "hourly at :55", "finance"),
    ("agg_nps", "NPS response", "90 days", "Trino", "daily at 07:15", "support"),
    ("dim_team", "team", "365 days", "Trino", "daily at 03:35", "analytics"),
    ("fct_logins", "login", "7 days", "Trino", "every 15 minutes", "security"),
    ("agg_mrr", "MRR dollar", "30 days", "Trino", "daily at 06:45", "finance"),
    ("dim_feature", "feature flag", "180 days", "Trino", "daily at 03:50", "growth"),
    ("fct_trials", "trial", "30 days", "Trino", "hourly at :25", "growth"),
    ("agg_churn", "churned account", "30 days", "Trino", "daily at 07:30", "analytics"),
    ("dim_industry", "industry", "365 days", "Trino", "daily at 03:45", "analytics"),
    ("fct_signups", "signup", "14 days", "Trino", "hourly at :15", "growth"),
    ("agg_ltv", "LTV dollar", "365 days", "Trino", "weekly on Monday 08:00", "finance"),
    ("dim_source", "source", "365 days", "Trino", "daily at 03:55", "growth"),
    ("fct_upgrades", "upgrade", "30 days", "Trino", "hourly at :40", "growth"),
    ("dim_org", "organization", "365 days", "Trino", "daily at 03:30", "analytics"),
    ("fct_exports", "export", "30 days", "Trino", "hourly at :10", "analytics"),
    ("agg_dau_mau", "DAU/MAU ratio", "28 days", "Trino", "daily at 05:15", "analytics"),
    ("dim_locale", "locale", "365 days", "Trino", "daily at 03:10", "analytics"),
    ("fct_webhooks", "webhook delivery", "7 days", "Trino", "every 15 minutes", "platform"),
    ("agg_gross_margin", "margin dollar", "30 days", "Trino", "daily at 06:15", "finance"),
    ("dim_pricebook", "price book entry", "365 days", "Trino", "daily at 03:00", "finance"),
    ("fct_disputes", "dispute", "90 days", "Trino", "hourly at :30", "finance"),
    ("agg_activation", "activated account", "14 days", "Trino", "daily at 05:45", "growth"),
    ("dim_segment", "segment", "180 days", "Trino", "daily at 04:00", "growth"),
]


def rows(mart):
    name, grain, window, dialect, cadence, group = mart
    return [
        ("metric", "metric-definition",
         "What counts as one row in {}?".format(name),
         "One {} inside the trailing {}.".format(grain, window)),
        ("dialect", "sql-dialect",
         "Which SQL dialect do I query {} with?".format(name),
         "{}. Use date_diff, not DATEDIFF.".format(dialect)),
        ("fresh", "freshness",
         "How often does {} land?".format(name),
         "It lands {} UTC.".format(cadence)),
        ("access", "access-policy",
         "Who can read {}?".format(name),
         "The {} group; PII columns need an approved grant.".format(group)),
    ]


def main():
    import json

    for i, mart in enumerate(MARTS, start=1):
        for prefix, tag, question, answer in rows(mart):
            record = {
                "id": "{}-{:03d}".format(prefix, i),
                "input": question,
                "expected": answer,
                "tags": [tag],
            }
            print(json.dumps(record, separators=(",", ":")))


if __name__ == "__main__":
    main()
