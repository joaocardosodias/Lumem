#!/bin/sh

set -eu

report="$(mktemp)"
trap 'rm -f "$report"' EXIT

set +e
govulncheck ./... >"$report" 2>&1
status=$?
set -e

cat "$report"

if [ "$status" -eq 0 ]; then
	exit 0
fi

# GO-2026-6452 has no fixed version in the vulnerability database yet. The
# pinned Excelize revision includes upstream commit 93f0b3ca, which validates
# negative shared-string indexes. Keep this exception tied to that exact
# revision so a dependency change or any additional finding fails the job.
finding_ids="$(sed -n 's/^Vulnerability #[0-9][0-9]*: //p' "$report")"
excelize_version="$(go list -m -f '{{.Version}}' github.com/xuri/excelize/v2)"
patched_version="v2.11.1-0.20260918021423-0434413565bf"

if [ "$finding_ids" = "GO-2026-6452" ] && [ "$excelize_version" = "$patched_version" ]; then
	echo "GO-2026-6452 is mitigated by the pinned upstream revision (includes commit 93f0b3ca); accepting this known database lag."
	exit 0
fi

exit "$status"
