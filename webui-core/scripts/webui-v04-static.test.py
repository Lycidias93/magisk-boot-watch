#!/usr/bin/env python3
from pathlib import Path

root = Path(__file__).resolve().parents[1]

def text(path):
    return (root / path).read_text(encoding="utf-8")

version = tuple(int(part) for part in text("CORE_VERSION").strip().split("."))
assert len(version) == 3 and version >= (0, 4, 0)
main = text("server/cmd/webui-server/main.go")
v04 = text("server/cmd/webui-server/v04.go")
index = text("module/webroot/index.html")
js = text("module/webroot/v04.js")
manifest = text("core/manifest.txt")

assert "statusControlTimeout" in main
assert "registerV04Handlers(mux, app)" in main
assert 'root-module-webui.extensions.v2' in v04
assert 'job-run-file' in v04
assert 'inventory-operation' in v04
assert 'loadV04ReferenceValues' in v04
assert '<script src="/v04.js"></script>' in index
assert 'document.hidden' in js
assert 'visibilitychange' in js
assert '/api/v1/v04/jobs' in js
assert '/api/v1/v04/inventory-operation' in js
assert 'server/cmd/webui-server/v04.go' in manifest
assert 'module/webroot/v04.js' in manifest
assert 'eval(' not in js
assert 'innerHTML' not in js
print("RESULT: WEBUI_CORE_V04_STATIC_CONTRACT_PASS")
