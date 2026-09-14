#!/usr/bin/env python3
"""Operator-only X1 cancellation over the daemon's private Unix socket."""
import hashlib
import http.client
import json
import socket
import re
import sys

if len(sys.argv) != 2:
    raise SystemExit('usage: stop_run.py RUN_ID')
try:
    run_id = sys.argv[1]
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_.-]{0,95}', run_id):
        raise ValueError('invalid run id')
    with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as transport:
        transport.settimeout(5)
        transport.connect('/var/run/agent-compose.sock')
        connection = http.client.HTTPConnection('localhost', timeout=5)
        connection.sock = transport
        try:
            connection.request('POST', '/agentcompose.v2.RunService/StopRun',
                               json.dumps({'runId': run_id, 'reason': 'x1 operator cancellation'}),
                               {'Content-Type': 'application/json', 'Connect-Protocol-Version': '1'})
            response = connection.getresponse()
            status = response.status
            body = response.read(1024 * 1024 + 1)
        finally:
            connection.close()
    if len(body) > 1024 * 1024:
        raise ValueError('oversized response')
    payload = json.loads(body) if status == 200 else {}
    if not isinstance(payload, dict):
        raise ValueError('invalid response')
except (OSError, http.client.HTTPException, ValueError):
    raise SystemExit('X1_STOP_TRANSPORT_OR_RESPONSE_ERROR')

# Do not emit the run transcript or arbitrary control-plane error text.
print(json.dumps({'runId': run_id, 'httpStatus': status,
                  'stopRequested': payload.get('stopRequested', False),
                  'responseSha256': hashlib.sha256(body).hexdigest()}))
raise SystemExit(0 if status == 200 and isinstance(payload.get('stopRequested'), bool) else 1)
