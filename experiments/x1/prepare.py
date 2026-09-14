#!/usr/bin/env python3
"""Prepare fixed synthetic trial inputs; no network, install or daemon access."""
import argparse
from pathlib import Path
import hashlib
import json
import shutil

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--trial', choices=('x1', 's2-01'), default='x1')
trial = parser.parse_args().trial
root = Path(__file__).resolve().parents[2]
output = root / ('.local/x1-runtime/build/service' if trial == 'x1' else '.local/s2-01/prepared')
if output.exists() or output.is_symlink():
    raise SystemExit('Refusing to overwrite an existing trial directory')
output.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
if trial == 's2-01':
    output.mkdir(mode=0o700)
service = output if trial == 'x1' else output / 'service'
shutil.copytree(Path(__file__).parent / 'service', service)
corpus = 'testdata/demo-corpus.json' if trial == 'x1' else 'testdata/s1-test-document.json'
for source, target in (
    ('protocol/compliance.proto', 'proto/compliance.proto'),
    (corpus, 'data/demo-corpus.json'),
):
    destination = service / target
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(root / source, destination)
(service / 'bin/compliance-kb.js').chmod(0o755)
if trial == 's2-01':
    inputs = output / 'inputs'
    requests = inputs / 'requests'
    requests.mkdir(parents=True)
    shutil.copyfile(root / 'configs/s2-native-test.json', inputs / 'config.json')
    for name, source in (
        ('normal', 'examples/s1-test.json'),
        ('no-evidence', 'examples/s1-model-insufficient.json'),
    ):
        request = json.loads((root / source).read_text())
        request['requestId'] = f's2-01-{name}'
        (requests / f'{name}.json').write_text(json.dumps(request, ensure_ascii=False, indent=2) + '\n')
    (output / 'receipts').mkdir(mode=0o700)
    observation = output / 'observation'
    observation.mkdir(mode=0o700)
    (observation / 'control.json').write_text('{"delayMs":0}\n')
manifest = [
    {'path': str(path.relative_to(output)), 'sha256': hashlib.sha256(path.read_bytes()).hexdigest()}
    for path in sorted(output.rglob('*')) if path.is_file()
]
print(json.dumps({'trial': trial, 'prepared': str(output.relative_to(root)), 'files': manifest,
                  'dependencies_installed': False, 'runtime_started': False}, indent=2))
