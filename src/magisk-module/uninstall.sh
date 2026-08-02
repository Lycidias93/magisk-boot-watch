#!/system/bin/sh
set -u

MODDIR=${0%/*}
SERVER="$MODDIR/bin/webui-server-arm64"
RUNTIME_DIR="/data/local/tmp/boot-watch-webui"
PID_FILE="$RUNTIME_DIR/server.pid"

if [ -f "$PID_FILE" ]; then
  PID=$(cat "$PID_FILE" 2>/dev/null || true)
  case "$PID" in ""|*[!0-9]*) PID="" ;; esac
  if [ -n "$PID" ] && [ -r "/proc/$PID/cmdline" ] &&
    tr '\000' ' ' < "/proc/$PID/cmdline" 2>/dev/null | grep -Fq "$SERVER"; then
    kill "$PID" 2>/dev/null || true
  fi
fi

rm -rf "$RUNTIME_DIR"
mkdir -p /data/adb/boot-watch 2>/dev/null || true
printf '%s\n' "Boot Watch Collector module removed. Runtime evidence remains under /data/adb/boot-watch" > /data/adb/boot-watch/uninstalled.txt 2>/dev/null || true
exit 0
