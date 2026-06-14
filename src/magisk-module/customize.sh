#!/system/bin/sh
SKIPUNZIP=0
LEGACY_MOD="/data/adb/modules/pixel-boot-watch"
ACTIVE_MOD="/data/adb/modules/boot-watch"
ui_print "********************************"
ui_print " Boot Watch Collector v0.2.6-test.1"
ui_print "********************************"
ui_print "- New module id: boot-watch"
ui_print "- Runtime: /data/adb/boot-watch"
ui_print "- Online updates: enabled through updateJson"
ui_print "- Legacy id migration: pixel-boot-watch -> boot-watch"
if [ -d "$LEGACY_MOD" ]; then
  if [ -f "$LEGACY_MOD/module.prop" ] && grep -q "^id=boot-watch$" "$LEGACY_MOD/module.prop" 2>/dev/null; then
    ui_print "- Legacy path already contains boot-watch id; not marking active module"
  else
    touch "$LEGACY_MOD/disable" "$LEGACY_MOD/remove" 2>/dev/null || true
    ui_print "- Marked legacy pixel-boot-watch module for removal"
  fi
else
  ui_print "- Legacy pixel-boot-watch module not present"
fi
if [ -n "${MODPATH:-}" ]; then
  rm -f "$MODPATH/disable" "$MODPATH/remove" 2>/dev/null || true
fi
if [ -f "$ACTIVE_MOD/module.prop" ] && grep -q "^id=boot-watch$" "$ACTIVE_MOD/module.prop" 2>/dev/null; then
  rm -f "$ACTIVE_MOD/disable" "$ACTIVE_MOD/remove" 2>/dev/null || true
fi
ui_print "- v0.2.6-test.1: diagnostics bundle vNext"
ui_print "- Adds AshLooper intervention, split logcat, pstore and focused dumpsys"
ui_print "- Stable update channel remains on v0.2.5"
ui_print "- v0.2.4: service-time active-module marker guard"
ui_print "- v0.2.4: robust boot export hook"
ui_print "- Setting executable permissions"
set_perm "$MODPATH/service.sh" 0 0 0755
set_perm "$MODPATH/boot-watch.sh" 0 0 0755
set_perm "$MODPATH/result-log-export.sh" 0 0 0755
set_perm "$MODPATH/action.sh" 0 0 0755
set_perm "$MODPATH/manual-collect.sh" 0 0 0755
set_perm "$MODPATH/uninstall.sh" 0 0 0755
ui_print "- Reboot required"
