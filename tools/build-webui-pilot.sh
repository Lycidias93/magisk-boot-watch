#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

./tools/verify-webui-pilot.sh

BUILD_DIR="$ROOT/build/webui-pilot"
DIST_DIR="$ROOT/dist"
STAGE="$BUILD_DIR/module"

rm -rf "$BUILD_DIR"
mkdir -p "$STAGE" "$DIST_DIR"
cp -a src/magisk-module/. "$STAGE/"
cp -f LICENSE WEBUI_CREDITS.md webui.lock "$STAGE/"
mkdir -p "$STAGE/third_party/licenses"
cp -f third_party/licenses/*.LICENSE "$STAGE/third_party/licenses/"

version=$(sed -n 's/^version=//p' src/magisk-module/module.prop | head -n 1)
version_code=$(sed -n 's/^versionCode=//p' src/magisk-module/module.prop | head -n 1)
module_id=$(sed -n 's/^id=//p' src/magisk-module/module.prop | head -n 1)
core_version=$(sed -n 's/^core_version=//p' webui.lock | head -n 1)
core_commit=$(sed -n 's/^core_commit=//p' webui.lock | head -n 1)

[[ -n "$version" && -n "$version_code" && -n "$module_id" && -n "$core_version" && -n "$core_commit" ]]

(
  cd webui-core
  CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build \
    -buildvcs=false \
    -trimpath \
    -ldflags "-s -w -X main.version=${core_version}" \
    -o "$STAGE/bin/webui-server-arm64" \
    ./server/cmd/webui-server
)

chmod 0755 \
  "$STAGE/action.sh" \
  "$STAGE/customize.sh" \
  "$STAGE/service.sh" \
  "$STAGE/uninstall.sh" \
  "$STAGE/boot-watch.sh" \
  "$STAGE/result-log-export.sh" \
  "$STAGE/manual-collect.sh" \
  "$STAGE/bin/module-control" \
  "$STAGE/bin/webui-server-arm64" \
  "$STAGE/META-INF/com/google/android/update-binary"

artifact="$DIST_DIR/${module_id}-${version}.zip"
repro="$BUILD_DIR/repro.zip"
python3 tools/package-webui-pilot.py "$STAGE" "$artifact"
python3 tools/package-webui-pilot.py "$STAGE" "$repro"
cmp -s "$artifact" "$repro" || {
  echo "FAIL package_not_reproducible"
  exit 1
}

unzip -tq "$artifact" >/dev/null
entries=$(unzip -Z1 "$artifact")
for entry in \
  module.prop \
  action.sh \
  boot-watch.sh \
  result-log-export.sh \
  bin/module-control \
  bin/webui-server-arm64 \
  webroot/index.html \
  webroot/app.js \
  webroot/app.css \
  webui.lock \
  LICENSE \
  WEBUI_CREDITS.md \
  third_party/licenses/F2FS-Optimizer.LICENSE; do
  grep -Fxq "$entry" <<< "$entries" || {
    echo "FAIL archive_entry_missing entry=$entry"
    exit 1
  }
done

for forbidden in \
  webroot/boot-watch-status.json \
  webroot/boot-watch-logs.json \
  webroot/boot-watch-runs.json \
  tools/boot-watch-webui-status-export.sh \
  tools/boot-watch-webui-log-export.sh; do
  if grep -Fxq "$forbidden" <<< "$entries"; then
    echo "FAIL legacy_archive_entry entry=$forbidden"
    exit 1
  fi
done

sha256=$(sha256sum "$artifact" | awk '{print $1}')
bytes=$(wc -c < "$artifact" | tr -d ' ')
manifest="$DIST_DIR/webui-pilot-build-manifest.json"
cat > "$manifest" <<EOF_MANIFEST
{
  "schema": "boot-watch-webui-pilot-build.v1",
  "module_id": "$module_id",
  "module_version": "$version",
  "module_version_code": $version_code,
  "core_version": "$core_version",
  "core_commit": "$core_commit",
  "artifact": "$(basename "$artifact")",
  "sha256": "$sha256",
  "bytes": $bytes
}
EOF_MANIFEST

echo "artifact=$artifact"
echo "artifact_sha256=$sha256"
echo "artifact_bytes=$bytes"
echo "build_manifest=$manifest"
echo "RESULT: BOOT_WATCH_WEBUI_BUILD_PASS"
