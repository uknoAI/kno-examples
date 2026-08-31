#!/bin/sh
set -eu
stages="doctor"
:
# >>> doctor
kno doctor --json
# <<< doctor
