#!/system/bin/sh
# Boot Watch Collector 0.2.11-vnext.1 comprehensive one-shot collector
# Bounded, local-only, no daemon after completion.

MOD="/data/adb/modules/boot-watch"
RT="/data/adb/boot-watch"
DL="/storage/emulated/0/Download"
VERSION="0.2.11-vnext.1"
VERSION_CODE="36"
PROFILE="${PBW_PROFILE:-standard}"
MAX_SECONDS="${PBW_MAX_SECONDS:-360}"
MAX_FILES="${PBW_MAX_FILES:-80}"
MAX_LINES="${PBW_MAX_LINES:-1500}"
RUN_ID="$(date +%Y%m%d_%H%M%S)_boot"
RUN="$RT/runs/$RUN_ID"
LOG="$RUN/boot-watch.log"
STAGES="$RUN/stages.txt"
SUMMARY="$RUN/summary.txt"
CLASSIFY="$RUN/classification.txt"
REDFLAGS="$RUN/red_flags.txt"
REDFLAGS_SUMMARY="$RUN/red_flags_summary.txt"
MARKER="$RUN/result.marker"
ARCHIVE="$DL/boot-watch-$RUN_ID.tar.gz"
RESULT_TXT="$DL/pixel_local__boot-watch-$RUN_ID-result.txt"
LAST_TXT="$DL/pixel_local__boot-watch-last-result.txt"
ACTION_TXT="$DL/pixel_local__boot-watch-action-last-result.txt"
STATUS_ENV="$DL/pixel_local__boot-watch-status.env"
START_EPOCH="$(date +%s)"
LOGD_STARTED_BY_PBW=0
LOGD_WAS_STOPPED=0

mkdir -p "$RUN" "$DL" "$RT/runs"
: > "$LOG"
: > "$STAGES"
: > "$SUMMARY"
: > "$CLASSIFY"
: > "$REDFLAGS"
: > "$REDFLAGS_SUMMARY"

log() { echo "$*" | tee -a "$LOG" >/dev/null; }
stage() { echo "stage=$1" >> "$STAGES"; echo "stage_time=$(date +%Y-%m-%dT%H:%M:%S%z)" >> "$STAGES"; log "STAGE $1"; }

run_file() {
  out="$1"; shift
  mkdir -p "$(dirname "$out")"
  {
    echo "cmd=$*"
    echo "time=$(date +%Y-%m-%dT%H:%M:%S%z)"
    echo "---"
    if command -v timeout >/dev/null 2>&1; then
      timeout 18 /system/bin/sh -c "$*" 2>&1 || true
    else
      /system/bin/sh -c "$*" 2>&1 || true
    fi
  } > "$out" 2>&1 || true
}

append_cmd() {
  out="$1"; shift
  {
    echo
    echo "### $*"
    if command -v timeout >/dev/null 2>&1; then
      timeout 18 /system/bin/sh -c "$*" 2>&1 || true
    else
      /system/bin/sh -c "$*" 2>&1 || true
    fi
  } >> "$out" 2>&1 || true
}

tail_file() {
  src="$1"; dst="$2"; lines="${3:-400}"
  mkdir -p "$(dirname "$dst")"
  if [ -f "$src" ]; then
    tail -n "$lines" "$src" > "$dst" 2>&1 || true
  fi
}

copy_latest_files() {
  src="$1"; dst="$2"; limit="${3:-40}"
  mkdir -p "$dst"
  if [ -d "$src" ]; then
    count=0
    for name in $(ls -1t "$src" 2>/dev/null | head -n "$limit"); do
      [ -e "$src/$name" ] || continue
      [ -f "$src/$name" ] || continue
      cp -p "$src/$name" "$dst/$name" 2>/dev/null || true
      count=$((count + 1))
    done
    log "copied=$count from=$src into=$dst"
  else
    log "missing_src=$src"
  fi
}

copy_tree_flat_safe() {
  src="$1"; dst="$2"; pattern="$3"; limit="${4:-40}"
  mkdir -p "$dst"
  count=0
  [ -d "$src" ] || { log "missing_src=$src"; return 0; }
  find "$src" -type f \( $pattern \) \
    ! -path "$RT/runs/*" \
    ! -path "$RT/backups/*" \
    ! -path "$MOD/*" \
    2>/dev/null | head -n "$limit" | while IFS= read -r p; do
      base="$(basename "$p" | tr -c 'A-Za-z0-9._-' '_' | cut -c1-72)"
      h="$(printf '%s' "$p" | sha256sum 2>/dev/null | awk '{print $1}' | cut -c1-12)"
      [ -n "$h" ] || h="nohash"
      cp -p "$p" "$dst/${h}_${base}.txt" 2>/dev/null || true
    done
  count="$(find "$dst" -type f 2>/dev/null | wc -l | tr -d ' ')"
  log "safe_flat_copied=$count from=$src into=$dst"
}

read_text_or_gzip() {
  f="$1"
  case "$f" in
    *.gz) gzip -dc "$f" 2>/dev/null || true ;;
    *.pb|*.dat) return 0 ;;
    *) cat "$f" 2>/dev/null || true ;;
  esac
}

classify_file() {
  f="$1"
  echo "--- $f ---" >> "$CLASSIFY"
  read_text_or_gzip "$f" | grep -E '^(Process:|Package:|Subject:|Cmd line:|Cmdline:|pid:|signal |Abort message:|Short Msg:|Long Msg:)|ANR in|FATAL EXCEPTION|Watchdog|dex2oat|safetycore|Permission denied|system_server|zygisk|lspd|XSupport|Lucky|AshReXcue|bootloop|rescue|lowmemorykiller|LMKD' 2>/dev/null | head -n 80 >> "$CLASSIFY" || true
}

classify_dir() {
  dir="$1"
  [ -d "$dir" ] || return 0
  find "$dir" -maxdepth 1 -type f 2>/dev/null | head -n "$MAX_FILES" | while IFS= read -r f; do
    classify_file "$f"
  done
}

