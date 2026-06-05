#!/system/bin/sh
set -eu
MODDIR="${0%/*}"
if [ ! -x "$MODDIR/boot-watch.sh" ]; then
  echo "missing_boot_watch=$MODDIR/boot-watch.sh" >&2
  exit 2
fi
/system/bin/sh "$MODDIR/boot-watch.sh" manual
latest="$(ls -1d /data/adb/boot-watch/runs/*_manual 2>/dev/null | tail -1 || true)"
if [ -z "$latest" ]; then
  latest="$(ls -1d /data/adb/boot-watch/runs/* 2>/dev/null | tail -1 || true)"
fi
if [ -n "$latest" ] && [ -x "$MODDIR/result-log-export.sh" ]; then
  /system/bin/sh "$MODDIR/result-log-export.sh" "$latest" || true
fi
