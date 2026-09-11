#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
if [ "$#" -ne 0 ]; then
  printf '%s\n' 'This demo accepts no user input or command arguments.' >&2
  exit 2
fi
command -v agent-compose >/dev/null 2>&1 || { printf '%s\n' 'BLOCKED: agent-compose is not installed.' >&2; exit 2; }
# Constant command only. Never append user questions or arbitrary paths.
agent-compose -f agent-compose.yml run compliance-query --command \
  '/opt/cqa/compliance-agent query --config /opt/cqa/configs/guest-demo.json --input /opt/cqa/examples/mlps.json'