log "Boot Watch Collector v$VERSION comprehensive collector"
log "run_id=$RUN_ID"
log "run_dir=$RUN"
log "profile=$PROFILE"
log "time=$(date +%Y-%m-%dT%H:%M:%S%z)"
log "uid=$(id 2>/dev/null || true)"
log "kernel=$(uname -a 2>/dev/null || true)"

stage "wait_boot_completed"
elapsed=0
while [ "$elapsed" -lt 180 ]; do
  bc="$(getprop sys.boot_completed 2>/dev/null || true)"
  dbc="$(getprop dev.bootcomplete 2>/dev/null || true)"
  log "wait elapsed=${elapsed}s sys.boot_completed=$bc dev.bootcomplete=$dbc"
  [ "$bc" = "1" ] && [ "$dbc" = "1" ] && break
  sleep 5
  elapsed=$((elapsed + 5))
done

logd_before="$(getprop init.svc.logd 2>/dev/null || true)"
log "logd_state_before=$logd_before"
if [ "$logd_before" = "stopped" ]; then
  LOGD_WAS_STOPPED=1
  start logd 2>/dev/null || true
  sleep 1
  after="$(getprop init.svc.logd 2>/dev/null || true)"
  log "ACTION start logd temporarily for boot watch capture"
  log "logd_state_after_start=$after"
  [ "$after" = "running" ] && LOGD_STARTED_BY_PBW=1
fi

stage "static_core"
mkdir -p "$RUN/boot" "$RUN/magisk" "$RUN/modules" "$RUN/zygisk_stack" "$RUN/zygisk" "$RUN/lsposed" "$RUN/art" "$RUN/binder" "$RUN/audio_safe" "$RUN/rescue" "$RUN/ashlooper_intervention" "$RUN/pstore" "$RUN/dumpsys" "$RUN/module_runtime" "$RUN/dispatcher" "$RUN/thermal" "$RUN/power" "$RUN/network" "$RUN/service_d" "$RUN/storage" "$RUN/memory" "$RUN/kernel" "$RUN/logcat" "$RUN/logcat_split" "$RUN/anr" "$RUN/tombstones" "$RUN/dropbox"

run_file "$RUN/boot/core.txt" "date -Is; uptime; id; getenforce 2>/dev/null || true; getprop sys.boot_completed; getprop dev.bootcomplete; getprop ro.boot.bootreason; getprop ro.boot.verifiedbootstate; getprop ro.boot.flash.locked; getprop ro.boot.vbmeta.device_state; getprop ro.boot.slot_suffix; getprop ro.build.fingerprint; getprop ro.build.version.release; getprop ro.build.version.sdk"
run_file "$RUN/boot/props_focus.txt" "getprop | grep -Ei 'boot|zygisk|magisk|dex2oat|art|thermal|wifi|radio|telephony|net.dns|lmk|memory|safety|debug' | head -n $MAX_LINES"
run_file "$RUN/magisk/version.txt" "magisk -V 2>/dev/null || true; magisk -v 2>/dev/null || true"
tail_file "/data/adb/magisk.log" "$RUN/magisk/magisk.log.txt" 600
tail_file "/cache/magisk.log" "$RUN/magisk/cache_magisk.log.txt" 600

