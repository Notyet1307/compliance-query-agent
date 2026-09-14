#!/bin/sh
# Generates local evidence; never installs dependencies or calls live services.
set -eu
cd "$(dirname "$0")/.."
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
OUT=.local/verification
mkdir -p "$OUT" bin
go version > "$OUT/go-version.txt"
gofmt -l cmd internal > "$OUT/gofmt.txt"
if [ -s "$OUT/gofmt.txt" ]; then cat "$OUT/gofmt.txt"; exit 1; fi
go test -json -count=1 ./... > "$OUT/go-tests.jsonl"
go vet ./... > "$OUT/vet.txt" 2>&1
go build -trimpath -o bin/compliance-agent ./cmd/compliance-agent > "$OUT/build.txt" 2>&1
python3 scripts/repo-check.py > "$OUT/repo-check.json"
python3 experiments/x1/prepare_test.py > "$OUT/trial-preparation.json"
python3 scripts/smoke.py --binary bin/compliance-agent > "$OUT/smoke.json"
python3 - "$OUT" <<'PYSUMMARY'
import json, pathlib, sys
p=pathlib.Path(sys.argv[1])
events=[json.loads(s) for s in (p/'go-tests.jsonl').read_text().splitlines()]
passed=[e['Test'] for e in events if e['Action']=='pass' and 'Test' in e]
failed=[e for e in events if e['Action']=='fail']
result={'status':'PASS' if not failed else 'FAIL','go_test_roots':len([n for n in passed if '/' not in n]),'go_test_events_including_subtests':len(passed),'smoke':json.loads((p/'smoke.json').read_text()),'trial_preparation':json.loads((p/'trial-preparation.json').read_text()),'evidence_directory':str(p)}
print(json.dumps(result,ensure_ascii=False,indent=2))
PYSUMMARY
