# Boot Watch Collector log coverage and vNext plan

Status: 2026-06-06
Baseline: v0.2.4 final verified

## Current collection model

Boot Watch Collector is bounded, local-only, and exits after collection. It produces:

- protected readable result logs in `/storage/emulated/0/Download/pixel_local__boot-watch-*`
- machine-readable `pixel_local__boot-watch-status.env`
- a detailed archive `boot-watch-<run_id>.tar.gz`
- runtime data under `/data/adb/boot-watch/runs/<run_id>`

## Current coverage

- Boot/core state and focused properties.
- Magisk version, module matrix, Magisk logs, service.d/post-fs-data.d listing and syntax checks.
- Zygisk, ART, dex2oat, SafetyCore/GMS/GSF/Vending and LSPosed/lspd focus.
- Binder service health and audio safe-volume state.
- AshLooper/AshReXcue bounded health and log excerpts.
- Pixel drop-dispatch bounded status when present.
- Thermal, battery, power, connectivity, Wi-Fi, IP route/address/rule snapshots.
- Dynamic snapshots at about +20s, +90s and +240s.
- Bounded logcat/dmesg tails and red-flag pattern captures.
- Latest `/data/anr`, `/data/tombstones`, and `/data/system/dropbox` files.
- Classification, red-flag summary, result marker, summary and stage timeline.

## Not included yet

- Dedicated split logcat buffers (`main`, `system`, `crash`, `events`, `kernel`, optional `radio`).
- `dumpsys dropbox` index/tag summary.
- `/sys/fs/pstore` / ramoops evidence.
- Wakeup source and suspend-blocker snapshots.
- Focused `dumpsys activity` process/service/provider state.
- Focused `dumpsys meminfo` summary.
- `dumpsys jobscheduler`, `alarm`, `deviceidle` summaries.
- SurfaceFlinger, WindowManager and input summaries.
- Per-module runtime logs beyond module matrix, unless specifically known and bounded.
- LSPosed module/config summaries beyond lspd status/logs.
- Opt-in app/package focus for banking/TAN/SafetyCore/PixelXpert style issues.
- Full bugreport or incident report handoff.

## vNext plan

<!-- MODULE_RUNTIME_LOGS_VNEXT_START -->
## Module runtime log discovery follow-up

Pixel discovery identified the following bounded candidates:

| Source | Observed files | vNext handling |
| --- | --- | --- |
| Frosty | `/data/adb/modules/Frosty/logs/kernel_tweaks.log`, `ram.log`, `services.log` | `extended` profile bounded tails. |
| LSPosed/lspd current logs | `/data/adb/lspd/log/kmsg.log`, `props.txt`, `modules_*.log`, `verbose_*.log` | Already covered by current LSPosed log copy; keep bounded in result summaries. |
| LSPosed/lspd rotated logs | `/data/adb/lspd/log.old/kmsg.log`, `props.txt`, `modules_*.log`, `verbose_*.log` | `extended` profile bounded tails for post-reboot regression context. |
| ReZygisk / Treat Wheel / Vector / Zygisk Detach | No dedicated runtime log files observed in the report excerpt | Status-only until real log paths are observed. |
| Tricky Store / Play Integrity Fix / Anti SafetyCore / RVMM mount | No dedicated runtime log files observed in the report excerpt | Status-only; avoid sensitive/private dumps. |

Collector rule: module-owned log contents are not copied in `standard` by default. They are gated behind `extended` or `PBW_COLLECT_MODULE_LOGS=1`, use hardcoded allowlists, and use bounded tails only.
<!-- MODULE_RUNTIME_LOGS_VNEXT_END -->


1. Keep `standard` profile stable and unchanged by default.
2. Add `extended` profile for bounded additional diagnostics.
3. Add explicit profile selection through config/action flow.
4. Add split logcat buffers first; it has the best signal/risk ratio.
5. Add pstore/ramoops second; it is useful for hard reboot/kernel-watchdog cases.
6. Add focused dumpsys bundles behind profile gates.
7. Keep full bugreport outside default collection and document it as manual/opt-in only.

## Risk guard

- No network upload.
- No telemetry.
- No unbounded directory copies.
- No deep LSPosed database dumps by default.
- No app private data collection except explicit future opt-in package focus.
- Keep all result files local and protected with `pixel_local__*` naming.

<!-- ZYGISK_STACK_TREAT_WHEEL_VECTOR_VNEXT_20260606_START -->
## Zygisk-stack vNext: ReZygisk, Treat Wheel and Vector

Context:

- ReZygisk is a standalone Zygisk API implementation for KernelSU, APatch and Magisk.
- Treat Wheel is designed for ReZygisk-based setups and has a ReVanced Umount (`RVU`) flow using per-module `tw_config` files.
- Vector is an ART hooking framework running as a Zygisk module and is relevant to LSPosed/libxposed-style troubleshooting.

Current v0.2.4 already provides useful indirect signal:

- Magisk module matrix and marker state.
- Zygisk/ART mountinfo focus.
- dex2oat overlay checks.
- `lspd` status and latest LSPosed logs.
- logcat/dmesg patterns for Zygisk, ReZygisk, dex2oat, SafetyCore, permission denials and ART/hooking red flags.

Not yet ideal:

- No dedicated ReZygisk summary file.
- No Treat Wheel / `tw_config` inventory.
- No Vector-specific summary file.
- No combined Zygisk-stack health line in `summary.txt` or `status.env`.

Proposed v0.2.5/vNext implementation plan:

1. Add `extended` profile only; leave `standard` unchanged.
2. Create `$RUN/zygisk_stack/`.
3. Add `zygisk_stack/summary.txt` with machine-readable keys:
   - `zygisk_stack_rezygisk_present=`
   - `zygisk_stack_rezygisk_conflict_hint=`
   - `zygisk_stack_treat_wheel_present=`
   - `zygisk_stack_treat_wheel_tw_config_count=`
   - `zygisk_stack_vector_present=`
   - `zygisk_stack_vector_red_flags=`
   - `zygisk_stack_lspd_present=`
   - `zygisk_stack_overall=`
4. Add bounded status files:
   - `rezygisk_status.txt`
   - `treat_wheel_status.txt`
   - `vector_status.txt`
   - `lsposed_status.txt`
5. Add safe discovery patterns:
   - module metadata from `/data/adb/modules/*/module.prop`
   - bounded status/log files under known module runtime paths
   - `tw_config` files from module folders, with hard line limits
   - logcat/dmesg red-flag extraction using already bounded collectors
6. Add export snippets to protected result text.
7. Add summary keys to `pixel_local__boot-watch-status.env` only after the collector is proven stable.

Explicit non-goals:

- Do not dump LSPosed SQLite databases by default.
- Do not collect app-private data.
- Do not modify Magisk/ReZygisk/Vector/Treat Wheel settings.
- Do not run full bugreport automatically.
<!-- ZYGISK_STACK_TREAT_WHEEL_VECTOR_VNEXT_20260606_END -->
## Runtime proof: extended module runtime logs

Status: PASS on 2026-06-06 after PR #11 (`8d3cf1a`).

The extended collector was verified with `PBW_PROFILE=extended` and `module_runtime_logs_enabled=1`. It collected bounded Frosty logs and LSPosed/lspd `log.old` files, produced `verify_extended_runtime=pass`, and confirmed `db_leak=absent`. The active stable scripts were restored after the test.

See `docs/module-runtime-logs-extended-test-20260606.md` for the full proof.
