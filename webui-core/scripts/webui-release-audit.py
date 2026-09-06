#!/usr/bin/env python3
from __future__ import annotations

import re
import sys
from html.parser import HTMLParser
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class AssetParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.assets: list[str] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        values = dict(attrs)
        if tag == "script" and values.get("src"):
            self.assets.append(values["src"] or "")
        if tag == "link" and values.get("rel") == "stylesheet" and values.get("href"):
            self.assets.append(values["href"] or "")


def normalize_asset(value: str) -> str:
    return value.split("?", 1)[0].split("#", 1)[0].lstrip("/")


def referenced_assets(html: str) -> list[str]:
    parser = AssetParser()
    parser.feed(html)
    return sorted({normalize_asset(item) for item in parser.assets if normalize_asset(item)})


def server_asset_routes(source: str) -> set[str]:
    routes: set[str] = set()
    for match in re.finditer(r'case\s+((?:"[A-Za-z0-9._/-]+"(?:\s*,\s*)?)+)\s*:', source):
        routes.update(item.lstrip("/") for item in re.findall(r'"([A-Za-z0-9._/-]+)"', match.group(1)))
    for match in re.finditer(r'(?:HandleFunc|Handle)\(\s*"/([A-Za-z0-9._/-]+)"', source):
        routes.add(match.group(1).lstrip("/"))
    return routes


def audit_asset_parity(html: str, server_source: str) -> tuple[list[str], list[str]]:
    assets = referenced_assets(html)
    routes = server_asset_routes(server_source)
    missing = [asset for asset in assets if asset not in routes]
    return assets, missing


def integration_contract(source: str) -> list[str]:
    required = {
        "bootstrap": '"$BASE/bootstrap?token=$TOKEN"',
        "root_page": '"$BASE/"',
        "capabilities": '"$BASE/api/v1/capabilities"',
        "status": '"$BASE/api/v1/status"',
        "config_post": '"$BASE/api/v1/config"',
        "config_roundtrip": '\"mode\":\"battery\"',
        "action": '"$BASE/api/v1/action"',
        "jobs": '"$BASE/api/v1/jobs"',
        "inventory": '"$BASE/api/v1/inventory?name=examples"',
        "unauthenticated": 'unauthenticated=',
        "origin_rejected": 'origin_rejected=',
    }
    return [name for name, needle in required.items() if needle not in source]


def self_test() -> int:
    html = '<link rel="stylesheet" href="a.css"><script src="/a.js"></script>'
    good_server = 'switch x { case "a.css", "a.js": }'
    bad_server = 'switch x { case "a.css": }'
    assets, missing = audit_asset_parity(html, good_server)
    if assets != ["a.css", "a.js"] or missing:
        print(f"FAIL: self_test_positive assets={assets} missing={missing}")
        return 1
    _, missing = audit_asset_parity(html, bad_server)
    if missing != ["a.js"]:
        print(f"FAIL: self_test_negative missing={missing}")
        return 1
    print("RESULT: WEBUI_RELEASE_AUDIT_SELF_TEST_PASS")
    return 0


def main() -> int:
    if sys.argv[1:] == ["--self-test"]:
        return self_test()
    if sys.argv[1:]:
        print("usage: webui-release-audit.py [--self-test]", file=sys.stderr)
        return 2

    html_path = ROOT / "module/webroot/index.html"
    server_paths = [
        ROOT / "server/cmd/webui-server/main.go",
        ROOT / "server/cmd/webui-server/v03.go",
        ROOT / "server/cmd/webui-server/v04.go",
    ]
    integration_path = ROOT / "scripts/integration-test.sh"

    failures = 0
    html = html_path.read_text(encoding="utf-8")
    server_source = "\n".join(path.read_text(encoding="utf-8") for path in server_paths)
    integration = integration_path.read_text(encoding="utf-8")

    assets, missing_assets = audit_asset_parity(html, server_source)
    print(f"asset_routes_total={len(assets)}")
    print(f"asset_routes_pass={len(assets) - len(missing_assets)}")
    if missing_assets:
        failures += len(missing_assets)
        print("asset_routes_missing=" + ",".join(missing_assets))
    else:
        print("asset_routes_missing=none")

    missing_integration = integration_contract(integration)
    if missing_integration:
        failures += len(missing_integration)
        print("integration_contract=FAIL")
        print("integration_contract_missing=" + ",".join(missing_integration))
    else:
        print("integration_contract=PASS")
        print("integration_contract_missing=none")

    print("evidence_collection=complete")
    print(f"failure_count={failures}")
    if failures:
        print("verdict=fail")
        print("RESULT: WEBUI_RELEASE_AUDIT_STATIC_FAIL outcome=failure workflow_exit_code=1")
        return 1

    print("verdict=pass")
    print("RESULT: WEBUI_RELEASE_AUDIT_STATIC_PASS outcome=success workflow_exit_code=0")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
