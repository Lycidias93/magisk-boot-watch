#!/system/bin/sh
# Preserve evidence runs by default. Only remove module files; runtime logs remain.
echo "Boot Watch Collector module removed. Runtime evidence remains under /data/adb/boot-watch" > /data/adb/boot-watch/uninstalled.txt 2>/dev/null || true
