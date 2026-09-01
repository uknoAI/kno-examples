#!/bin/sh
set -eu
stages="doctor reads-a-file reads-a-store"

# `reads-a-file` needs nothing an earlier stage wrote; `reads-a-store` does.
# shellcheck disable=SC2034 # read by `verify lint`, not by this script
independent_stages="reads-a-file"
:
# >>> doctor
kno doctor --json
# <<< doctor
# >>> reads-a-file
kno eval inspect --evals cases.jsonl --holdout-frac 0.2
# <<< reads-a-file
# >>> reads-a-store
kno report --value-run-id toy-value --db kno.db
# <<< reads-a-store
