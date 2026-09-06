#!/usr/bin/env bash
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"
required=(README.md LICENSE WEBUI_CREDITS.md webui.lock webui-core/CORE_VERSION webui-core/go.mod webui-core/core/manifest.txt src/magisk-module/module.prop src/magisk-module/action.sh src/magisk-module/boot-watch.sh src/magisk-module/result-log-export.sh src/magisk-module/bin/module-control tools/verify-webui-core-pin.py tools/webui-consumer-contract-test.py tools/webui-integration-test.sh tools/webui-release-audit.py tools/package-vnext.py)
for f in "${required[@]}"; do [[ -s "$f" ]] || { echo "FAIL missing=$f"; exit 1; }; done
while IFS= read -r -d '' f; do sh -n "$f"; done < <(find src/magisk-module -type f -name '*.sh' -print0)
sh -n src/magisk-module/bin/module-control
bash -n tools/webui-integration-test.sh
python3 -m py_compile tools/verify-webui-core-pin.py tools/webui-consumer-contract-test.py tools/webui-release-audit.py tools/package-vnext.py
grep -Fxq 'version=0.2.11-vnext.1' src/magisk-module/module.prop
grep -Fxq 'versionCode=36' src/magisk-module/module.prop
grep -Fq 'VERSION="0.2.11-vnext.1"' src/magisk-module/boot-watch.sh
grep -Fq 'VERSION_CODE="36"' src/magisk-module/boot-watch.sh
grep -Fxq 'core_version=0.6.3' webui.lock
grep -Fxq 'source_commit=6791a05be79f162979c76a286f7cdbdd9ce1cc6b' webui.lock
grep -Fq 'collect_zygisk_stack_support' src/magisk-module/boot-watch.sh
grep -Fq 'magisk --denylist ls' src/magisk-module/boot-watch.sh
grep -Fq 'lspd_config_contents_collected=no' src/magisk-module/boot-watch.sh
! grep -Fq '/data/user_de/0/' src/magisk-module/boot-watch.sh
for forbidden in src/magisk-module/tools/boot-watch-webui-log-export.sh src/magisk-module/tools/boot-watch-webui-status-export.sh src/magisk-module/tools/action.original-before-webui-wrapper.sh src/magisk-module/webroot/boot-watch-status.json src/magisk-module/webroot/boot-watch-logs.json src/magisk-module/webroot/boot-watch-runs.json src/magisk-module/webroot/style.css; do [[ ! -e "$forbidden" ]] || { echo "FAIL legacy=$forbidden"; exit 1; }; done
python3 tools/verify-webui-core-pin.py
gofmt_out=$(gofmt -l webui-core/server); [[ -z "$gofmt_out" ]] || { echo "FAIL gofmt"; printf "%s\n" "$gofmt_out"; exit 1; }
(cd webui-core && go vet ./... && go test ./...)
python3 webui-core/scripts/webui-release-audit.py --self-test
python3 tools/webui-consumer-contract-test.py
python3 tools/webui-release-audit.py
./tools/webui-integration-test.sh
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
(cd webui-core && CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -buildvcs=false -trimpath -o "$tmp/webui-server-arm64" ./server/cmd/webui-server)
[[ -s "$tmp/webui-server-arm64" ]]
echo upstream_core_ci_run=34037562507
echo shared_core_acceptance=exact_blob_pin_plus_upstream_ci
echo webui_release_audit_required=yes
echo device_webui_audit=pending_required_before_stable_release
echo 'RESULT: BOOT_WATCH_VNEXT_REPO_VERIFY_PASS'
