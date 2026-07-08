#!/system/bin/sh
# BOOT_WATCH_WEBUI_V028_ACTION_WRAPPER_START
# Safety: Boot-Watch-only, WebUI read-only exports, no host/SSH/route/DNS actions.
set +e
MODDIR="${0%/*}"
export MODDIR
ORIG="$MODDIR/tools/action.original-before-webui-wrapper.sh"
STATEXP="$MODDIR/tools/boot-watch-webui-status-export.sh"
LOGEXP="$MODDIR/tools/boot-watch-webui-log-export.sh"
LOG="$MODDIR/action.webui-wrapper-last.log"
TMP="$LOG.tmp.$$"

printf '%s\n' "BOOT_WATCH_WEBUI_ACTION_WRAPPER version=0.2.10-webui-runtime-root-hotfix"
printf '%s\n' "scope=bootwatch_only"
printf '%s\n' "routeguard=no DNS/HA/VIP/default-route/static-route/MagicDNS/subnet-route change"

orig_rc=127
if [ -f "$ORIG" ]; then
  sh "$ORIG" > "$TMP" 2>&1
  orig_rc=$?
  cat "$TMP"
else
  printf '%s\n' "WARN original_action_missing"
  : > "$TMP"
fi
mv -f "$TMP" "$LOG" 2>/dev/null || true

webui_rc=127
if [ -x "$STATEXP" ]; then
  sh "$STATEXP"
  webui_rc=$?
fi
printf '%s\n' "webui_export_rc=$webui_rc"

log_rc=127
if [ -x "$LOGEXP" ]; then
  sh "$LOGEXP"
  log_rc=$?
fi
printf '%s\n' "log_export_rc=$log_rc"
printf '%s\n' "original_action_rc=$orig_rc"

if [ "$webui_rc" = 0 ] && [ "$log_rc" = 0 ] && [ "$orig_rc" = 2 ]; then
  if grep -q 'reason=no_result_export' "$LOG" 2>/dev/null; then
    printf '%s\n' "WARN legacy_action_no_result_export_treated_as_webui_refresh_only"
    printf '%s\n' "RESULT: BOOT_WATCH_WEBUI_ACTION_REFRESH_DONE rc=0 reason=legacy_no_result_export"
    exit 0
  fi
fi

if [ "$webui_rc" != 0 ]; then
  printf '%s\n' "RESULT: BOOT_WATCH_WEBUI_ACTION_REFRESH_DONE rc=$webui_rc reason=webui_export_failed"
  exit "$webui_rc"
fi
if [ "$log_rc" != 0 ]; then
  printf '%s\n' "RESULT: BOOT_WATCH_WEBUI_ACTION_REFRESH_DONE rc=$log_rc reason=log_export_failed"
  exit "$log_rc"
fi
if [ "$orig_rc" != 0 ]; then
  printf '%s\n' "RESULT: BOOT_WATCH_WEBUI_ACTION_REFRESH_DONE rc=$orig_rc reason=original_action_failed"
  exit "$orig_rc"
fi

printf '%s\n' "RESULT: BOOT_WATCH_WEBUI_ACTION_REFRESH_DONE rc=0"
exit 0
