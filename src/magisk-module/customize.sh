#!/system/bin/sh
SKIPUNZIP=0
LEGACY_MOD="/data/adb/modules/pixel-boot-watch"
ui_print "********************************"
ui_print " Boot Watch Collector v0.2.0"
ui_print "********************************"
ui_print "- New module id: boot-watch"
ui_print "- Runtime: /data/adb/boot-watch"
ui_print "- Online updates: enabled through updateJson"
ui_print "- Legacy id migration: pixel-boot-watch -> boot-watch"
if [ -d "$LEGACY_MOD" ]; then
  touch "$LEGACY_MOD/disable" "$LEGACY_MOD/remove" 2>/dev/null || true
  ui_print "- Marked legacy pixel-boot-watch module for removal"
else
  ui_print "- Legacy pixel-boot-watch module not present"
fi
ui_print "- Reboot required"
