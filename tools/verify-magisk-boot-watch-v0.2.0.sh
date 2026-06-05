#!/usr/bin/env bash
set -euo pipefail
ZIP="${1:-releases/v0.2.0/magisk-boot-watch-v0.2.0.zip}"
WORK="${TMPDIR:-.}/mbw_verify_$$"
rm -rf "$WORK"
mkdir -p "$WORK"
unzip -q "$ZIP" -d "$WORK"
test -f "$WORK/module.prop"
grep -qx 'id=boot-watch' "$WORK/module.prop"
grep -qx 'name=Boot Watch Collector' "$WORK/module.prop"
grep -qx 'version=0.2.0' "$WORK/module.prop"
grep -qx 'versionCode=20' "$WORK/module.prop"
grep -q '^updateJson=https://raw.githubusercontent.com/Lycidias93/magisk-boot-watch/main/update.json$' "$WORK/module.prop"
test -x "$WORK/service.sh"
test -x "$WORK/boot-watch.sh"
test -x "$WORK/result-log-export.sh"
for f in "$WORK"/*.sh; do
  test -f "$f" || continue
  if command -v file >/dev/null 2>&1; then
    file "$f" | grep -q CRLF && { echo "FAIL crlf $f"; exit 1; } || true
  fi
  sh -n "$f"
done
grep -R 'id=pixel-boot-watch' "$WORK/module.prop" && { echo 'FAIL legacy id in module.prop'; exit 1; } || true
echo "RESULT: MAGISK_BOOT_WATCH_VERIFY_PASS version=0.2.0"
rm -rf "$WORK"
