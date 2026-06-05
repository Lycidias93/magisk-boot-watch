#!/system/bin/sh
MODDIR="${0%/*}"
RT="/data/adb/pixel-boot-watch"
mkdir -p "$RT"
{
  echo "service_start=$(date +%Y-%m-%dT%H:%M:%S%z)"
  echo "module=pixel-boot-watch"
  echo "version=0.1.5.1"
} >> "$RT/service-launch.log" 2>&1
(
  /system/bin/sh "$MODDIR/boot-watch.sh" boot >> "$RT/service-launch.log" 2>&1
) &
exit 0
