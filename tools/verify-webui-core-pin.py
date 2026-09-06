#!/usr/bin/env python3
from pathlib import Path
import hashlib, sys
ROOT=Path(__file__).resolve().parents[1]
failures=[]
for manifest in sorted((ROOT/"webui-core").glob("core-source-blobs-*.tsv")):
    for raw in manifest.read_text(encoding='utf-8').splitlines():
        if not raw.strip(): continue
        target, source, expected = raw.split('\t')
        path=ROOT/target
        if not path.is_file(): failures.append(f'missing:{target}'); continue
        data=path.read_bytes()
        actual=hashlib.sha1(b'blob '+str(len(data)).encode()+b'\0'+data).hexdigest()
        if actual != expected: failures.append(f'blob:{target}:{actual}:{expected}')
if failures: print('FAIL webui_core_pin='+';'.join(failures)); sys.exit(1)
print('core_version=0.6.3')
print('core_commit=6791a05be79f162979c76a286f7cdbdd9ce1cc6b')
print('RESULT: BOOT_WATCH_WEBUI_CORE_PIN_PASS')
