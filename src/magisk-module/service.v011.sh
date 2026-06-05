#!/system/bin/sh
# Boot Watch Collector original service launcher preserved for v0.1.3 wrapper compatibility.
MODDIR="${0%/*}"
RT="/data/adb/boot-watch"
LOG="$RT/service-launch.log"
mkdir -p "$RT"
(
  echo "start=$(date -Is)"
  echo "version=0.1.3-service-v011-compatible"
  if [ -x "$MODDIR/boot-watch.sh" ]; then
    /system/bin/sh "$MODDIR/boot-watch.sh" boot
  else
    echo "missing_boot_watch=$MODDIR/boot-watch.sh"
    exit 2
  fi
) >> "$LOG" 2>&1 &
