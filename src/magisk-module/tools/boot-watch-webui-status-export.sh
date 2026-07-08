#!/system/bin/sh
set -eu
MODDIR="${0%/tools/boot-watch-webui-status-export.sh}"
DL="/storage/emulated/0/Download"
OUT="$MODDIR/webroot/boot-watch-status.json"
TMP="$OUT.tmp.$$"
SRC=""
for c in \
  "$DL/pixel_local__boot-watch-status.env" \
  "$DL/pixel_local__boot-watch-status.env" \
  "$DL/pixel_local__boot-watch-action-last-result.txt" \
  "$DL/pixel_local__boot-watch-last-result.txt" \
  "$DL/pixel_local__boot-watch-action-last-result.txt" \
  "$DL/pixel_local__boot-watch-last-result.txt"
do
  if [ -f "$c" ]; then SRC="$c"; break; fi
done
json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g; s/\r/\\r/g'
}
{
  printf '{\n'
  first=1
  if [ -n "$SRC" ] && [ -f "$SRC" ]; then
    while IFS='=' read -r k v; do
      case "$k" in
        pbw_*|bw_*) ;;
        *) continue ;;
      esac
      case "$k" in *password*|*passphrase*|*token*|*secret*|*key*) continue ;; esac
      if [ "$first" -eq 0 ]; then printf ',\n'; fi
      first=0
      printf '  "%s": "%s"' "$(json_escape "$k")" "$(json_escape "$v")"
    done < "$SRC"
  fi
  if [ "$first" -eq 0 ]; then printf ',\n'; fi
  printf '  "bw_webui_generated": "%s",\n' "$(date +%Y-%m-%dT%H:%M:%S%z 2>/dev/null || date)"
  printf '  "bw_webui_source": "%s",\n' "${SRC:-none}"
  printf '  "bw_webui_scope": "bootwatch_only",\n'
  printf '  "bw_webui_no_shell": "yes",\n'
  printf '  "bw_webui_no_host_run": "yes",\n'
  printf '  "bw_webui_no_route_change": "yes"\n'
  printf '}\n'
} > "$TMP"
mv -f "$TMP" "$OUT"
chmod 0644 "$OUT" 2>/dev/null || true
