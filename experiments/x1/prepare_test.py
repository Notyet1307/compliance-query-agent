#!/usr/bin/env python3
"""Offline check: correct trial data and refusal to overwrite existing receipts."""
from pathlib import Path
import hashlib
import json
import shutil
import subprocess
import sys
import tempfile

root = Path(__file__).resolve().parents[2]
with tempfile.TemporaryDirectory(prefix='cqa-prepare-') as directory:
    checkout = Path(directory)
    for relative in ('experiments/x1', 'protocol', 'testdata', 'configs', 'examples'):
        shutil.copytree(root / relative, checkout / relative)
    command = [sys.executable, str(checkout / 'experiments/x1/prepare.py')]
    prepared = subprocess.run(command + ['--trial', 's2-01'], env={}, capture_output=True, text=True, check=True)
    output = checkout / json.loads(prepared.stdout)['prepared']
    corpus = json.loads((output / 'service/data/demo-corpus.json').read_text())
    assert corpus == json.loads((checkout / 'testdata/s1-test-document.json').read_text())
    request = json.loads((output / 'inputs/requests/normal.json').read_text())
    original = json.loads((checkout / 'examples/s1-test.json').read_text())
    assert request.pop('requestId') != original.pop('requestId')
    assert request == original
    assert not json.loads((output / 'inputs/config.json').read_text())['networkApproved']
    assert (output / 'receipts').stat().st_mode & 0o777 == 0o700
    sentinel = output / 'receipts/previous.json'
    sentinel.write_text('{"state":"blocked"}')
    before = {str(p): hashlib.sha256(p.read_bytes()).hexdigest() for p in output.rglob('*') if p.is_file()}
    repeat = subprocess.run(command + ['--trial', 's2-01'], env={}, capture_output=True)
    assert repeat.returncode != 0
    assert before == {str(p): hashlib.sha256(p.read_bytes()).hexdigest() for p in output.rglob('*') if p.is_file()}
    legacy = subprocess.run(command, env={}, capture_output=True, text=True, check=True)
    legacy_output = checkout / json.loads(legacy.stdout)['prepared']
    assert json.loads((legacy_output / 'data/demo-corpus.json').read_text()) == json.loads((checkout / 'testdata/demo-corpus.json').read_text())
print(json.dumps({'status': 'PASS', 'checks': ['single_document', 'same_question_new_identity', 'network_disabled', 'private_receipts', 'existing_trial_preserved', 'x1_data_unchanged']}))
