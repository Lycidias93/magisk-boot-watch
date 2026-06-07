#!/system/bin/sh
MODDIR="${0%/*}"
RT="/data/adb/boot-watch"
DL="/storage/emulated/0/Download"
LAST="$DL/pixel_local__boot-watch-last-result.txt"
ACTION="$DL/pixel_local__boot-watch-action-last-result.txt"
STATUS="$DL/pixel_local__boot-watch-status.env"
mkdir -p "$RT" "$DL"

if [ "${1:-}" = "--status" ]; then
  echo "mode=status"
  echo "module=boot-watch"
  grep -E '^(id|name|version|versionCode|description)=' "$MODDIR/module.prop" 2>/dev/null || true
  for f in "$LAST" "$ACTION" "$STATUS"; do
    if [ -f "$f" ]; then
      echo "file_present=$f"
      ls -l "$f" 2>/dev/null || true
    else
      echo "file_missing=$f"
    fi
  done
  [ -f "$STATUS" ] && cat "$STATUS" 2>/dev/null || true
  echo "RESULT: BOOT_WATCH_ACTION_STATUS_DONE rc=0"
  exit 0
fi

before_last_result_exists=no
before_action_result_exists=no
before_status_exists=no
[ -f "$LAST" ] && before_last_result_exists=yes
[ -f "$ACTION" ] && before_action_result_exists=yes
[ -f "$STATUS" ] && before_status_exists=yes
printf 'before_last_result_exists=%s
' "$before_last_result_exists"
printf 'before_action_result_exists=%s
' "$before_action_result_exists"
printf 'before_status_exists=%s
' "$before_status_exists"

if [ -x "$MODDIR/result-log-export.sh" ]; then
  /system/bin/sh "$MODDIR/result-log-export.sh" "" action
  rc=$?
else
  echo "RESULT: BOOT_WATCH_ACTION_EXPORT_DONE rc=2 reason=no_result_export"
  exit 2
fi

after_last_result_exists=no
after_action_result_exists=no
after_status_exists=no
[ -f "$LAST" ] && after_last_result_exists=yes
[ -f "$ACTION" ] && after_action_result_exists=yes
[ -f "$STATUS" ] && after_status_exists=yes
printf 'after_last_result_exists=%s
' "$after_last_result_exists"
printf 'after_action_result_exists=%s
' "$after_action_result_exists"
printf 'after_status_exists=%s
' "$after_status_exists"
[ -f "$LAST" ] && printf 'last_result_size=%s
' "$(wc -c < "$LAST" 2>/dev/null || echo 0)"
[ -f "$ACTION" ] && printf 'action_result_size=%s
' "$(wc -c < "$ACTION" 2>/dev/null || echo 0)"
[ -f "$STATUS" ] && printf 'status_env_size=%s
' "$(wc -c < "$STATUS" 2>/dev/null || echo 0)"
echo "protected_names=yes"
echo "sortify_hold_expected=yes"
[ "$rc" = "0" ] || exit "$rc"
echo "RESULT: BOOT_WATCH_ACTION_EXPORT_DONE rc=0"
exit 0
