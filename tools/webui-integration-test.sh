#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
TMP=$(mktemp -d)
PID=""

cleanup() {
  if [[ -n "$PID" ]]; then
    kill "$PID" 2>/dev/null || true
    wait "$PID" 2>/dev/null || true
  fi
  rm -rf "$TMP"
}
trap cleanup EXIT

cp -a "$ROOT/src/magisk-module" "$TMP/module"
sed -i '1s|^#!/system/bin/sh$|#!/bin/sh|' "$TMP/module/bin/module-control"
mkdir -p "$TMP/state/runs" "$TMP/runtime" "$TMP/download"
chmod 0700 "$TMP/state" "$TMP/runtime" "$TMP/download"

archive="$TMP/download/boot-watch-20260802_231500_boot.tar.gz"
printf 'fixture archive\n' > "$archive"
cat > "$TMP/download/pixel_local__boot-watch-status.env" <<EOF_STATUS
pbw_result=PASS
pbw_run_id=20260802_231500_boot
pbw_mode=standard
pbw_version=0.2.10-webui-runtime-root-hotfix
pbw_archive_path=$archive
pbw_generated=2026-08-02T23:15:00+0200
pbw_file_name_too_long=absent
EOF_STATUS
cat > "$TMP/download/pixel_local__boot-watch-last-result.txt" <<'EOF_LAST'
pbw_result=PASS
pbw_run_id=20260802_231500_boot
token=SHOULD_NOT_LEAK
EOF_LAST
cat > "$TMP/download/pixel_local__boot-watch-action-last-result.txt" <<'EOF_ACTION'
RESULT: BOOT_WATCH_ACTION_DONE outcome=success workflow_exit_code=0
EOF_ACTION
cat > "$TMP/download/pixel_local__boot-watch-20260802_231500_boot-result.txt" <<'EOF_RUN'
pbw_run_id=20260802_231500_boot
pbw_result=PASS
pbw_mode=standard
pbw_version=0.2.10-webui-runtime-root-hotfix
EOF_RUN
cat > "$TMP/download/pixel_local__boot-watch-20260801_101500_boot-result.txt" <<'EOF_RUN_OLD'
pbw_run_id=20260801_101500_boot
pbw_result=WARN
pbw_mode=extended
pbw_version=0.2.9
EOF_RUN_OLD
cat > "$TMP/state/service-launch.log" <<'EOF_SERVICE'
service_start=2026-08-02T23:15:00+0200
module=boot-watch
version=0.2.11-webui-core-pilot.1
EOF_SERVICE

cd "$ROOT/webui-core"
go build -buildvcs=false -trimpath -o "$TMP/webui-server" ./server/cmd/webui-server

TOKEN=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
printf '%s\n' "$TOKEN" > "$TMP/runtime/bootstrap.token"
chmod 0600 "$TMP/runtime/bootstrap.token"

BOOT_WATCH_DOWNLOAD_DIR="$TMP/download" \
"$TMP/webui-server" \
  -listen 127.0.0.1:0 \
  -webroot "$TMP/module/webroot" \
  -control "$TMP/module/bin/module-control" \
  -module-dir "$TMP/module" \
  -state-dir "$TMP/state" \
  -runtime-dir "$TMP/runtime" \
  -token-file "$TMP/runtime/bootstrap.token" \
  -idle-timeout 1m \
  -session-ttl 1m \
  -job-timeout 1m \
  -max-jobs 1 \
  -state-file "$TMP/runtime/state.json" \
  -pid-file "$TMP/runtime/pid" \
  > "$TMP/server.log" 2>&1 &
PID=$!

for _ in $(seq 1 80); do
  [[ -s "$TMP/runtime/state.json" ]] && break
  sleep 0.1
done

PORT=$(sed -n 's/.*"port":[[:space:]]*\([0-9][0-9]*\).*/\1/p' "$TMP/runtime/state.json")
[[ "$PORT" =~ ^[0-9]+$ ]]
BASE="http://127.0.0.1:$PORT"
COOKIE="$TMP/cookies.txt"

curl -fsS "$BASE/api/v1/health" | grep -Fq '"ok":true'

cmdline=$(tr '\000' ' ' < "/proc/$PID/cmdline")
if grep -Fq "$TOKEN" <<< "$cmdline"; then
  echo "FAIL token_visible_in_process_argv"
  exit 1
fi

bootstrap_code=$(curl -sS -c "$COOKIE" -o /dev/null -w '%{http_code}' "$BASE/bootstrap?token=$TOKEN")
[[ "$bootstrap_code" == 303 ]]
[[ ! -e "$TMP/runtime/bootstrap.token" ]]
[[ "$(curl -sS -o /dev/null -w '%{http_code}' "$BASE/bootstrap?token=$TOKEN")" == 403 ]]

curl -fsS -b "$COOKIE" "$BASE/" | grep -Fq 'Root Module WebUI'
capabilities=$(curl -fsS -b "$COOKIE" "$BASE/api/v1/capabilities")
grep -Fq '"config":false' <<< "$capabilities"
grep -Fq '"actions":false' <<< "$capabilities"
grep -Fq '"jobs":false' <<< "$capabilities"
grep -Fq '"logs":true' <<< "$capabilities"
grep -Fq '"inventory":true' <<< "$capabilities"

status=$(curl -fsS -b "$COOKIE" "$BASE/api/v1/status")
grep -Fq '"id":"boot-watch"' <<< "$status"
grep -Fq '"label":"Result"' <<< "$status"
grep -Fq '"value":"PASS"' <<< "$status"
grep -Fq '"level":"good"' <<< "$status"
grep -Fq '"read_only_adapter":true' <<< "$status"

log=$(curl -fsS -b "$COOKIE" "$BASE/api/v1/log?lines=300")
grep -Fq 'STATUS ENV' <<< "$log"
grep -Fq 'pbw_result=PASS' <<< "$log"
if grep -Fq 'SHOULD_NOT_LEAK' <<< "$log"; then
  echo "FAIL secret_redaction"
  exit 1
fi

runs=$(curl -fsS -b "$COOKIE" "$BASE/api/v1/inventory?name=runs")
grep -Fq '20260802_231500_boot' <<< "$runs"
grep -Fq '20260801_101500_boot' <<< "$runs"
curl -fsS -b "$COOKIE" "$BASE/api/v1/inventory?name=evidence" | grep -Fq '"name":"Status ENV"'

for endpoint in config action jobs; do
  code=$(curl -sS -b "$COOKIE" -o /dev/null -w '%{http_code}' "$BASE/api/v1/$endpoint")
  [[ "$code" == 404 || "$code" == 405 ]]
done

[[ "$(curl -sS -o /dev/null -w '%{http_code}' "$BASE/api/v1/status")" == 401 ]]

kill "$PID"
wait "$PID" || true
PID=""

echo "RESULT: BOOT_WATCH_WEBUI_INTEGRATION_PASS"
