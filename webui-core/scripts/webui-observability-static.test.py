#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
js = (ROOT / "module/webroot/observability.js").read_text()
css = (ROOT / "module/webroot/observability.css").read_text()
index = (ROOT / "module/webroot/index.html").read_text()
manifest = (ROOT / "core/manifest.txt").read_text() if (ROOT / "core/manifest.txt").exists() else ""
version = (ROOT / "CORE_VERSION").read_text().strip() if (ROOT / "CORE_VERSION").exists() else "0.6.3"

required_js = [
    'const CORE_VERSION = "0.6.3"',
    'const MAX_OPERATIONS = 200',
    'window.fetch = async function observedFetch',
    'SENSITIVE_KEY',
    'sanitizeStatus',
    'action_state: value.action_state',
    'sanitizeJobs',
    'response.clone()',
    'config.apply',
    'collectionMode',
    'return mode ? `collection.${mode}` : "collection.change";',
    'import.apply',
    'workflow.start',
    'inventory.operation',
    'beforeunload',
    'suppressBeforeUnload',
    'window.location.reload()',
    'Request bodies, shell commands and job output are not recorded here.',
    'globalThis.fetch = async function actionFeedbackFetch',
    'actionFeedbackPanel',
    'Latest action result',
    'className = "action-card hidden"',
    'output.className = "job-output"',
    'button.textContent !== "Run check"',
    'Action failed. Details are shown in Actions.',
    'completed. Output is shown in Actions.',
    'body.slice(0, 2048)',
]
for needle in required_js:
    assert needle in js, f"missing observability contract marker: {needle}"

for forbidden in [
    "innerHTML",
    "insertAdjacentHTML",
    "eval(",
    "new Function",
    "ksu.exec",
    "apatch.exec",
    "magisk.exec",
    "Android.exec",
    "localStorage",
    "sessionStorage",
]:
    assert forbidden not in js, f"forbidden observability pattern: {forbidden}"

assert 'init?.body' in js, "collection mode may inspect only the already supplied request body"
assert 'body.slice(0, 2048)' in js, "request-body inspection must stay bounded"
assert 'JSON.parse(body)' not in js, "observability must not parse or retain arbitrary request payloads"
assert 'snapshots.set' in js and 'operations.push' in js
assert 'embedded-host-bootstrap.js' in index and 'observability.css' in index and 'observability.js' in index
assert index.index('embedded-host-bootstrap.js') < index.index('race-guard.js') < index.index('observability.js') < index.index('app.js') < index.index('/v03.js') < index.index('/v04.js')
assert '.core-dirty-bar' in css and '.core-operation-entry' in css
assert '.shell {' in css and 'padding-bottom: calc(104px + env(safe-area-inset-bottom));' in css
assert 'padding-bottom: calc(190px + env(safe-area-inset-bottom));' in css
assert version == "0.6.3", f"expected CORE_VERSION 0.6.3, got {version}"
if manifest:
    assert "module/webroot/embedded-host-bootstrap.js" in manifest
    assert "module/webroot/observability.js" in manifest
    assert "module/webroot/observability.css" in manifest
    assert "scripts/webui-observability-static.test.py" in manifest

print("RESULT: WEBUI_CORE_V061_OBSERVABILITY_STATIC_PASS")
