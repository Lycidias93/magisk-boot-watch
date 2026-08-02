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

html = (ROOT / "src/magisk-module/webroot/index.html").read_text(encoding="utf-8")
parser = Parser()
parser.feed(html)

expected_ids = {
    "moduleName", "moduleVersion", "connectionBadge", "notice", "statusCards",
    "statusDetails", "configForm", "dirtyBadge", "saveConfigButton",
    "actionCards", "jobLaunchers", "jobList", "inventoryLaunchers",
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
if parser.scripts != ["app.js"]:
    failures.append(f"scripts={parser.scripts}")
if "app.css" not in parser.links:
    failures.append("app_css_missing")
if 'aria-live="polite"' not in html:
    failures.append("aria_live_missing")
if parser.features != {"config", "actions", "jobs", "inventory", "logs"}:
    failures.append(f"features={sorted(parser.features)}")

javascript = (ROOT / "src/magisk-module/webroot/app.js").read_text(encoding="utf-8")
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
):
    if guard not in javascript:
        failures.append(f"guard={guard}")
for forbidden in ("ksu.exec", "apatch.exec", "magisk.exec", "webui.exec", "Android.exec", "eval(", "new Function"):
    if forbidden in javascript:
        failures.append(f"forbidden={forbidden}")

action = (ROOT / "src/magisk-module/action.sh").read_text(encoding="utf-8")
for required in ("-token-file", "/data/local/tmp/", "/bootstrap?token=", "-self-test"):
    if required not in action:
        failures.append(f"action_contract={required}")
if ' -token "$TOKEN"' in action or " -token " in action:
    failures.append("token_in_argv")

control = (ROOT / "src/magisk-module/bin/module-control").read_text(encoding="utf-8")
for operation in ("capabilities)", "status)", "log)", "inventory)"):
    if operation not in control:
        failures.append(f"control_operation={operation}")
for forbidden_operation in ("config-apply)", "action-file)", "job-run)"):
    if forbidden_operation in control:
        failures.append(f"read_only_operation_present={forbidden_operation}")
if '"features":{"config":false,"logs":true,"actions":false,"jobs":false,"inventory":true}' not in control:
    failures.append("read_only_capabilities_missing")

if failures:
    print("FAIL: webui_contract=" + ",".join(failures))
    sys.exit(1)
print("RESULT: WEBUI_CONTRACT_TEST_PASS")
