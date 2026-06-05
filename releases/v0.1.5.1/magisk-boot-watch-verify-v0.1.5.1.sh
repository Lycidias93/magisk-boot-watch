#!/usr/bin/env bash
set -euo pipefail
ZIP="${1:-pixel-boot-watch-magisk-v0.1.5.1.zip}"
if [ ! -s "$ZIP" ]; then
  echo "RESULT: PIXEL_BOOT_WATCH_VERIFY_FAIL reason=zip_missing path=$ZIP"
  exit 2
fi
WORK="${TMPDIR:-/tmp}/pbw_verify_v0151_$$"
rm -rf "$WORK"
mkdir -p "$WORK"
unzip -q "$ZIP" -d "$WORK"
need_files="module.prop boot-watch.sh result-log-export.sh action.sh service.sh customize.sh README.md"
for f in $need_files; do
  test -s "$WORK/$f" || { echo "RESULT: PIXEL_BOOT_WATCH_VERIFY_FAIL reason=missing_$f"; exit 3; }
done
grep -q '^id=pixel-boot-watch$' "$WORK/module.prop"
grep -q '^version=0.1.5.1$' "$WORK/module.prop"
grep -q '^versionCode=16$' "$WORK/module.prop"
grep -q 'pixel_local__pixel-boot-watch-' "$WORK/result-log-export.sh"
grep -q 'pixel_local__pixel-boot-watch-status.env' "$WORK/result-log-export.sh"
grep -q 'pbw_file_name_too_long_count' "$WORK/result-log-export.sh"
grep -q 'BOOTWATCH_STATUS' "$WORK/result-log-export.sh"
grep -q 'collect_ashlooper_health' "$WORK/boot-watch.sh"
grep -q 'red_flags_summary' "$WORK/boot-watch.sh"
grep -q 'file_name_too_long_count' "$WORK/boot-watch.sh"
grep -q '^VERSION="0.1.5.1"' "$WORK/boot-watch.sh"
grep -q '^VERSION_CODE="16"' "$WORK/boot-watch.sh"
for f in boot-watch.sh result-log-export.sh action.sh service.sh customize.sh; do
  sh -n "$WORK/$f"
done
sha256sum "$ZIP"
echo "RESULT: PIXEL_BOOT_WATCH_VERIFY_PASS version=0.1.5.1"
