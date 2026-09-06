#!/usr/bin/env bash
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"
./tools/verify-vnext.sh
BUILD="$ROOT/build/vnext"
STAGE="$BUILD/module"
DIST="$ROOT/dist"
rm -rf "$BUILD"
mkdir -p "$STAGE" "$DIST"
cp -a src/magisk-module/. "$STAGE/"
cp -f LICENSE WEBUI_CREDITS.md webui.lock "$STAGE/"
mkdir -p "$STAGE/third_party/licenses"
cp -f third_party/licenses/*.LICENSE "$STAGE/third_party/licenses/"
core_version=$(cat webui-core/CORE_VERSION)
(cd webui-core && CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=$core_version" -o "$STAGE/bin/webui-server-arm64" ./server/cmd/webui-server)
chmod 0755 "$STAGE/action.sh" "$STAGE/customize.sh" "$STAGE/service.sh" "$STAGE/uninstall.sh" "$STAGE/boot-watch.sh" "$STAGE/result-log-export.sh" "$STAGE/manual-collect.sh" "$STAGE/bin/module-control" "$STAGE/bin/webui-server-arm64" "$STAGE/META-INF/com/google/android/update-binary"
artifact="$DIST/boot-watch-0.2.11-vnext.1.zip"
repro="$BUILD/repro.zip"
python3 tools/package-vnext.py "$STAGE" "$artifact"
python3 tools/package-vnext.py "$STAGE" "$repro"
cmp -s "$artifact" "$repro" || { echo "FAIL package_not_reproducible"; exit 1; }
unzip -tq "$artifact" >/dev/null
for e in module.prop action.sh boot-watch.sh result-log-export.sh bin/module-control bin/webui-server-arm64 webroot/index.html webroot/embedded-host-bootstrap.js webroot/mobile-input-viewport.js webroot/observability.js webui.lock; do unzip -Z1 "$artifact" | grep -Fxq "$e" || { echo "FAIL entry=$e"; exit 1; }; done
sha=$(sha256sum "$artifact" | awk '{print $1}')
bytes=$(wc -c < "$artifact" | tr -d " ")
printf '{"schema":"boot-watch-vnext-build.v1","module_version":"0.2.11-vnext.1","version_code":36,"core_version":"0.6.3","core_commit":"6791a05be79f162979c76a286f7cdbdd9ce1cc6b","artifact":"boot-watch-0.2.11-vnext.1.zip","sha256":"%s","bytes":%s,"webui_release_audit_required":true,"device_webui_audit":"pending"}\n' "$sha" "$bytes" > "$DIST/vnext-build-manifest.json"
echo "artifact=$artifact"
echo "artifact_sha256=$sha"
echo "artifact_bytes=$bytes"
echo 'RESULT: BOOT_WATCH_VNEXT_BUILD_PASS'
