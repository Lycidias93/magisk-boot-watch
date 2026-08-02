#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

required=(
  README.md
  LICENSE
  WEBUI_CREDITS.md
  webui.lock
  src/magisk-module/module.prop
  src/magisk-module/action.sh
  src/magisk-module/customize.sh
  src/magisk-module/service.sh
  src/magisk-module/uninstall.sh
  src/magisk-module/boot-watch.sh
  src/magisk-module/result-log-export.sh
  src/magisk-module/bin/module-control
  src/magisk-module/webroot/index.html
  src/magisk-module/webroot/app.js
  src/magisk-module/webroot/app.css
  webui-core/go.mod
  webui-core/server/cmd/webui-server/main.go
  webui-core/server/cmd/webui-server/main_test.go
  tools/webui-contract-test.py
  tools/webui-integration-test.sh
  tools/package-webui-pilot.py
  docs/webui-core-pilot.md
  docs/VNEXT.md
)

for file in "${required[@]}"; do
  [[ -s "$file" ]] || {
    echo "FAIL missing_or_empty file=$file"
    exit 1
  }
done

if find . -path ./.git -prune -o -type f -print0 |
  xargs -0 grep -Il $'\r' | grep -q .; then
  echo "FAIL crlf_detected"
  exit 1
fi

for file in tools/verify-webui-pilot.sh tools/build-webui-pilot.sh tools/webui-integration-test.sh; do
  head -n 1 "$file" | grep -Fxq '#!/usr/bin/env bash'
  [[ -x "$file" ]]
  bash -n "$file"
done

for file in \
  src/magisk-module/action.sh \
  src/magisk-module/customize.sh \
  src/magisk-module/service.sh \
  src/magisk-module/uninstall.sh \
  src/magisk-module/boot-watch.sh \
  src/magisk-module/result-log-export.sh \
  src/magisk-module/bin/module-control \
  src/magisk-module/META-INF/com/google/android/update-binary; do
  head -n 1 "$file" | grep -Eq '^#!/(system/bin/sh|sbin/sh)$'
  [[ -x "$file" ]]
  sh -n "$file"
done

grep -Fxq 'id=boot-watch' src/magisk-module/module.prop
grep -Fxq 'version=0.2.11-webui-core-pilot.1' src/magisk-module/module.prop
grep -Fxq 'versionCode=36' src/magisk-module/module.prop
grep -Fq 'core_version=0.2.1' webui.lock
grep -Fq 'core_commit=cdb872d2afb9f86300dd26f6474820ab5de3efca' webui.lock

grep -Fq '/data/local/tmp/' src/magisk-module/action.sh
grep -Fq -- '-token-file' src/magisk-module/action.sh
if grep -Eq -- '(^|[[:space:]])-token([[:space:]]|$)' src/magisk-module/action.sh; then
  echo "FAIL token_passed_in_argv"
  exit 1
fi

for forbidden in \
  src/magisk-module/tools/boot-watch-webui-log-export.sh \
  src/magisk-module/tools/boot-watch-webui-status-export.sh \
  src/magisk-module/tools/action.original-before-webui-wrapper.sh \
  src/magisk-module/webroot/boot-watch-status.json \
  src/magisk-module/webroot/boot-watch-logs.json \
  src/magisk-module/webroot/boot-watch-runs.json \
  src/magisk-module/webroot/style.css; do
  [[ ! -e "$forbidden" ]] || {
    echo "FAIL legacy_webui_file_present file=$forbidden"
    exit 1
  }
done

if grep -RInE 'https?://(cdn|unpkg|jsdelivr|fonts\.googleapis|google-analytics)' src/magisk-module/webroot; then
  echo "FAIL remote_web_asset"
  exit 1
fi
if grep -RInE 'ksu\.exec|apatch\.exec|magisk\.exec|webui\.exec|Android\.exec' src/magisk-module/webroot; then
  echo "FAIL root_exec_bridge_in_core_ui"
  exit 1
fi

grep -Fq '"features":{"config":false,"logs":true,"actions":false,"jobs":false,"inventory":true}' \
  src/magisk-module/bin/module-control
grep -Fq '"read_only_adapter":true' src/magisk-module/bin/module-control
grep -Fq 'Array.isArray(status.summary)' src/magisk-module/webroot/app.js
grep -Fq 'function applyFeatureVisibility()' src/magisk-module/webroot/app.js

gofmt_output=$(gofmt -l webui-core/server)
[[ -z "$gofmt_output" ]] || {
  echo "FAIL gofmt"
  printf '%s\n' "$gofmt_output"
  exit 1
}

(
  cd webui-core
  go vet ./...
  go test ./...
)
python3 tools/webui-contract-test.py
./tools/webui-integration-test.sh

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
(
  cd webui-core
  CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build \
    -buildvcs=false \
    -trimpath \
    -o "$tmp/webui-server-arm64" \
    ./server/cmd/webui-server
)
[[ -s "$tmp/webui-server-arm64" ]]

echo "RESULT: BOOT_WATCH_WEBUI_VERIFY_PASS"
