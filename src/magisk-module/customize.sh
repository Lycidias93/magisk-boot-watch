#!/system/bin/sh
SKIPUNZIP=0
LEGACY_MOD="/data/adb/modules/pixel-boot-watch"
ACTIVE_MOD="/data/adb/modules/boot-watch"

ui_print "********************************"
ui_print " Boot Watch WebUI Core Pilot"
ui_print "********************************"
ui_print "- Module: 0.2.11-webui-core-pilot.1"
ui_print "- Shared WebUI core: 0.2.1"
ui_print "- Runtime state: /data/adb/boot-watch"
ui_print "- WebUI session: /data/local/tmp/boot-watch-webui"
ui_print "- Browser host: loopback only"
ui_print "- Collector logic: unchanged"
ui_print "- WebUI capability: read-only"

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

[ -n "${MODPATH:-}" ] && rm -f "$MODPATH/disable" "$MODPATH/remove" 2>/dev/null || true
if [ -f "$ACTIVE_MOD/module.prop" ] && grep -q "^id=boot-watch$" "$ACTIVE_MOD/module.prop" 2>/dev/null; then
  rm -f "$ACTIVE_MOD/disable" "$ACTIVE_MOD/remove" 2>/dev/null || true
fi

for file in \
  service.sh \
  boot-watch.sh \
  result-log-export.sh \
  action.sh \
  manual-collect.sh \
  uninstall.sh \
  bin/module-control \
  bin/webui-server-arm64
do
  set_perm "$MODPATH/$file" 0 0 0755
done
set_perm "$MODPATH/META-INF/com/google/android/update-binary" 0 0 0755

ui_print "- Reboot required"
