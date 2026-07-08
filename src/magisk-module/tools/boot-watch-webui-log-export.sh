#!/system/bin/sh
set -eu

MOD="${MODDIR:-/data/adb/modules/boot-watch}"
WEB="$MOD/webroot"
DL="/storage/emulated/0/Download"
LOG_JSON="$WEB/boot-watch-logs.json"
RUNS_JSON="$WEB/boot-watch-runs.json"
TMP_LOG="$LOG_JSON.tmp.$$"
TMP_RUNS="$RUNS_JSON.tmp.$$"
MAX_LINES="${BOOT_WATCH_WEBUI_LOG_LINES:-160}"
MAX_RUNS="${BOOT_WATCH_WEBUI_HISTORY_LIMIT:-10}"

json_escape() {
  sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

b64_stream() {
  if command -v base64 >/dev/null 2>&1; then
    base64 | tr -d '\n\r'
  elif command -v toybox >/dev/null 2>&1; then
    toybox base64 | tr -d '\n\r'
  else
    cat | tr -d '\000-\037\177'
  fi
}

safe_log_b64() {
  f="$1"
  if [ -f "$f" ]; then
    tail -n "$MAX_LINES" "$f" 2>/dev/null \
      | grep -Evi 'password|passphrase|psk|private_key|token|secret|cookie|authorization|bearer|apikey|api_key' \
      | b64_stream
  fi
}

file_obj() {
  label="$1"
  path="$2"
  exists="no"
  size="0"
  mtime=""
  if [ -f "$path" ]; then
    exists="yes"
    size="$(wc -c < "$path" 2>/dev/null || echo 0)"
    mtime="$(date -r "$path" '+%Y-%m-%dT%H:%M:%S%z' 2>/dev/null || true)"
  fi
  label_e="$(printf '%s' "$label" | json_escape)"
  path_e="$(printf '%s' "$path" | json_escape)"
  body_b64="$(safe_log_b64 "$path")"
  printf '{"label":"%s","path":"%s","exists":"%s","size":"%s","mtime":"%s","max_lines":"%s","encoding":"base64","content_b64":"%s"}' \
    "$label_e" "$path_e" "$exists" "$size" "$mtime" "$MAX_LINES" "$body_b64"
}

run_val() {
  key="$1"
  f="$2"
  grep -m1 "^$key=" "$f" 2>/dev/null | sed "s/^$key=//" | tr -d '"' | tr -d "'" | tr -d '\r' || true
}

mkdir -p "$WEB"

{
  printf '{\n'
  printf '  "bw_logs_generated":"%s",\n' "$(date '+%Y-%m-%dT%H:%M:%S%z')"
  printf '  "bw_logs_scope":"bootwatch_only",\n'
  printf '  "bw_logs_no_shell":"yes",\n'
  printf '  "bw_logs_no_host_run":"yes",\n'
  printf '  "bw_logs_no_route_change":"yes",\n'
  printf '  "bw_logs_no_cross_module":"yes",\n'
  printf '  "logs":[\n    '
  file_obj 'Status ENV' "$DL/pixel_local__boot-watch-status.env"
  printf ',\n    '
  file_obj 'Last Result' "$DL/pixel_local__boot-watch-last-result.txt"
  printf ',\n    '
  file_obj 'Action Result' "$DL/pixel_local__boot-watch-action-last-result.txt"
  printf ',\n    '
  file_obj 'Wrapper Log' "$MOD/action.webui-wrapper-last.log"
  printf ',\n    '
  file_obj 'Module Prop' "$MOD/module.prop"
  printf ',\n    '
  file_obj 'WebUI Status JSON' "$WEB/boot-watch-status.json"
  printf '\n  ]\n'
  printf '}\n'
} > "$TMP_LOG"
mv -f "$TMP_LOG" "$LOG_JSON"
chmod 0644 "$LOG_JSON"

{
  printf '{\n'
  printf '  "bw_runs_generated":"%s",\n' "$(date '+%Y-%m-%dT%H:%M:%S%z')"
  printf '  "bw_runs_scope":"bootwatch_only",\n'
  printf '  "runs":[\n'
  first=1
  find "$DL" -maxdepth 1 -type f -name 'pixel_local__boot-watch-*_boot-result.txt' 2>/dev/null \
    | sort -r \
    | head -n "$MAX_RUNS" \
    | while read -r f; do
        base="$(basename "$f")"
        run_id="$(printf '%s' "$base" | sed -n 's/^pixel_local__boot-watch-\(.*\)-result.txt$/\1/p')"
        result="$(run_val pbw_result "$f")"
        mode="$(run_val pbw_mode "$f")"
        size="$(wc -c < "$f" 2>/dev/null || echo 0)"
        mtime="$(date -r "$f" '+%Y-%m-%dT%H:%M:%S%z' 2>/dev/null || true)"
        path_e="$(printf '%s' "$f" | json_escape)"
        if [ "$first" = 1 ]; then
          printf '    '
          first=0
        else
          printf ',\n    '
        fi
        printf '{"run_id":"%s","result":"%s","mode":"%s","size":"%s","mtime":"%s","path":"%s"}' \
          "$run_id" "${result:-unknown}" "${mode:-unknown}" "$size" "$mtime" "$path_e"
      done
  printf '\n  ]\n'
  printf '}\n'
} > "$TMP_RUNS"
mv -f "$TMP_RUNS" "$RUNS_JSON"
chmod 0644 "$RUNS_JSON"

echo "boot_watch_webui_log_export=PASS"
echo "logs_json=$LOG_JSON"
echo "runs_json=$RUNS_JSON"
exit 0
