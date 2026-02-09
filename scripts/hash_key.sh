#!/usr/bin/env bash
set -euo pipefail
if [ $# -lt 1 ]; then
  echo "usage: $0 <plain_key>" >&2
  exit 1
fi
python3 - <<PY
import hashlib
print(hashlib.sha256(b"$1").hexdigest())
PY
