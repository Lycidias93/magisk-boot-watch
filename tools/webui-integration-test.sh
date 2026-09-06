#!/usr/bin/env bash
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
TMP=$(mktemp -d)
PID=""
cleanup(){ if [[ -n "$PID" ]]; then kill "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true; fi; rm -rf "$TMP"; }
trap cleanup EXIT
cp -a "$ROOT/src/magisk-module" "$TMP/module"
sed -i '1s|^#!/system/bin/sh$|#!/bin/sh|' "$TMP/module/bin/module-control"
mkdir -p "$TMP/state/runs" "$TMP/runtime" "$TMP/download"
chmod 0700 "$TMP/state" "$TMP/runtime" "$TMP/download"
archive="$TMP/download/boot-watch-20260906_160000_boot.tar.gz"
printf 'fixture archive\n' > "$archive"
cat > "$TMP/download/pixel_local__boot-watch-status.env" <<EOF
pbw_result=PASS
pbw_run_id=20260906_160000_boot
pbw_mode=extended
pbw_version=0.2.11-vnext.1
pbw_archive_path=$archive
pbw_generated=2026-09-06T16:04:00+0200
pbw_file_name_too_long=absent
pbw_zygisk_support_profile=extended
pbw_zygisk_stack_overall=ready
pbw_lsposed_present=yes
pbw_lsposed_module_aliases=zygisk_lsposed
pbw_lspd_service=running
pbw_lspd_log_files=4
pbw_lspd_rotated_log_files=3
pbw_lspd_config_files=2
pbw_vector_present=yes
pbw_vector_module_aliases=zygisk_vector
pbw_magisk_denylist_state=collected
pbw_magisk_denylist_entries=12
EOF
cat > "$TMP/download/pixel_local__boot-watch-last-result.txt" <<EOF
pbw_result=PASS
token=SHOULD_NOT_LEAK
EOF
cat > "$TMP/download/pixel_local__boot-watch-action-last-result.txt" <<EOF
RESULT: BOOT_WATCH_ACTION_DONE outcome=success workflow_exit_code=0
EOF
cat > "$TMP/download/pixel_local__boot-watch-20260906_160000_boot-result.txt" <<EOF
pbw_run_id=20260906_160000_boot
pbw_result=PASS
pbw_mode=extended
pbw_version=0.2.11-vnext.1
EOF
printf 'service_start=fixture\nmodule=boot-watch\nversion=0.2.11-vnext.1\n' > "$TMP/state/service-launch.log"
(cd "$ROOT/webui-core" && go build -buildvcs=false -trimpath -o "$TMP/webui-server" ./server/cmd/webui-server)
TOKEN=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
printf "%s\n" "$TOKEN" > "$TMP/runtime/bootstrap.token"
chmod 0600 "$TMP/runtime/bootstrap.token"
BOOT_WATCH_DOWNLOAD_DIR="$TMP/download" "$TMP/webui-server" -listen 127.0.0.1:0 -webroot "$TMP/module/webroot" -control "$TMP/module/bin/module-control" -module-dir "$TMP/module" -state-dir "$TMP/state" -runtime-dir "$TMP/runtime" -token-file "$TMP/runtime/bootstrap.token" -idle-timeout 1m -session-ttl 1m -job-timeout 1m -max-jobs 1 -state-file "$TMP/runtime/state.json" -pid-file "$TMP/runtime/pid" > "$TMP/server.log" 2>&1 &
PID=$!
for _ in $(seq 1 80); do [[ -s "$TMP/runtime/state.json" ]] && break; sleep 0.1; done
PORT=$(sed -n 's/.*"port":[[:space:]]*\([0-9][0-9]*\).*/\1/p' "$TMP/runtime/state.json")
[[ "$PORT" =~ ^[0-9]+$ ]]
BASE="http://127.0.0.1:$PORT"
COOKIE="$TMP/cookies.txt"
curl -fsS "$BASE/api/v1/health" | grep -Fq '"ok":true'
cmdline=$(tr "\000" " " < "/proc/$PID/cmdline")
! grep -Fq "$TOKEN" <<< "$cmdline"
[[ "$(curl -sS -c "$COOKIE" -o /dev/null -w "%{http_code}" "$BASE/bootstrap?token=$TOKEN")" == 303 ]]
[[ ! -e "$TMP/runtime/bootstrap.token" ]]
[[ "$(curl -sS -o /dev/null -w "%{http_code}" "$BASE/bootstrap?token=$TOKEN")" == 403 ]]
for asset in app.css race-guard.css observability.css embedded-host-bootstrap.js race-guard.js observability.js mobile-input-viewport.js app.js v03.js v04.js; do curl -fsS -b "$COOKIE" "$BASE/$asset" >/dev/null; done
caps=$(curl -fsS -b "$COOKIE" "$BASE/api/v1/capabilities")
grep -Fq '"config":false' <<<"$caps"
grep -Fq '"actions":false' <<<"$caps"
grep -Fq '"jobs":false' <<<"$caps"
grep -Fq '"logs":true' <<<"$caps"
grep -Fq '"inventory":true' <<<"$caps"
status=$(curl -fsS -b "$COOKIE" "$BASE/api/v1/status")
grep -Fq '"label":"Zygisk stack"' <<<"$status"
grep -Fq '"value":"ready"' <<<"$status"
grep -Fq '"label":"LSPosed"' <<<"$status"
grep -Fq '"label":"Vector"' <<<"$status"
log=$(curl -fsS -b "$COOKIE" "$BASE/api/v1/log?lines=300")
grep -Fq 'pbw_result=PASS' <<<"$log"
! grep -Fq 'SHOULD_NOT_LEAK' <<<"$log"
runs=$(curl -fsS -b "$COOKIE" "$BASE/api/v1/inventory?name=runs")
grep -Fq '20260906_160000_boot' <<<"$runs"
zyg=$(curl -fsS -b "$COOKIE" "$BASE/api/v1/inventory?name=zygisk_stack")
grep -Fq 'LSPosed / lspd' <<<"$zyg"
grep -Fq 'config contents not collected' <<<"$zyg"
curl -fsS -b "$COOKIE" "$BASE/api/v1/inventory?name=evidence" | grep -Fq '"name":"Status ENV"'
for endpoint in config action jobs; do code=$(curl -sS -b "$COOKIE" -o /dev/null -w "%{http_code}" "$BASE/api/v1/$endpoint"); [[ "$code" == 404 || "$code" == 405 ]]; done
[[ "$(curl -sS -o /dev/null -w "%{http_code}" "$BASE/api/v1/status")" == 401 ]]
origin_code=$(curl -sS -b "$COOKIE" -H "Origin: http://evil.invalid" -H "X-WebUI-Request: 1" -H "Content-Type: application/json" -d '{"name":"x"}' -o /dev/null -w "%{http_code}" "$BASE/api/v1/jobs")
[[ "$origin_code" == 403 ]]
echo asset_http=PASS
echo status_lsposed_vector=PASS
echo disabled_mutation_surfaces=PASS
echo origin_rejected=PASS
echo 'RESULT: BOOT_WATCH_WEBUI_INTEGRATION_PASS'
