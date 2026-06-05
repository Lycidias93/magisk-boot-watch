#!/system/bin/sh
# Export latest Pixel Boot Watch run into readable protected Download result log.
RT="/data/adb/pixel-boot-watch"
DL="/storage/emulated/0/Download"
RUN="${1:-}"
MODE="${2:-action}"
if [ -z "$RUN" ]; then
  RUN="$(find "$RT/runs" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort | tail -1)"
fi
if [ -z "$RUN" ] || [ ! -d "$RUN" ]; then
  echo "RESULT: PIXEL_BOOT_WATCH_RESULT_LOG_DONE rc=2 reason=no_run"
  exit 2
fi
RUN_ID="$(basename "$RUN")"
OUT="$DL/pixel_local__pixel-boot-watch-$RUN_ID-result.txt"
LAST="$DL/pixel_local__pixel-boot-watch-last-result.txt"
ACTION="$DL/pixel_local__pixel-boot-watch-action-last-result.txt"
STATUS="$DL/pixel_local__pixel-boot-watch-status.env"
TMP="$OUT.tmp.$$"
STATUS_TMP="$STATUS.tmp.$$"
mkdir -p "$DL"

grep_marker() { grep -E "^$1=" "$RUN/result.marker" 2>/dev/null | tail -1 | cut -d= -f2-; }
grep_summary() { grep -E "^$1=" "$RUN/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-; }
marker_version="$(grep_marker version)"
marker_versionCode="$(grep_marker versionCode)"
marker_archive="$(grep_marker archive_path)"
marker_result="$(grep_marker result_path)"
summary_file_name_too_long="$(grep_summary file_name_too_long)"
pbw_file_name_too_long_count="$(grep_summary file_name_too_long_count)"
if [ -z "$pbw_file_name_too_long_count" ]; then
  pbw_file_name_too_long_count="$(grep -E '^File name too long=' "$RUN/red_flags_summary.txt" 2>/dev/null | tail -1 | sed 's/^File name too long=//')"
fi
[ -n "$pbw_file_name_too_long_count" ] || pbw_file_name_too_long_count=0
case "$pbw_file_name_too_long_count" in
  0) pbw_file_name_too_long=absent ;;
  *[!0-9]*)
    case "$summary_file_name_too_long" in
      absent|""|*'File name too long=0'*) pbw_file_name_too_long=absent; pbw_file_name_too_long_count=0 ;;
      *) pbw_file_name_too_long=present ;;
    esac
    ;;
  *) pbw_file_name_too_long=present ;;
esac

{
  echo "pbw_result=PASS"
  echo "pbw_version=${marker_version:-0.1.5.1}"
  echo "pbw_versionCode=${marker_versionCode:-16}"
  echo "pbw_run_id=$RUN_ID"
  echo "pbw_mode=$MODE"
  echo "pbw_archive_path=${marker_archive}"
  echo "pbw_result_path=$OUT"
  echo "pbw_last_result_path=$LAST"
  echo "pbw_action_result_path=$ACTION"
  echo "pbw_status_env_path=$STATUS"
  echo "pbw_file_name_too_long=$pbw_file_name_too_long"
  echo "pbw_file_name_too_long_count=$pbw_file_name_too_long_count"
  echo "pbw_protected_names=yes"
  echo "pbw_sortify_hold_expected=yes"
  echo "pbw_logd_restored=$(grep_marker logd_state_final)"
  echo "pbw_ashlooper_present=$(grep_summary ashlooper_present)"
  echo "pbw_ashlooper_version=$(grep_summary ashlooper_version)"
  echo "pbw_ashlooper_logs_found=$(grep_summary ashlooper_logs_found)"
  echo "pbw_generated=$(date +%Y-%m-%dT%H:%M:%S%z)"
} > "$STATUS_TMP" 2>/dev/null || true
mv -f "$STATUS_TMP" "$STATUS"

{
  echo "# Pixel Boot Watch result"
  echo "generated=$(date +%Y-%m-%dT%H:%M:%S%z)"
  echo "mode=$MODE"
  echo "run_id=$RUN_ID"
  echo "run_dir=$RUN"
  echo "protected_names=yes"
  echo "sortify_hold_expected=yes"
  echo "status_env_path=$STATUS"
  echo
  echo "## BOOTWATCH_STATUS"
  cat "$STATUS" 2>/dev/null || echo "missing_status_env=yes"
  echo
  echo "## result.marker"
  cat "$RUN/result.marker" 2>/dev/null || echo "missing_result_marker=yes"
  echo
  echo "## summary"
  cat "$RUN/summary.txt" 2>/dev/null || echo "missing_summary=yes"
  echo
  echo "## stages"
  cat "$RUN/stages.txt" 2>/dev/null || echo "missing_stages=yes"
  echo
  echo "## ashlooper health"
  sed -n '1,220p' "$RUN/rescue/ashlooper_health.txt" 2>/dev/null || echo "missing_ashlooper_health=yes"
  echo
  echo "## red flags summary"
  sed -n '1,120p' "$RUN/red_flags_summary.txt" 2>/dev/null || echo "missing_red_flags_summary=yes"
  echo
  echo "## red flags"
  sed -n '1,220p' "$RUN/red_flags.txt" 2>/dev/null || echo "missing_red_flags=yes"
  echo
  echo "## classification"
  sed -n '1,360p' "$RUN/classification.txt" 2>/dev/null || echo "missing_classification=yes"
  echo
  echo "## service health"
  sed -n '1,180p' "$RUN/binder/service_health.txt" 2>/dev/null || true
  echo
  echo "## module matrix"
  sed -n '1,260p' "$RUN/modules/module_matrix.txt" 2>/dev/null || true
  echo
  echo "## zygisk/art focus"
  sed -n '1,220p' "$RUN/zygisk/dex2oat_overlay.txt" 2>/dev/null || true
  sed -n '1,220p' "$RUN/art/package_dexopt.txt" 2>/dev/null || true
  echo
  echo "## storage pressure"
  sed -n '1,180p' "$RUN/storage/storage_plus_240s.txt" 2>/dev/null || sed -n '1,180p' "$RUN/storage/storage_plus_90s.txt" 2>/dev/null || true
  echo
  echo "## memory pressure"
  sed -n '1,220p' "$RUN/memory/memory_plus_240s.txt" 2>/dev/null || sed -n '1,220p' "$RUN/memory/memory_plus_90s.txt" 2>/dev/null || true
  echo
  echo "## latest archived files"
  for d in anr tombstones dropbox; do
    echo "-- $d --"
    ls -lt "$RUN/$d" 2>/dev/null | head -20 || true
  done
  echo
  echo "RESULT: PIXEL_BOOT_WATCH_RESULT_LOG_DONE rc=0"
} > "$TMP" 2>&1 || true
mv -f "$TMP" "$OUT"
cp -f "$OUT" "$LAST"
if [ "$MODE" = "action" ]; then
  cp -f "$OUT" "$ACTION"
fi
chown 1023:1023 "$OUT" "$LAST" "$ACTION" "$STATUS" 2>/dev/null || true
chmod 0660 "$OUT" "$LAST" "$ACTION" "$STATUS" 2>/dev/null || true
printf 'result_file=%s
last_file=%s
status_file=%s
' "$OUT" "$LAST" "$STATUS"
[ "$MODE" = "action" ] && printf 'action_file=%s
' "$ACTION"
grep -E '^archive_path=' "$RUN/result.marker" 2>/dev/null || true
echo "protected_names=yes"
echo "sortify_hold_expected=yes"
echo "RESULT: PIXEL_BOOT_WATCH_RESULT_LOG_DONE rc=0"
exit 0
