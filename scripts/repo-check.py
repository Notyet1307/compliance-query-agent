#!/usr/bin/env python3
"""Check authored data and local Markdown links, not upstream compatibility."""
import json, re
from pathlib import Path
from urllib.parse import unquote
ROOT = Path(__file__).resolve().parents[1]
excluded={'.local','bin','.git','evidence'}
paths=[p for p in ROOT.rglob('*') if p.is_file() and not excluded.intersection(p.relative_to(ROOT).parts)]
json_count=0; link_count=0; broken=[]
for p in paths:
    if p.suffix=='.json':
        json.loads(p.read_text()); json_count+=1
    if p.suffix=='.md':
        for target in re.findall(r'\]\(([^)]+)\)',p.read_text()):
            if '://' in target or target.startswith('#') or target.startswith('mailto:'): continue
            target=unquote(target.split('#')[0]); link_count+=1
            if target and not (p.parent/target).exists(): broken.append({'file':str(p.relative_to(ROOT)),'target':target})
required=['README.md','AGENTS.md','CONTEXT.md','docs/product.md','docs/specs/mvp.md','docs/architecture.md','docs/development.md','docs/handoff.md','docs/agents/issue-tracker.md','docs/agents/domain.md','docs/agents/skill-usage.md','prompts/01_OMP接管合规查询智能体.prompt.md','Makefile','agent-compose.yml']
assert all((ROOT/x).is_file() for x in required),'missing required material'
assert not broken,broken
print(json.dumps({'status':'PASS','json_files_parsed':json_count,'local_markdown_links_checked':link_count,'broken_links':broken,'upstream_parser_validation':'NOT_RUN'},ensure_ascii=False,indent=2))
