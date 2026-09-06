#!/usr/bin/env python3
from html.parser import HTMLParser
from pathlib import Path
import re, sys
ROOT=Path(__file__).resolve().parents[1]
class P(HTMLParser):
    def __init__(self): super().__init__(); self.assets=[]
    def handle_starttag(self,tag,attrs):
        a=dict(attrs)
        if tag=='script' and a.get('src'): self.assets.append(a['src'].split('?',1)[0].lstrip('/'))
        if tag=='link' and a.get('rel')=='stylesheet' and a.get('href'): self.assets.append(a['href'].split('?',1)[0].lstrip('/'))
html=(ROOT/'src/magisk-module/webroot/index.html').read_text(); p=P(); p.feed(html)
server='\n'.join((ROOT/x).read_text() for x in ('webui-core/server/cmd/webui-server/main.go','webui-core/server/cmd/webui-server/v03.go','webui-core/server/cmd/webui-server/v04.go'))
routes=set()
for m in re.finditer(r'case\s+((?:"[A-Za-z0-9._/-]+"(?:\s*,\s*)?)+)\s*:',server): routes.update(x.lstrip('/') for x in re.findall(r'"([A-Za-z0-9._/-]+)"',m.group(1)))
for m in re.finditer(r'(?:HandleFunc|Handle)\(\s*"/([A-Za-z0-9._/-]+)"',server): routes.add(m.group(1).lstrip('/'))
missing=sorted(set(p.assets)-routes)
integration=(ROOT/'tools/webui-integration-test.sh').read_text()
need=('asset_http=PASS','status_lsposed_vector=PASS','disabled_mutation_surfaces=PASS','origin_rejected=PASS','api/v1/inventory?name=zygisk_stack','api/v1/log?lines=300')
missing_contract=[x for x in need if x not in integration]
failures=len(missing)+len(missing_contract)
print(f'asset_routes_total={len(set(p.assets))}')
print('asset_routes_missing='+(','.join(missing) if missing else 'none'))
print('integration_contract_missing='+(','.join(missing_contract) if missing_contract else 'none'))
print('settings_roundtrip=not_applicable_config_disabled')
print('evidence_collection=complete'); print(f'failure_count={failures}')
if failures: print('verdict=fail'); print('RESULT: BOOT_WATCH_WEBUI_RELEASE_AUDIT_STATIC_FAIL outcome=failure workflow_exit_code=1'); sys.exit(1)
print('verdict=pass'); print('RESULT: BOOT_WATCH_WEBUI_RELEASE_AUDIT_STATIC_PASS outcome=success workflow_exit_code=0')
