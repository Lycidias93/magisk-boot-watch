#!/usr/bin/env python3
from html.parser import HTMLParser
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
failures = []


class Parser(HTMLParser):
    def __init__(self):
        super().__init__()
        self.lang = ""
        self.ids = set()
        self.inline_scripts = 0
        self.inline_styles = 0
        self.scripts = []
        self.links = []
        self.features = set()

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if tag == "html":
            self.lang = attrs.get("lang", "")
        if "id" in attrs:
            self.ids.add(attrs["id"])
        if "data-feature" in attrs:
            self.features.add(attrs["data-feature"])
        if tag == "script":
            source = attrs.get("src")
            if source:
                self.scripts.append(source)
            else:
                self.inline_scripts += 1
        if tag == "style":
            self.inline_styles += 1
        if tag == "link":
            self.links.append(attrs.get("href", ""))


html = (ROOT / "module/webroot/index.html").read_text(encoding="utf-8")
parser = Parser()
parser.feed(html)

expected_ids = {
    "moduleName", "moduleVersion", "connectionBadge", "notice", "statusCards",
    "statusDetails", "configForm", "dirtyBadge", "saveConfigButton",
    "actionStateSummary", "actionCards", "jobLaunchers", "jobList",
    "inventoryLaunchers", "inventoryRefreshButton", "inventoryMeta",
    "inventoryOutput", "logFilter", "logOutput", "safetyCards",
}
missing = expected_ids - parser.ids
if missing:
    failures.append(f"missing_ids={sorted(missing)}")
if parser.lang != "en":
    failures.append(f"lang={parser.lang}")
if parser.inline_scripts:
    failures.append("inline_script")
if parser.inline_styles:
    failures.append("inline_style")
if parser.scripts != ["embedded-host-bootstrap.js", "race-guard.js", "observability.js", "mobile-input-viewport.js", "app.js", "/v03.js", "/v04.js"]:
    failures.append(f"scripts={parser.scripts}")
for stylesheet in ("app.css", "race-guard.css", "observability.css"):
    if stylesheet not in parser.links:
        failures.append(f"stylesheet_missing={stylesheet}")
if 'aria-live="polite"' not in html:
    failures.append("aria_live_missing")
if parser.features != {"config", "actions", "jobs", "inventory", "logs"}:
    failures.append(f"features={sorted(parser.features)}")

embedded = (ROOT / "module/webroot/embedded-host-bootstrap.js").read_text(encoding="utf-8")
for guard in (
    'const bridge = window.ksu;',
    'location.hostname === "mui.kernelsu.org"',
    '/^\\/data\\/adb\\/modules\\/[A-Za-z0-9._-]+$/.test(moduleDir)',
    'moduleDir.endsWith(`/${moduleId}`)',
    'bridge.exec(command, `window.${callbackName}`);',
    '--print-url',
    'WEBUI_BOOTSTRAP_URL=',
    'http:\\/\\/127\\.0\\.0\\.1:',
    'target.startsWith("/api/v1/")',
    'location.replace(match[1])',
):
    if guard not in embedded:
        failures.append(f"embedded_guard={guard}")
for forbidden in (
    "apatch", "magisk", "Android.exec", "eval(", "new Function", "innerHTML", "insertAdjacentHTML",
):
    if forbidden in embedded:
        failures.append(f"embedded_forbidden={forbidden}")

mobile_input = (ROOT / "module/webroot/mobile-input-viewport.js").read_text(encoding="utf-8")
for guard in (
    "window.visualViewport",
    "focusin",
    "scrollIntoView",
    "viewport.addEventListener('resize'",
    "viewport.addEventListener('scroll'",
):
    if guard not in mobile_input:
        failures.append(f"mobile_input_guard={guard}")

javascript = (ROOT / "module/webroot/app.js").read_text(encoding="utf-8")
for endpoint in (
    "/api/v1/capabilities", "/api/v1/status", "/api/v1/config",
    "/api/v1/action", "/api/v1/jobs", "/api/v1/inventory", "/api/v1/log",
):
    if endpoint not in javascript:
        failures.append(f"endpoint={endpoint}")
for guard in (
    'credentials: "same-origin"',
    'headers.set("X-WebUI-Request", "1")',
    'cache: "no-store"',
    'function applyFeatureVisibility()',
    'Array.isArray(status.summary)',
    'scrollIntoView({ behavior: "auto", block: "nearest", inline: "nearest" })',
    'function configuredState(definition)',
    'Configured · leave blank to preserve.',
    'function actionState()',
    'function renderActionSummary(actions, current)',
    'Preview only',
    'Preview current setting',
    'Reapply current setting',
    'state.inventoryCache',
    'inventorySequence',
    'aria-pressed',
    'function loadInventory(name, { force = false } = {})',
    'function syncRunState()',
    'definition.apply_job',
    'state.actionJobSyncers',
    'started in Jobs.',
):
    if guard not in javascript:
        failures.append(f"guard={guard}")

