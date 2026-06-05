#!/usr/bin/env bash
set -euo pipefail
ZIP="${1:?zip path required}"
tmp="$(mktemp -d)"
cleanup() { rm -rf "$tmp"; }
trap cleanup EXIT
unzip -q "$ZIP" -d "$tmp"
grep -q '^id=boot-watch$' "$tmp/module.prop"
grep -q '^version=0.2.3$' "$tmp/module.prop"
grep -q '^versionCode=23$' "$tmp/module.prop"
grep -q '^updateJson=https://raw.githubusercontent.com/Lycidias93/magisk-boot-watch/main/update.json$' "$tmp/module.prop"
grep -q 'LEGACY_MOD="/data/adb/modules/pixel-boot-watch"' "$tmp/service.sh"
grep -q 'ACTIVE_MOD="/data/adb/modules/boot-watch"' "$tmp/service.sh"
grep -q 'active_marker_self_heal=done' "$tmp/service.sh"
if grep -q 'LEGACY_MOD="/data/adb/modules/boot-watch"' "$tmp/service.sh"; then
  echo "FAIL: service.sh still points LEGACY_MOD to active module" >&2
  exit 1
fi
grep -q 'auto_export_start' "$tmp/boot-watch.sh"
grep -q 'result-log-export.sh' "$tmp/boot-watch.sh"
for f in service.sh boot-watch.sh result-log-export.sh action.sh manual-collect.sh uninstall.sh; do
  test -x "$tmp/$f"
done
echo "RESULT: MAGISK_BOOT_WATCH_VERIFY_PASS version=0.2.3"
