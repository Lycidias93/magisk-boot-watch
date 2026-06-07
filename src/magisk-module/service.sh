#!/system/bin/sh
MODDIR="${0%/*}"
RT="/data/adb/boot-watch"
LEGACY_MOD="/data/adb/modules/pixel-boot-watch"
ACTIVE_MOD="/data/adb/modules/boot-watch"
mkdir -p "$RT"
{
  echo "service_start=$(date +%Y-%m-%dT%H:%M:%S%z)"
  echo "module=boot-watch"
  echo "version=0.2.5-test.1"
  if [ -d "$LEGACY_MOD" ]; then
    if [ -f "$LEGACY_MOD/module.prop" ] && grep -q "^id=boot-watch$" "$LEGACY_MOD/module.prop" 2>/dev/null; then
      echo "legacy_pixel_boot_watch_marked_remove=no_active_id_at_legacy_path"
    else
      touch "$LEGACY_MOD/disable" "$LEGACY_MOD/remove" 2>/dev/null || true
      echo "legacy_pixel_boot_watch_marked_remove=yes"
    fi
  else
    echo "legacy_pixel_boot_watch_marked_remove=no_legacy_module"
  fi
  if [ -f "$ACTIVE_MOD/module.prop" ] && grep -q "^id=boot-watch$" "$ACTIVE_MOD/module.prop" 2>/dev/null; then
    rm -f "$ACTIVE_MOD/disable" "$ACTIVE_MOD/remove" 2>/dev/null || true
    echo "active_marker_self_heal=done"
  else
    echo "active_marker_self_heal=skipped"
  fi
} >> "$RT/service-launch.log" 2>&1
(
  /system/bin/sh "$MODDIR/boot-watch.sh" boot >> "$RT/service-launch.log" 2>&1
) &
exit 0
