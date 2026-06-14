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
<!-- ASHLOOPER_INTERVENTION_VNEXT_20260609_START -->
## AshLooper intervention vNext: disabled-module evidence gap

Status: planned after the installed `v0.2.5` stable post-reboot PASS.

Observed case:

- AshLooper disabled a module because of a pTune-related stability/boot-loop protection event.
- By the time Boot Watch ran, the affected module was already disabled.
- The module therefore had no useful runtime logs available for collection.
- Boot Watch `standard` PASS remained correct, but the root-cause evidence was incomplete.

Current signal already collected:

- AshLooper module health and `settings.prop` excerpts.
- Per-module matrix with `disable=` and `remove=` marker state.
- Module metadata from `module.prop`.
- Standard boot result and protected export files.

vNext goal:

Capture AshLooper intervention context explicitly so a protection action is visible even when the target module no longer produces logs.

Proposed collector additions:

1. Add an `ashlooper_intervention/` run directory.
2. Capture bounded AshLooper state on every profile:
   - `/data/adb/modules/AshLooper/settings.prop`
   - AshLooper module metadata
   - `disable`, `remove`, `service.sh`, `post-fs-data.sh`, `action.sh` marker state
   - bounded AshLooper runtime/state files under known `/data/adb/ashlooper` paths
3. Snapshot disabled/remove module markers for all modules:
   - module id
   - `disable=present|absent`
   - `remove=present|absent`
   - marker mtime where available
   - module `version` and `versionCode`
4. Add a conservative classifier:
   - `ashlooper_intervention_possible=yes|no|unknown`
   - `ashlooper_disabled_candidates=<ids>`
   - `ashlooper_reason_candidate=ptune|unknown`
   - `module_logs_missing_because_disabled=yes|no|unknown`
5. Keep root-cause language conservative:
   - report `module disabled before collection`
   - do not claim AshLooper caused the disable unless an AshLooper file explicitly proves it
6. Preserve privacy/risk guard:
   - no app-private data
   - no unbounded recursive copies
   - no automatic re-enable or module mutation
   - no network upload
   - standard profile may collect marker/state metadata, but module log contents stay gated behind `extended`

Expected result text additions:

- `## AshLooper intervention`
- `ashlooper_intervention_possible=...`
- `ashlooper_disabled_candidates=...`
- `module_logs_missing_because_disabled=...`

Acceptance criteria:

- A boot where AshLooper disables pTune or another module produces a PASS result plus an explicit disabled-before-collection explanation.
- Missing module logs are classified as an evidence limitation, not as a Boot Watch collector failure.
- Existing v0.2.5 standard behavior stays bounded and read-only.
<!-- ASHLOOPER_INTERVENTION_VNEXT_20260609_END -->

## Runtime proof: extended module runtime logs

Status: PASS on 2026-06-06 after PR #11 (`8d3cf1a`).

The extended collector was verified with `PBW_PROFILE=extended` and `module_runtime_logs_enabled=1`. It collected bounded Frosty logs and LSPosed/lspd `log.old` files, produced `verify_extended_runtime=pass`, and confirmed `db_leak=absent`. The active stable scripts were restored after the test.

See `docs/module-runtime-logs-extended-test-20260606.md` for the full proof.


## v0.2.5-test.1 packaging note

- `v0.2.5-test.1` is a manual prerelease ZIP, not a stable update-channel rollout.
- The stable `update.json` intentionally remains on v0.2.4 until the extended collector has another post-install/post-reboot proof.
- `standard` profile behavior remains unchanged.
- `extended` profile or `PBW_COLLECT_MODULE_LOGS=1` enables bounded Frosty and LSPosed/lspd rotated log tails.

## Installed v0.2.5-test.1 proof

The `v0.2.5-test.1` prerelease is installed-validated on Pixel:

- standard post-reboot run: `PASS`, `profile=standard`, `module_runtime_logs_enabled=0`
- installed extended run: `PASS`, `profile=extended`, `module_runtime_logs_enabled=1`
- collected Frosty bounded tails: `3`
- collected LSPosed/lspd rotated bounded tails: `4`
- LSPosed DB leak guard: `db_leak=absent`
- file-name-too-long guard: `absent`

See [`v0.2.5-test.1-post-reboot-and-extended-proof-20260608.md`](./v0.2.5-test.1-post-reboot-and-extended-proof-20260608.md).

<!-- V025_STABLE_PROMOTION_20260608 -->

## v0.2.5 stable promotion — 2026-06-08

`v0.2.5` promotes the previously validated `v0.2.5-test.1` extended module runtime collector to the stable update channel.

Validation baseline:

- Standard post-reboot run: PASS.
- Installed extended profile run: PASS.
- Frosty bounded tails: 3.
- LSPosed/lspd rotated bounded tails: 4.
- LSPosed DB leak guard: `db_leak=absent`.
- File-name-too-long guard: `absent`.

The `standard` profile remains bounded and unchanged for foreign module log contents; Frosty/LSPosed runtime log content collection remains gated by `PBW_PROFILE=extended`, `PBW_PROFILE=debug`, or `PBW_COLLECT_MODULE_LOGS=1`.