css = (ROOT / "module/webroot/app.css").read_text(encoding="utf-8")
for guard in (
    "scrollbar-width: none",
    ".action-card.active-state",
    ".inventory-launcher.active",
    ".inventory-table td::before",
    "overflow-wrap: anywhere",
):
    if guard not in css:
        failures.append(f"css_guard={guard}")

race_guard = (ROOT / "module/webroot/race-guard.js").read_text(encoding="utf-8")
for guard in (
    'webuiStatusReady',
    'webuiMutationBusy',
    'another operation is still completing',
    'sequence !== logSequence',
    'sequence !== statusSequence',
    'path === "/api/v1/action"',
    'path === "/api/v1/config"',
    'path === "/api/v1/jobs"',
):
    if guard not in race_guard:
        failures.append(f"race_guard={guard}")

observability = (ROOT / "module/webroot/observability.js").read_text(encoding="utf-8")
for guard in (
    'const CORE_VERSION = "0.6.3"',
    'const MAX_OPERATIONS = 200',
    'window.fetch = async function observedFetch',
    'sanitizeStatus',
    'sanitizeJobs',
    'SENSITIVE_KEY',
    'beforeunload',
    'suppressBeforeUnload',
    'window.location.reload()',
):
    if guard not in observability:
        failures.append(f"observability_guard={guard}")

v03 = (ROOT / "module/webroot/v03.js").read_text(encoding="utf-8")
for endpoint in (
    "/api/v1/v03/capabilities", "/api/v1/v03/collection",
    "/api/v1/v03/import", "/api/v1/v03/import/apply", "/api/v1/v03/export",
):
    if endpoint not in v03:
        failures.append(f"v03_endpoint={endpoint}")
for guard in (
    'headers.set("X-WebUI-Request", "1")',
    'credentials: "same-origin"',
    'cache: "no-store"',
    'Preview changes',
    'Apply reviewed changes',
    'Validate & preview',
    'Apply reviewed import',
    'definition.max_bytes',
    'function resultSummary(value)',
    'function syncApplyState()',
    'function syncImportApply()',
    'New record added.',
    'recordCount',
    'aria-live',
):
    if guard not in v03:
        failures.append(f"v03_guard={guard}")

v04 = (ROOT / "module/webroot/v04.js").read_text(encoding="utf-8")
for endpoint in (
    "/api/v1/v04/capabilities", "/api/v1/v04/reference",
    "/api/v1/v04/jobs", "/api/v1/v04/inventory-operation",
):
    if endpoint not in v04:
        failures.append(f"v04_endpoint={endpoint}")
for guard in (
    'headers.set("X-WebUI-Request", "1")',
    'credentials: "same-origin"',
    'cache: "no-store"',
    'document.hidden',
    'visibilitychange',
    'dedupe: reused active job',
    'phases:',
):
    if guard not in v04:
        failures.append(f"v04_guard={guard}")

for label, source in (("app", javascript), ("race_guard", race_guard), ("observability", observability), ("v03", v03), ("v04", v04), ("mobile_input", mobile_input)):
    for forbidden in (
        "ksu.exec", "apatch.exec", "magisk.exec", "webui.exec", "Android.exec",
        "eval(", "new Function", "innerHTML =", "insertAdjacentHTML",
    ):
        if forbidden in source:
            failures.append(f"{label}_forbidden={forbidden}")

action = (ROOT / "module/action.sh").read_text(encoding="utf-8")
for required in ("-token-file", "/data/local/tmp/", "/bootstrap?token=", "-self-test", "--print-url", "WEBUI_BOOTSTRAP_URL="):
    if required not in action:
        failures.append(f"action_contract={required}")
if ' -token "$TOKEN"' in action or " -token " in action:
    failures.append("token_in_argv")

control = (ROOT / "module/bin/module-control").read_text(encoding="utf-8")
for operation in ("capabilities)", "config-apply)", "action-file)", "job-run)", "inventory)"):
    if operation not in control:
        failures.append(f"control_operation={operation}")

server_v03 = (ROOT / "server/cmd/webui-server/v03.go").read_text(encoding="utf-8")
for required in (
    'v03CapabilitySchema = "root-module-webui.extensions.v1"',
    'maxV03UploadBytes',
    'requireV03JSONMutation',
    'requireV03UploadMutation',
    'matching unexpired preview required',
    'file outside private upload directory',
    'credential_material',
):
    if required not in server_v03:
        failures.append(f"v03_server_guard={required}")

server_v04 = (ROOT / "server/cmd/webui-server/v04.go").read_text(encoding="utf-8")
for required in (
    'v04CapabilitySchema = "root-module-webui.extensions.v2"',
    'job-run-file',
    'dedupe_keys',
    'loadV04ReferenceValues',
    'resolveInventoryItem',
    'inventory-operation',
):
    if required not in server_v04:
        failures.append(f"v04_server_guard={required}")

if failures:
    print("FAIL: webui_contract=" + ",".join(failures))
    sys.exit(1)
print("RESULT: WEBUI_CONTRACT_TEST_PASS")