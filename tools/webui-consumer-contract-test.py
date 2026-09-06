#!/usr/bin/env python3
from html.parser import HTMLParser
from pathlib import Path
import sys
ROOT=Path(__file__).resolve().parents[1]
WEB=ROOT/'src/magisk-module/webroot'
failures=[]
class P(HTMLParser):
    def __init__(self): super().__init__(); self.assets=[]
    def handle_starttag(self,tag,attrs):
        a=dict(attrs)
        if tag=='script' and a.get('src'): self.assets.append(a['src'].split('?',1)[0].lstrip('/'))
        if tag=='link' and a.get('rel')=='stylesheet' and a.get('href'): self.assets.append(a['href'].split('?',1)[0].lstrip('/'))
html=(WEB/'index.html').read_text(encoding='utf-8'); p=P(); p.feed(html)
for asset in p.assets:
    if not (WEB/asset).is_file(): failures.append('missing_asset='+asset)
for forbidden in ('cdn.','unpkg.com','jsdelivr','google-analytics','fonts.googleapis'):
    if forbidden in html: failures.append('remote='+forbidden)
control=(ROOT/'src/magisk-module/bin/module-control').read_text(encoding='utf-8')
for item in ('"features":{"config":false,"logs":true,"actions":false,"jobs":false,"inventory":true}', '"name":"zygisk_stack"', 'print_zygisk_inventory', '"read_only_adapter":true'):
    if item not in control: failures.append('control='+item)
collector=(ROOT/'src/magisk-module/boot-watch.sh').read_text(encoding='utf-8')
for item in ('collect_zygisk_stack_support','magisk --denylist ls','lspd_config_contents_collected=no','manager_private_cache_collected=no','VERSION_CODE="36"'):
    if item not in collector: failures.append('collector='+item)
for forbidden in ('/data/user_de/0/org.lsposed.manager/cache','/data/user_de/0/io.github.lsposed.manager/cache'):
    if forbidden in collector: failures.append('private_cache='+forbidden)
if failures: print('FAIL consumer_contract='+','.join(failures)); sys.exit(1)
print('RESULT: BOOT_WATCH_WEBUI_CONSUMER_CONTRACT_PASS')
