#!/usr/bin/env python3
"""Real CLI/process/loopback API checks with fictional data. No external services."""
import argparse, copy, hashlib, json, os, secrets, socket, subprocess, tempfile, time, urllib.error, urllib.request
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
parser=argparse.ArgumentParser()
parser.add_argument('--binary',default='bin/compliance-agent')
args=parser.parse_args()
binary=(ROOT/args.binary).resolve()
assert binary.is_file(), 'build binary first'
checks=[]
def passed(name): checks.append({'name':name,'status':'PASS'})
def run(req, config, success=True):
    c=subprocess.run([str(binary),'query','--config',str(config),'--input','-'],input=json.dumps(req),text=True,capture_output=True,timeout=12,cwd=ROOT)
    assert (c.returncode==0)==success, (c.returncode,c.stdout,c.stderr)
    return json.loads(c.stdout if success else c.stderr)
with tempfile.TemporaryDirectory(prefix='cqa-smoke-') as td:
    tmp=Path(td)
    cfg=json.loads((ROOT/'configs/demo.json').read_text())
    cfg['knowledge']['corpusPath']=str(ROOT/'testdata/demo-corpus.json')
    cfg['storeDir']=str(tmp/'receipts')
    cp=tmp/'config.json'; cp.write_text(json.dumps(cfg))
    statuses={'mlps':'REFERENCE_ONLY','ciip':'REFERENCE_ONLY','policy':'REFERENCE_ONLY','industry':'REFERENCE_ONLY','crypto':'REFERENCE_ONLY','insufficient':'INSUFFICIENT_EVIDENCE','stale':'NEEDS_REVIEW','conflict':'NEEDS_REVIEW','future':'INSUFFICIENT_EVIDENCE'}
    for name,status in statuses.items():
        req=json.loads((ROOT/f'examples/{name}.json').read_text())
        out=run(req,cp)
        assert out['status']==status,(name,out)
        assert out['dataMode']=='synthetic_demo' and out['humanReviewRequired'] and not out['entailmentVerified']
        for cite in out['citations']:
            assert cite['synthetic'] and cite['sha256']==hashlib.sha256(cite['quote'].encode()).hexdigest()
        passed('cli_'+name)
    req=json.loads((ROOT/'examples/mlps.json').read_text())
    one=run(req,cp); two=run(req,cp)
    assert one==two
    passed('cli_replay_across_process_restarts')
    changed=copy.deepcopy(req); changed['question']+='以及责任人'
    assert run(changed,cp,False)['error']=='IDEMPOTENCY_CONFLICT'
    passed('cli_idempotency_conflict')
    invalid=copy.deepcopy(req); invalid['requestId']='../escape'
    assert run(invalid,cp,False)['error']=='INVALID_REQUEST'
    passed('cli_invalid_request_id')
    sock=socket.socket(); sock.bind(('127.0.0.1',0)); port=sock.getsockname()[1]; sock.close()
    cfg['server']['listen']=f'127.0.0.1:{port}'; cp.write_text(json.dumps(cfg))
    token=secrets.token_urlsafe(40)
    env={**os.environ,'CQA_API_TOKEN':token}
    proc=subprocess.Popen([str(binary),'serve','--config',str(cp)],cwd=ROOT,env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
    opener=urllib.request.build_opener(urllib.request.ProxyHandler({}))
    base=f'http://127.0.0.1:{port}'
    def http(path,data=None,auth=True,ctype='application/json'):
        headers={'Content-Type':ctype}
        if auth: headers['Authorization']='Bearer '+token
        payload=None if data is None else (data if isinstance(data,bytes) else json.dumps(data).encode())
        r=urllib.request.Request(base+path,data=payload,headers=headers)
        try:
            with opener.open(r,timeout=5) as v: return v.status,dict(v.headers),json.loads(v.read())
        except urllib.error.HTTPError as e:
            return e.code,dict(e.headers),json.loads(e.read())
    try:
        for _ in range(80):
            if proc.poll() is not None: raise RuntimeError('server failed to start')
            try:
                status,_,body=http('/healthz',auth=False)
                break
            except (urllib.error.URLError,ConnectionError): time.sleep(.05)
        else: raise RuntimeError('server unavailable')
        assert status==200 and body['meaning']=='process_only_not_external_integration'
        passed('http_health_process_only')
        req['requestId']='http-smoke-1'
        assert http('/v1/query',req,False)[0]==401
        passed('http_auth_required')
        status,_,result=http('/v1/query',req)
        assert status==200 and result['status']=='REFERENCE_ONLY'
        passed('http_real_query')
        status,headers,result2=http('/v1/query',req)
        assert status==200 and result==result2 and headers.get('X-Cqa-Replayed')=='true',headers
        passed('http_replay')
        assert http('/v1/query',req,ctype='application/jsonfoo')[0]==415
        passed('http_reject_invalid_media_type')
        assert http('/v1/query',b'{"question":"one","question":"two"}')[0]==400
        passed('http_reject_duplicate_json')
        assert http('/v1/query',b'x'*65537)[0]==413
        passed('http_size_limit')
    finally:
        proc.terminate()
        stdout,stderr=proc.communicate(timeout=10)
    assert proc.returncode==0,(proc.returncode,stderr)
    assert token not in stdout+stderr
    passed('http_clean_shutdown_no_credential_reflection')
print(json.dumps({'status':'PASS','checks':checks,'check_count':len(checks),'data':'synthetic_only','network':'literal_loopback_only','live_llm':'NOT_RUN','live_octobus':'NOT_RUN','agent_compose':'NOT_RUN','accord':'NOT_RUN'},ensure_ascii=False,indent=2))
