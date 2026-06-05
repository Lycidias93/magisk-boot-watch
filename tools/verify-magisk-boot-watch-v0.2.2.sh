#!/usr/bin/env bash
set -euo pipefail
ZIP="${1:-releases/v0.2.2/magisk-boot-watch-v0.2.2.zip}"
test -s "$ZIP"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
unzip -q "$ZIP" -d "$WORK/module"
grep -qx 'id=boot-watch' "$WORK/module/module.prop"
grep -qx 'version=0.2.2' "$WORK/module/module.prop"
grep -qx 'versionCode=22' "$WORK/module/module.prop"
grep -q 'updateJson=https://raw.githubusercontent.com/Lycidias93/magisk-boot-watch/main/update.json' "$WORK/module/module.prop"
grep -q 'auto_export_start=' "$WORK/module/boot-watch.sh"
grep -Fq '/system/bin/sh "$MOD/result-log-export.sh"' "$WORK/module/boot-watch.sh"
grep -q 'LEGACY_MOD="/data/adb/modules/pixel-boot-watch"' "$WORK/module/customize.sh"
! grep -q 'LEGACY_MOD="/data/adb/modules/boot-watch"' "$WORK/module/customize.sh"
grep -Fq 'rm -f "$MODPATH/disable" "$MODPATH/remove"' "$WORK/module/customize.sh"
sh -n "$WORK/module/service.sh"
sh -n "$WORK/module/boot-watch.sh"
sh -n "$WORK/module/result-log-export.sh"
sh -n "$WORK/module/action.sh"
sh -n "$WORK/module/customize.sh"
echo 'RESULT: MAGISK_BOOT_WATCH_VERIFY_PASS version=0.2.2'