{
  echo "module_matrix_time=$(date -Is)"
  for d in /data/adb/modules/*; do
    [ -d "$d" ] || continue
    echo "--- $(basename "$d") ---"
    grep -E '^(id|name|version|versionCode|description)=' "$d/module.prop" 2>/dev/null || true
    [ -e "$d/disable" ] && echo "disable=present" || echo "disable=absent"
    [ -e "$d/remove" ] && echo "remove=present" || echo "remove=absent"
    [ -f "$d/service.sh" ] && echo "service.sh=present" || echo "service.sh=absent"
    [ -f "$d/post-fs-data.sh" ] && echo "post-fs-data.sh=present" || echo "post-fs-data.sh=absent"
    [ -f "$d/action.sh" ] && echo "action.sh=present" || echo "action.sh=absent"
  done
} > "$RUN/modules/module_matrix.txt" 2>&1 || true

run_file "$RUN/service_d/listing.txt" "ls -la /data/adb/service.d /data/adb/post-fs-data.d 2>/dev/null || true"
run_file "$RUN/service_d/syntax_quick.txt" "for f in /data/adb/service.d/*.sh /data/adb/post-fs-data.d/*.sh; do test -f \$f || continue; echo --- \$f ---; ls -l \$f; head -1 \$f; /system/bin/sh -n \$f && echo syntax=ok || echo syntax=fail; done"

stage "zygisk_art_lsposed"
run_file "$RUN/zygisk/mountinfo_focus.txt" "cat /proc/self/mountinfo 2>/dev/null | grep -Ei 'zygisk|vector|dex2oat|app_process|libart|lsposed|rezygisk|zygisknext|shamiko|tricky|playintegrity|safetycore' || true"
run_file "$RUN/zygisk/dex2oat_overlay.txt" "ls -lZ /apex/com.android.art/bin/dex2oat64 2>/dev/null || ls -l /apex/com.android.art/bin/dex2oat64 2>/dev/null || true; find /data/adb/modules -path '*/bin/dex2oat64' -type f -exec ls -lZ {} \\; 2>/dev/null || true; cat /proc/self/mountinfo 2>/dev/null | grep -Ei 'dex2oat64|safetycore' || true"
run_file "$RUN/lsposed/lspd_status.txt" "getprop init.svc.lspd 2>/dev/null || true; pidof lspd 2>/dev/null || true; find /data/adb/lspd/log -maxdepth 1 -type f 2>/dev/null | sort | tail -20"
copy_latest_files "/data/adb/lspd/log" "$RUN/lsposed/lspd_log" 8
run_file "$RUN/art/package_dexopt.txt" "cmd package bg-dexopt-job status 2>&1 || true; cmd package list staged-sessions 2>&1 || true; cmd package get-stagedsessions 2>&1 || true; pm path com.google.android.safetycore 2>&1 || true; pm path com.google.android.gms 2>&1 || true; pm path com.google.android.gsf 2>&1 || true; pm list packages | grep -E 'safetycore|gms|gsf|vending' || true"

stage "binder_audio_runtime"
run_file "$RUN/binder/service_health.txt" "service check settings 2>&1 || true; service check package 2>&1 || true; service check activity 2>&1 || true; service check power 2>&1 || true; service check dropbox 2>&1 || true; service list 2>/dev/null | grep -E 'settings|package|activity|power|dropbox|thermal' || true; cmd settings get global safe_media_volume_enabled 2>&1 || true; pm path com.google.android.safetycore 2>&1 || true"
run_file "$RUN/audio_safe/status.txt" "echo service_file=\$(test -f /data/adb/service.d/99-audio-safe-volume.sh && echo present || echo absent); settings get global audio_safe_csd_next_warning 2>&1 || true; settings get global safe_media_volume_enabled 2>&1 || true; settings get global audio_safe_volume_state 2>&1 || true; settings get global audio_safe_csd_current_value 2>&1 || true; settings get global audio_safe_csd_dose_records 2>&1 || true"
collect_ashlooper_health() {
  out="$RUN/rescue/ashlooper_health.txt"
  logs="$RUN/rescue/ashlooper_logs"
  mkdir -p "$logs"
  {
    echo "ashlooper_health_time=$(date +%Y-%m-%dT%H:%M:%S%z)"
    if [ -d /data/adb/modules/AshLooper ]; then
      echo "ashlooper_present=yes"
      grep -E '^(id|name|version|versionCode|description)=' /data/adb/modules/AshLooper/module.prop 2>/dev/null || true
      [ -e /data/adb/modules/AshLooper/disable ] && echo "ashlooper_disable=present" || echo "ashlooper_disable=absent"
      [ -e /data/adb/modules/AshLooper/remove ] && echo "ashlooper_remove=present" || echo "ashlooper_remove=absent"
      [ -f /data/adb/modules/AshLooper/service.sh ] && echo "ashlooper_service=present" || echo "ashlooper_service=absent"
      [ -f /data/adb/modules/AshLooper/post-fs-data.sh ] && echo "ashlooper_post_fs_data=present" || echo "ashlooper_post_fs_data=absent"
      [ -f /data/adb/modules/AshLooper/action.sh ] && echo "ashlooper_action=present" || echo "ashlooper_action=absent"
      for f in /data/adb/modules/AshLooper/settings.prop /data/adb/modules/AshLooper/config.prop /data/adb/modules/AshLooper/state.prop /data/adb/modules/AshLooper/status.prop; do
        if [ -f "$f" ]; then
          echo "--- $f ---"
          sed -n '1,160p' "$f" 2>/dev/null || true
        fi
      done
    else
      echo "ashlooper_present=no"
    fi
    echo "--- bounded candidates ---"
    find /data/adb/modules/AshLooper /data/adb/AshLooper /data/adb/ashlooper /data/adb/AshReXcue /data/adb/ashrexcue -type f 2>/dev/null | grep -Ei 'ash|rescue|boot|loop|safe|log|status|state|prop|txt|json' | head -80 || true
  } > "$out" 2>&1 || true
  find /data/adb/modules/AshLooper /data/adb/AshLooper /data/adb/ashlooper /data/adb/AshReXcue /data/adb/ashrexcue -type f 2>/dev/null | grep -Ei '\.(log|txt|prop|json)$|ash|rescue|boot|loop|safe|status|state' | head -10 | while IFS= read -r f; do
    base="$(basename "$f" | tr -c 'A-Za-z0-9._-' '_' | cut -c1-72)"
    h="$(printf '%s' "$f" | sha256sum 2>/dev/null | awk '{print $1}' | cut -c1-12)"
    [ -n "$h" ] || h="nohash"
    {
      echo "source=$f"
      echo "time=$(date +%Y-%m-%dT%H:%M:%S%z)"
      echo "---"
      tail -n 200 "$f" 2>/dev/null || sed -n '1,200p' "$f" 2>/dev/null || true
    } > "$logs/${h}_${base}.txt" 2>&1 || true
  done
}


collect_ashlooper_intervention() {
  stage "ashlooper_intervention"
  mkdir -p "$RUN/ashlooper_intervention"
  summary="$RUN/ashlooper_intervention/summary.txt"
  disabled="$RUN/ashlooper_intervention/disabled_modules.txt"
  state="$RUN/ashlooper_intervention/ashlooper_state.txt"
  disabled_candidates=""
  disabled_count=0
  reason_candidate="unknown"

  {
    echo "disabled_modules_time=$(date +%Y-%m-%dT%H:%M:%S%z)"
    for d in /data/adb/modules/*; do
      [ -d "$d" ] || continue
      mid="$(basename "$d")"
      disable_state=absent
      remove_state=absent
      disable_mtime=""
      remove_mtime=""
      [ -e "$d/disable" ] && disable_state=present && disable_mtime="$(date -r "$d/disable" +%Y-%m-%dT%H:%M:%S%z 2>/dev/null || true)"
      [ -e "$d/remove" ] && remove_state=present && remove_mtime="$(date -r "$d/remove" +%Y-%m-%dT%H:%M:%S%z 2>/dev/null || true)"
      if [ "$disable_state" = "present" ] || [ "$remove_state" = "present" ]; then
        echo "--- module=$mid ---"
        grep -E '^(id|name|version|versionCode|description)=' "$d/module.prop" 2>/dev/null || true
        echo "disable=$disable_state"
        echo "disable_mtime=$disable_mtime"
        echo "remove=$remove_state"
        echo "remove_mtime=$remove_mtime"
      fi
    done
  } > "$disabled" 2>&1 || true

  {
    echo "ashlooper_state_time=$(date +%Y-%m-%dT%H:%M:%S%z)"
    for f in \
      /data/adb/modules/AshLooper/settings.prop \
      /data/adb/modules/AshLooper/config.prop \
      /data/adb/modules/AshLooper/state.prop \
      /data/adb/modules/AshLooper/status.prop \
      /data/adb/ashlooper/tmp_modules.json \
      /data/adb/ashlooper/module.json; do
      if [ -f "$f" ]; then
        echo "--- $f ---"
        sed -n '1,220p' "$f" 2>/dev/null || true
      fi
    done
  } > "$state" 2>&1 || true

  for d in /data/adb/modules/*; do
    [ -d "$d" ] || continue
    mid="$(basename "$d")"
    [ "$mid" = "AshLooper" ] && continue
    if [ -e "$d/disable" ] || [ -e "$d/remove" ]; then
      disabled_count=$((disabled_count + 1))
      if [ -z "$disabled_candidates" ]; then
        disabled_candidates="$mid"
      else
        disabled_candidates="$disabled_candidates,$mid"
      fi
    fi
  done

  case ",$disabled_candidates," in
    *,ptune,*) reason_candidate="ptune" ;;
    *,pTune,*) reason_candidate="ptune" ;;
  esac

  ash_present="no"
  [ -d /data/adb/modules/AshLooper ] && ash_present="yes"

  intervention_possible="no"
  logs_missing="no"
  if [ "$ash_present" = "yes" ] && [ "$disabled_count" -gt 0 ]; then
    intervention_possible="yes"
    logs_missing="yes"
  fi
  [ -n "$disabled_candidates" ] || disabled_candidates="none"
  [ "$disabled_candidates" = "none" ] && reason_candidate="none"

  {
    echo "ashlooper_intervention_time=$(date +%Y-%m-%dT%H:%M:%S%z)"
    echo "ashlooper_present=$ash_present"
    echo "ashlooper_intervention_possible=$intervention_possible"
    echo "ashlooper_disabled_candidates=$disabled_candidates"
    echo "ashlooper_disabled_count=$disabled_count"
    echo "ashlooper_reason_candidate=$reason_candidate"
    echo "module_logs_missing_because_disabled=$logs_missing"
    echo "disabled_modules_file=$disabled"
    echo "ashlooper_state_file=$state"
    echo "note=module disabled before collection is classified as evidence limitation unless AshLooper state explicitly proves cause"
  } > "$summary" 2>&1 || true
}

collect_split_logcat() {
  label="$1"
  dst="$RUN/logcat_split/$label"
  mkdir -p "$dst"
  for buf in main system crash events kernel radio; do
    run_file "$dst/logcat_${buf}.txt" "logcat -b $buf -d -t 800 2>/dev/null || true"
  done
}

collect_pstore() {
  stage "pstore"
  mkdir -p "$RUN/pstore/files"
  {
    echo "pstore_time=$(date +%Y-%m-%dT%H:%M:%S%z)"
    if [ -d /sys/fs/pstore ]; then
      echo "pstore_present=yes"
      ls -la /sys/fs/pstore 2>/dev/null || true
    else
      echo "pstore_present=no"
    fi
  } > "$RUN/pstore/summary.txt" 2>&1 || true
  if [ -d /sys/fs/pstore ]; then
    find /sys/fs/pstore -maxdepth 1 -type f 2>/dev/null | head -20 | while IFS= read -r f; do
      base="$(basename "$f" | tr -c 'A-Za-z0-9._-' '_' | cut -c1-72)"
      h="$(printf '%s' "$f" | sha256sum 2>/dev/null | awk '{print $1}' | cut -c1-12)"
      [ -n "$h" ] || h="nohash"
      {
        echo "source=$f"
        echo "time=$(date +%Y-%m-%dT%H:%M:%S%z)"
        echo "---"
        sed -n '1,260p' "$f" 2>/dev/null || true
      } > "$RUN/pstore/files/${h}_${base}.txt" 2>&1 || true
    done
  fi
}

collect_focused_dumpsys() {
  stage "focused_dumpsys"
  mkdir -p "$RUN/dumpsys"
  run_file "$RUN/dumpsys/activity_processes.txt" "dumpsys activity processes 2>/dev/null | head -300 || true"
  run_file "$RUN/dumpsys/activity_services.txt" "dumpsys activity services 2>/dev/null | head -300 || true"
  run_file "$RUN/dumpsys/activity_providers.txt" "dumpsys activity providers 2>/dev/null | head -260 || true"
  run_file "$RUN/dumpsys/meminfo_summary.txt" "dumpsys meminfo 2>/dev/null | head -320 || true"
  run_file "$RUN/dumpsys/jobscheduler.txt" "dumpsys jobscheduler 2>/dev/null | head -320 || true"
  run_file "$RUN/dumpsys/alarm.txt" "dumpsys alarm 2>/dev/null | head -300 || true"
  run_file "$RUN/dumpsys/deviceidle.txt" "dumpsys deviceidle 2>/dev/null | head -220 || true"
  run_file "$RUN/dumpsys/window.txt" "dumpsys window 2>/dev/null | head -260 || true"
  run_file "$RUN/dumpsys/input.txt" "dumpsys input 2>/dev/null | head -240 || true"
  run_file "$RUN/dumpsys/surfaceflinger.txt" "dumpsys SurfaceFlinger 2>/dev/null | head -260 || true"
}

collect_diagnostics_vnext_bundle() {
  collect_ashlooper_intervention
  collect_pstore
  collect_focused_dumpsys
}

collect_module_runtime_logs() {
  stage "module_runtime_logs"
  mkdir -p "$RUN/module_runtime" "$RUN/module_runtime/frosty" "$RUN/module_runtime/lsposed_log_old"
  {
    echo "module_runtime_time=$(date +%Y-%m-%dT%H:%M:%S%z)"
    echo "profile=$PROFILE"
    echo "pbw_collect_module_logs=${PBW_COLLECT_MODULE_LOGS:-0}"
    for m in Frosty rezygisk treat_wheel zygisk_vector zygisk-detach tricky_store playintegrityfix anti_safetycore rvmm-zygisk-mount dirtysepbypass sortify ssh_drop_dispatcher; do
      d="/data/adb/modules/$m"
      [ -d "$d" ] || continue
      echo "--- module=$m ---"
      grep -E '^(id|name|version|versionCode|description)=' "$d/module.prop" 2>/dev/null || true
      [ -e "$d/disable" ] && echo "disable=present" || echo "disable=absent"
      [ -e "$d/remove" ] && echo "remove=present" || echo "remove=absent"
      if [ -d "$d/logs" ]; then
        echo "logs_dir=present"
        find "$d/logs" -maxdepth 1 -type f 2>/dev/null | while IFS= read -r f; do
          s="$(wc -c < "$f" 2>/dev/null || echo 0)"
          mt="$(date -r "$f" +%Y-%m-%dT%H:%M:%S%z 2>/dev/null || true)"
          echo "log_candidate size=$s mtime=$mt path=$f"
        done
      else
        echo "logs_dir=absent"
      fi
    done
  } > "$RUN/module_runtime/known_modules.txt" 2>&1 || true

  module_logs_enabled=0
  case "$PROFILE" in
    extended|debug) module_logs_enabled=1 ;;
  esac
  [ "${PBW_COLLECT_MODULE_LOGS:-0}" = "1" ] && module_logs_enabled=1
  echo "module_runtime_logs_enabled=$module_logs_enabled" >> "$RUN/module_runtime/known_modules.txt"

  if [ "$module_logs_enabled" != "1" ]; then
    log "module_runtime_logs_content_skipped profile=$PROFILE pbw_collect_module_logs=${PBW_COLLECT_MODULE_LOGS:-0}"
    return 0
  fi

  for f in kernel_tweaks.log ram.log services.log; do
    tail_file "/data/adb/modules/Frosty/logs/$f" "$RUN/module_runtime/frosty/$f.txt" 600
  done

  for f in kmsg.log props.txt; do
    tail_file "/data/adb/lspd/log.old/$f" "$RUN/module_runtime/lsposed_log_old/$f.txt" 600
  done
  old_modules="$(find /data/adb/lspd/log.old -maxdepth 1 -type f -name 'modules_*.log' 2>/dev/null | sort | tail -1)"
  old_verbose="$(find /data/adb/lspd/log.old -maxdepth 1 -type f -name 'verbose_*.log' 2>/dev/null | sort | tail -1)"
  [ -n "$old_modules" ] && tail_file "$old_modules" "$RUN/module_runtime/lsposed_log_old/$(basename "$old_modules").txt" 600
  [ -n "$old_verbose" ] && tail_file "$old_verbose" "$RUN/module_runtime/lsposed_log_old/$(basename "$old_verbose").txt" 600
}
collect_zygisk_stack_support() {
  stage "zygisk_stack_support"
  mkdir -p "$RUN/zygisk_stack"
  summary="$RUN/zygisk_stack/summary.txt"
  inventory="$RUN/zygisk_stack/inventory.txt"
  denylist="$RUN/zygisk_stack/magisk_denylist.txt"

  append_alias() {
    current="$1"
    value="$2"
    if [ "$current" = "none" ]; then printf "%s" "$value"; else printf "%s,%s" "$current" "$value"; fi
  }

  lsposed_aliases=none
  vector_aliases=none
  [ -d /data/adb/modules/zygisk_lsposed ] && lsposed_aliases="$(append_alias "$lsposed_aliases" zygisk_lsposed)"
  [ -d /data/adb/modules/LSPosed ] && lsposed_aliases="$(append_alias "$lsposed_aliases" LSPosed)"
  [ -d /data/adb/modules/vector ] && vector_aliases="$(append_alias "$vector_aliases" vector)"
  [ -d /data/adb/modules/zygisk_vector ] && vector_aliases="$(append_alias "$vector_aliases" zygisk_vector)"

  lspd_runtime_present=no
  [ -d /data/adb/lspd ] && lspd_runtime_present=yes
  lspd_service="$(getprop init.svc.lspd 2>/dev/null || true)"
  [ -n "$lspd_service" ] || lspd_service=unknown
  lspd_pid="$(pidof lspd 2>/dev/null | head -c 128 || true)"
  [ -n "$lspd_pid" ] || lspd_pid=none
  lspd_log_files="$(find /data/adb/lspd/log -maxdepth 1 -type f 2>/dev/null | wc -l | tr -d " ")"
  lspd_rotated_log_files="$(find /data/adb/lspd/log.old -maxdepth 1 -type f 2>/dev/null | wc -l | tr -d " ")"
  lspd_config_files="$(find /data/adb/lspd/config -maxdepth 1 -type f 2>/dev/null | wc -l | tr -d " ")"
  [ -n "$lspd_log_files" ] || lspd_log_files=0
  [ -n "$lspd_rotated_log_files" ] || lspd_rotated_log_files=0
  [ -n "$lspd_config_files" ] || lspd_config_files=0

  lsposed_present=no
  if [ "$lspd_runtime_present" = yes ] || [ "$lsposed_aliases" != none ]; then lsposed_present=yes; fi
  vector_present=no
  [ "$vector_aliases" != none ] && vector_present=yes

  zygisk_stack_overall=not_detected
  if [ "$lsposed_present" = yes ]; then
    if [ "$lspd_service" = running ] || [ "$lspd_pid" != none ]; then zygisk_stack_overall=ready; else zygisk_stack_overall=attention; fi
  elif [ "$vector_present" = yes ]; then
    zygisk_stack_overall=detected
  fi

  support_profile=metadata_only
  collect_sensitive=0
  case "$PROFILE" in extended|debug) collect_sensitive=1 ;; esac
  [ "${PBW_COLLECT_ZYGISK_SUPPORT:-0}" = 1 ] && collect_sensitive=1
  denylist_state=skipped_profile_gate
  denylist_entries=0
  if [ "$collect_sensitive" = 1 ]; then
    support_profile=extended
    if command -v magisk >/dev/null 2>&1; then
      if magisk --denylist ls > "$denylist" 2>&1; then
        denylist_state=collected
        denylist_entries="$(grep -cve "^[[:space:]]*$" "$denylist" 2>/dev/null || true)"
      else
        denylist_state=command_failed
      fi
    else
      denylist_state=magisk_cli_missing
    fi
  fi

  {
    echo "zygisk_stack_inventory_time=$(date +%Y-%m-%dT%H:%M:%S%z)"
    for d in /data/adb/modules/zygisk_lsposed /data/adb/modules/LSPosed /data/adb/modules/vector /data/adb/modules/zygisk_vector; do
      [ -d "$d" ] || continue
      echo "--- module_path=$d ---"
      grep -E "^(id|name|version|versionCode|description)=" "$d/module.prop" 2>/dev/null || true
      [ -e "$d/disable" ] && echo "disable=present" || echo "disable=absent"
      [ -e "$d/remove" ] && echo "remove=present" || echo "remove=absent"
    done
    for logdir in /data/adb/lspd/log /data/adb/lspd/log.old; do
      [ -d "$logdir" ] || continue
      echo "--- log_dir=$logdir ---"
      find "$logdir" -maxdepth 1 -type f 2>/dev/null | sort | head -n 40 | while IFS= read -r f; do
        bytes="$(wc -c < "$f" 2>/dev/null || echo 0)"
        modified="$(date -r "$f" +%Y-%m-%dT%H:%M:%S%z 2>/dev/null || true)"
        echo "file bytes=$bytes modified=$modified path=$f"
      done
    done
    echo "privacy=lspd_config_contents_not_collected"
    echo "privacy=manager_app_private_cache_not_collected"
  } > "$inventory" 2>&1 || true

  {
    echo "zygisk_stack_time=$(date +%Y-%m-%dT%H:%M:%S%z)"
    echo "zygisk_support_profile=$support_profile"
    echo "zygisk_stack_overall=$zygisk_stack_overall"
    echo "lsposed_present=$lsposed_present"
    echo "lsposed_module_aliases=$lsposed_aliases"
    echo "lspd_runtime_present=$lspd_runtime_present"
    echo "lspd_service=$lspd_service"
    echo "lspd_pid_present=$( [ "$lspd_pid" = none ] && echo no || echo yes )"
    echo "lspd_log_files=$lspd_log_files"
    echo "lspd_rotated_log_files=$lspd_rotated_log_files"
    echo "lspd_config_files=$lspd_config_files"
    echo "lspd_config_contents_collected=no"
    echo "vector_present=$vector_present"
    echo "vector_module_aliases=$vector_aliases"
    echo "magisk_denylist_state=$denylist_state"
    echo "magisk_denylist_entries=$denylist_entries"
    echo "manager_private_cache_collected=no"
  } > "$summary" 2>&1 || true
}
collect_ashlooper_health
collect_zygisk_stack_support
collect_diagnostics_vnext_bundle
collect_module_runtime_logs
run_file "$RUN/dispatcher/pixel_drop_dispatch_status.txt" "if [ -d /data/adb/pixel-drop-dispatch ]; then find /data/adb/pixel-drop-dispatch -maxdepth 3 -type f -name '*.log' 2>/dev/null | sort | tail -30; echo; tail -160 /data/adb/pixel-drop-dispatch/log/health.log 2>/dev/null || tail -160 /data/adb/pixel-drop-dispatch/health.log 2>/dev/null || true; echo; tail -160 /data/adb/pixel-drop-dispatch/log/dispatch.log 2>/dev/null || tail -160 /data/adb/pixel-drop-dispatch/dispatch.log 2>/dev/null || true; else echo runtime_present=no; fi"
run_file "$RUN/thermal/thermal_power.txt" "dumpsys battery 2>/dev/null | head -160 || true; echo; dumpsys thermalservice 2>/dev/null | head -220 || true; echo; dumpsys power 2>/dev/null | head -220 || true; getprop | grep -i thermal || true"
run_file "$RUN/network/local_status.txt" "ip addr 2>/dev/null || true; echo; ip route 2>/dev/null || true; echo; ip rule 2>/dev/null || true; echo; getprop | grep -Ei 'net.dns|wifi|radio|telephony' || true; echo; dumpsys connectivity 2>/dev/null | head -220 || true; echo; dumpsys wifi 2>/dev/null | head -220 || true"

collect_dynamic() {
  label="$1"
  stage "$label"
  run_file "$RUN/logcat/logcat_all_tail_${label}.txt" "logcat -d -t 1200 2>/dev/null || true"
  run_file "$RUN/logcat/logcat_patterns_${label}.txt" "logcat -d -t 1800 2>/dev/null | grep -Ei 'FATAL EXCEPTION|ANR in|Watchdog|system_server|zygote|lspd|zygisk|rezygisk|dex2oat|installd|PackageManager|ActivityManager|lowmemorykiller|LMKD|tombstoned|AshReXcue|bootloop|rescue|XSupport|Lucky|SafetyCore|Permission denied|avc: denied|settings|Failed transaction' | tail -240 || true"
  collect_split_logcat "$label"
  run_file "$RUN/kernel/dmesg_tail_${label}.txt" "dmesg -T 2>/dev/null | tail -500 || dmesg 2>/dev/null | tail -500 || true"
  run_file "$RUN/kernel/dmesg_patterns_${label}.txt" "dmesg -T 2>/dev/null | grep -Ei 'avc: denied|lmk|lowmemory|panic|fatal|thermal|binder|zygisk|dex2oat|safetycore' | tail -240 || true"
  run_file "$RUN/storage/storage_${label}.txt" "df -h /data /storage/emulated/0 2>/dev/null || true; echo; du -d 1 /storage/emulated/0/Download 2>/dev/null | tail -80 || true; echo; du -x -d 1 /data 2>/dev/null | tail -80 || true"
  run_file "$RUN/memory/memory_${label}.txt" "free -h 2>/dev/null || cat /proc/meminfo | head -80; echo; cat /proc/pressure/memory 2>/dev/null || true; echo; cat /proc/pressure/io 2>/dev/null || true; echo; cat /proc/pressure/cpu 2>/dev/null || true; echo; ps -A -o PID,USER,RSS,NAME 2>/dev/null | tail -80 || true"
  copy_latest_files "/data/anr" "$RUN/anr" 30
  copy_latest_files "/data/tombstones" "$RUN/tombstones" 50
  copy_latest_files "/data/system/dropbox" "$RUN/dropbox" 80
}

sleep 20
collect_dynamic "plus_20s"
sleep 70
collect_dynamic "plus_90s"
sleep 150
collect_dynamic "plus_240s"

stage "final_dynamic_copy"
copy_latest_files "/data/anr" "$RUN/anr" 40
copy_latest_files "/data/tombstones" "$RUN/tombstones" 60
copy_latest_files "/data/system/dropbox" "$RUN/dropbox" 100

stage "classification"
classify_dir "$RUN/anr"
classify_dir "$RUN/tombstones"
classify_dir "$RUN/dropbox"
for f in "$RUN/logcat"/logcat_patterns_*.txt "$RUN/kernel"/dmesg_patterns_*.txt "$RUN/art/package_dexopt.txt" "$RUN/binder/service_health.txt"; do
  [ -f "$f" ] && classify_file "$f"
done

{
  echo "red_flags_time=$(date -Is)"
  grep -RhiE 'ANR in system_server|system_server.*ANR|Watchdog|FATAL EXCEPTION|bootloop|rescue|XSupport|Lucky|dex2oat|safetycore|Permission denied|Failed transaction|File name too long|LMKD|lowmemorykiller|avc: denied' "$RUN" 2>/dev/null | head -300 || true
} > "$REDFLAGS" 2>&1 || true
{
  echo "red_flags_summary_time=$(date +%Y-%m-%dT%H:%M:%S%z)"
  for pat in 'FinalizerWatchdogDaemon' 'Okio Watchdog' 'dex2oat' 'safetycore' 'zygisk_vector' 'Permission denied' 'Failed transaction' 'LMKD' 'lowmemorykiller' 'File name too long' 'avc: denied'; do
    c="$(grep -RhiF "$pat" "$RUN" 2>/dev/null | wc -l | tr -d ' ')"
    echo "$pat=$c"
  done
} > "$REDFLAGS_SUMMARY" 2>&1 || true

stage "summary"
anr_count="$(find "$RUN/anr" -type f 2>/dev/null | wc -l | tr -d ' ')"
tomb_count="$(find "$RUN/tombstones" -type f 2>/dev/null | wc -l | tr -d ' ')"
drop_count="$(find "$RUN/dropbox" -type f 2>/dev/null | wc -l | tr -d ' ')"
ash_count="$(find "$RUN/rescue/ashlooper_logs" -type f 2>/dev/null | wc -l | tr -d ' ')"
module_runtime_count="$(find "$RUN/module_runtime" -type f 2>/dev/null | wc -l | tr -d ' ')"
frosty_log_count="$(find "$RUN/module_runtime/frosty" -type f 2>/dev/null | wc -l | tr -d ' ')"
lsposed_old_log_count="$(find "$RUN/module_runtime/lsposed_log_old" -type f 2>/dev/null | wc -l | tr -d ' ')"
ashlooper_present="$(grep -E '^ashlooper_present=' "$RUN/rescue/ashlooper_health.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
ashlooper_version="$(grep -E '^version=' "$RUN/rescue/ashlooper_health.txt" 2>/dev/null | head -1 | cut -d= -f2-)"
ashlooper_intervention_possible="$(grep -E '^ashlooper_intervention_possible=' "$RUN/ashlooper_intervention/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
ashlooper_disabled_candidates="$(grep -E '^ashlooper_disabled_candidates=' "$RUN/ashlooper_intervention/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
ashlooper_reason_candidate="$(grep -E '^ashlooper_reason_candidate=' "$RUN/ashlooper_intervention/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
module_logs_missing_because_disabled="$(grep -E '^module_logs_missing_because_disabled=' "$RUN/ashlooper_intervention/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
pstore_files="$(find "$RUN/pstore/files" -type f 2>/dev/null | wc -l | tr -d ' ')"
split_logcat_files="$(find "$RUN/logcat_split" -type f 2>/dev/null | wc -l | tr -d ' ')"
focused_dumpsys_files="$(find "$RUN/dumpsys" -type f 2>/dev/null | wc -l | tr -d ' ')"
zygisk_support_profile="$(grep -E '^zygisk_support_profile=' "$RUN/zygisk_stack/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
zygisk_stack_overall="$(grep -E '^zygisk_stack_overall=' "$RUN/zygisk_stack/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
lsposed_present="$(grep -E '^lsposed_present=' "$RUN/zygisk_stack/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
lsposed_module_aliases="$(grep -E '^lsposed_module_aliases=' "$RUN/zygisk_stack/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
lspd_service="$(grep -E '^lspd_service=' "$RUN/zygisk_stack/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
lspd_log_files="$(grep -E '^lspd_log_files=' "$RUN/zygisk_stack/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
lspd_rotated_log_files="$(grep -E '^lspd_rotated_log_files=' "$RUN/zygisk_stack/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
lspd_config_files="$(grep -E '^lspd_config_files=' "$RUN/zygisk_stack/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
vector_present="$(grep -E '^vector_present=' "$RUN/zygisk_stack/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
vector_module_aliases="$(grep -E '^vector_module_aliases=' "$RUN/zygisk_stack/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
magisk_denylist_state="$(grep -E '^magisk_denylist_state=' "$RUN/zygisk_stack/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
magisk_denylist_entries="$(grep -E '^magisk_denylist_entries=' "$RUN/zygisk_stack/summary.txt" 2>/dev/null | tail -1 | cut -d= -f2-)"
{
  echo "Boot Watch Collector v$VERSION comprehensive summary"
  echo "version=$VERSION"
  echo "versionCode=$VERSION_CODE"
  echo "profile=$PROFILE"
  echo "run_id=$RUN_ID"
  echo "run_dir=$RUN"
  echo "archive_path=$ARCHIVE"
  echo "result_path=$RESULT_TXT"
  echo "last_result_path=$LAST_TXT"
  echo "action_result_path=$ACTION_TXT"
  echo "status_env_path=$STATUS_ENV"
  echo "result_log_protected_name=yes"
  echo "sortify_expected_hold=yes"
  echo "started=$(date -d @$START_EPOCH +%Y-%m-%dT%H:%M:%S%z 2>/dev/null || echo $START_EPOCH)"
  echo "finished=$(date +%Y-%m-%dT%H:%M:%S%z)"
  echo "anr_files=$anr_count"
  echo "tombstone_files=$tomb_count"
  echo "dropbox_files=$drop_count"
  echo "ashlooper_present=${ashlooper_present:-unknown}"
  echo "ashlooper_version=${ashlooper_version:-unknown}"
  echo "ashlooper_logs_found=$ash_count"
  echo "ashlooper_intervention_possible=${ashlooper_intervention_possible:-unknown}"
  echo "ashlooper_disabled_candidates=${ashlooper_disabled_candidates:-unknown}"
  echo "ashlooper_reason_candidate=${ashlooper_reason_candidate:-unknown}"
  echo "module_logs_missing_because_disabled=${module_logs_missing_because_disabled:-unknown}"
  echo "pstore_files=${pstore_files:-0}"
  echo "split_logcat_files=${split_logcat_files:-0}"
  echo "focused_dumpsys_files=${focused_dumpsys_files:-0}"
  echo "zygisk_support_profile=${zygisk_support_profile:-unknown}"
  echo "zygisk_stack_overall=${zygisk_stack_overall:-unknown}"
  echo "lsposed_present=${lsposed_present:-unknown}"
  echo "lsposed_module_aliases=${lsposed_module_aliases:-none}"
  echo "lspd_service=${lspd_service:-unknown}"
  echo "lspd_log_files=${lspd_log_files:-0}"
  echo "lspd_rotated_log_files=${lspd_rotated_log_files:-0}"
  echo "lspd_config_files=${lspd_config_files:-0}"
  echo "vector_present=${vector_present:-unknown}"
  echo "vector_module_aliases=${vector_module_aliases:-none}"
  echo "magisk_denylist_state=${magisk_denylist_state:-unknown}"
  echo "magisk_denylist_entries=${magisk_denylist_entries:-0}"
  echo "ashrexcue_safe_files=$ash_count"
  echo "module_runtime_files=$module_runtime_count"
  echo "frosty_log_files=$frosty_log_count"
  echo "lsposed_old_log_files=$lsposed_old_log_count"
  echo "logd_state_before=$logd_before"
  echo "logd_started_by_bootwatch=$LOGD_STARTED_BY_PBW"
  echo "logd_was_stopped=$LOGD_WAS_STOPPED"
  echo "logd_state_current=$(getprop init.svc.logd 2>/dev/null || true)"
  echo "storage_data=$(df -h /data 2>/dev/null | tail -1)"
  echo "storage_emulated=$(df -h /storage/emulated/0 2>/dev/null | tail -1)"
  echo "safetycore_dex2oat_permission_denied=$(grep -RhiE 'safetycore|dex2oat|Permission denied' "$RUN" 2>/dev/null | head -1 | sed 's/^[[:space:]]*//')"
  echo "settings_transaction_failure=$(grep -Rhi 'Failure calling service settings' "$RUN" 2>/dev/null | head -1 | sed 's/^[[:space:]]*//')"
  echo "system_server_red_flag=$(grep -RhiE 'ANR in system_server|Watchdog.*system_server' "$RUN" 2>/dev/null | head -1 | sed 's/^[[:space:]]*//')"
  fname_too_long_count="$(grep -E '^File name too long=' "$RUN/red_flags_summary.txt" 2>/dev/null | tail -1 | sed 's/^File name too long=//')"
  [ -n "$fname_too_long_count" ] || fname_too_long_count=0
  echo "file_name_too_long_count=$fname_too_long_count"
  if [ "$fname_too_long_count" = "0" ]; then
    echo "file_name_too_long=absent"
  else
    echo "file_name_too_long=present"
  fi
} > "$SUMMARY" 2>&1 || true

stage "logd_restore"
logd_before_restore="$(getprop init.svc.logd 2>/dev/null || true)"
log "logd_state_before_restore=$logd_before_restore"
if [ "$LOGD_STARTED_BY_PBW" = "1" ] && [ "$LOGD_WAS_STOPPED" = "1" ]; then
  stop logd 2>/dev/null || true
  sleep 1
  log "logd_restore_stop_attempted=1 final=$(getprop init.svc.logd 2>/dev/null || true)"
else
  log "logd_restore_skipped started_by_bootwatch=$LOGD_STARTED_BY_PBW was_stopped=$LOGD_WAS_STOPPED"
fi

stage "marker_archive_export"
{
  echo "RESULT: BOOT_WATCH_BOOT_DONE rc=0"
  echo "version=$VERSION"
  echo "versionCode=$VERSION_CODE"
  echo "profile=$PROFILE"
  echo "mode=boot"
  echo "run_id=$RUN_ID"
  echo "run_dir=$RUN"
  echo "archive_path=$ARCHIVE"
  echo "result_path=$RESULT_TXT"
  echo "last_result_path=$LAST_TXT"
  echo "action_result_path=$ACTION_TXT"
  echo "status_env_path=$STATUS_ENV"
  echo "result_log_protected_name=yes"
  echo "sortify_expected_hold=yes"
  echo "logd_state_final=$(getprop init.svc.logd 2>/dev/null || true)"
  echo "finished=$(date +%Y-%m-%dT%H:%M:%S%z)"
} > "$MARKER"

# Archive after marker so marker is included.
tar -czf "$ARCHIVE" -C "$RT/runs" "$RUN_ID" 2>>"$LOG" || true

stage "auto_export"
log "auto_export_start=$MOD/result-log-export.sh"
export_rc=0
if [ -f "$MOD/result-log-export.sh" ]; then
  /system/bin/sh "$MOD/result-log-export.sh" "$RUN" boot >> "$LOG" 2>&1 || export_rc="$?"
  log "auto_export_rc=$export_rc"
else
  export_rc=127
  log "auto_export_missing=$MOD/result-log-export.sh"
fi

if [ -x "$MOD/result-log-export.sh" ]; then
  /system/bin/sh "$MOD/result-log-export.sh" "$RUN" boot >> "$LOG" 2>&1 || true
fi

log "RESULT: BOOT_WATCH_BOOT_DONE rc=0 run_dir=$RUN archive=$ARCHIVE result=$RESULT_TXT"
exit 0
