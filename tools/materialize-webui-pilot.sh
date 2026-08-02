#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
TEMPLATE=${1:?template checkout required}
EXPECTED_COMMIT=cdb872d2afb9f86300dd26f6474820ab5de3efca

cd "$ROOT"
[[ -f .webui-pilot-materialize-ready ]] || {
  echo "RESULT: BOOT_WATCH_WEBUI_MATERIALIZE_SKIPPED reason=marker_absent"
  exit 0
}

actual_commit=$(git -C "$TEMPLATE" rev-parse HEAD)
[[ "$actual_commit" = "$EXPECTED_COMMIT" ]] || {
  echo "FAIL template_commit expected=$EXPECTED_COMMIT actual=$actual_commit"
  exit 1
}
[[ "$(tr -d '\r\n' < "$TEMPLATE/CORE_VERSION")" = 0.2.1 ]]

install -D -m 0755 "$TEMPLATE/module/action.sh" src/magisk-module/action.sh
install -D -m 0644 "$TEMPLATE/module/webroot/index.html" src/magisk-module/webroot/index.html
install -D -m 0644 "$TEMPLATE/module/webroot/app.js" src/magisk-module/webroot/app.js
install -D -m 0644 "$TEMPLATE/module/webroot/app.css" src/magisk-module/webroot/app.css
install -D -m 0644 "$TEMPLATE/go.mod" webui-core/go.mod
install -D -m 0644 "$TEMPLATE/server/cmd/webui-server/main.go" webui-core/server/cmd/webui-server/main.go
install -D -m 0644 "$TEMPLATE/server/cmd/webui-server/main_test.go" webui-core/server/cmd/webui-server/main_test.go
install -D -m 0755 "$TEMPLATE/scripts/package-module.py" tools/package-webui-pilot.py
mkdir -p third_party/licenses
cp -f "$TEMPLATE"/third_party/licenses/*.LICENSE third_party/licenses/

rm -f \
  src/magisk-module/tools/action.original-before-webui-wrapper.sh \
  src/magisk-module/tools/boot-watch-webui-log-export.sh \
  src/magisk-module/tools/boot-watch-webui-status-export.sh \
  src/magisk-module/webroot/boot-watch-logs.json \
  src/magisk-module/webroot/boot-watch-runs.json \
  src/magisk-module/webroot/boot-watch-status.json \
  src/magisk-module/webroot/style.css

chmod 0755 \
  src/magisk-module/action.sh \
  tools/package-webui-pilot.py

rm -f .webui-pilot-materialize-ready tools/materialize-webui-pilot.sh

git add -A

echo "RESULT: BOOT_WATCH_WEBUI_MATERIALIZE_DONE core_commit=$EXPECTED_COMMIT core_version=0.2.1"
