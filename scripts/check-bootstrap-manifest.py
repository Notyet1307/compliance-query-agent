#!/usr/bin/env python3
"""Verify bootstrap file hashes. Does not imply test/environment qualification."""
import hashlib, json
from pathlib import Path
root=Path(__file__).resolve().parents[1]
p=root/'evidence/bootstrap/source-manifest.json'
manifest=json.loads(p.read_text())
failures=[]
for item in manifest['files']:
    target=(root/item['path']).resolve()
    if not target.is_relative_to(root) or not target.is_file():
        failures.append(item['path']); continue
    if hashlib.sha256(target.read_bytes()).hexdigest()!=item['sha256']:
        failures.append(item['path'])
print(json.dumps({'status':'PASS' if not failures else 'FAIL','files_checked':len(manifest['files']),'mismatches':failures},ensure_ascii=False,indent=2))
raise SystemExit(1 if failures else 0)
