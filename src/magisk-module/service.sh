#!/system/bin/sh
MODDIR="${0%/*}"
RT="/data/adb/boot-watch"
LEGACY_MOD="/data/adb/modules/pixel-boot-watch"
mkdir -p "$RT"
{
  echo "service_start=$(date +%Y-%m-%dT%H:%M:%S%z)"
  echo "module=boot-watch"
  echo "version=0.2.0"
  if [ -d "$LEGACY_MOD" ]; then
    touch "$LEGACY_MOD/disable" "$LEGACY_MOD/remove" 2>/dev/null || true
    echo "legacy_pixel_boot_watch_marked_remove=yes"
  else
    echo "legacy_pixel_boot_watch_marked_remove=no_legacy_module"
  fi
} >> "$RT/service-launch.log" 2>&1
(
  /system/bin/sh "$MODDIR/boot-watch.sh" boot >> "$RT/service-launch.log" 2>&1
) &
exit 0
