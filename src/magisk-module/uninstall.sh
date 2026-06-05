#!/system/bin/sh
# Preserve evidence runs by default. Only remove module files; runtime logs remain.
echo "Pixel Boot Watch module removed. Runtime evidence remains under /data/adb/pixel-boot-watch" > /data/adb/pixel-boot-watch/uninstalled.txt 2>/dev/null || true
