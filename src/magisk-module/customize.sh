#!/system/bin/sh
SKIPUNZIP=0
LEGACY_MOD="/data/adb/modules/boot-watch"
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

# v0.2.1 executable permission hardening
ui_print "- v0.2.1: robust boot export hook"
ui_print "- Setting executable permissions"
set_perm "$MODPATH/service.sh" 0 0 0755
set_perm "$MODPATH/boot-watch.sh" 0 0 0755
set_perm "$MODPATH/result-log-export.sh" 0 0 0755
set_perm "$MODPATH/action.sh" 0 0 0755
set_perm "$MODPATH/manual-collect.sh" 0 0 0755
set_perm "$MODPATH/uninstall.sh" 0 0 0755
