# Boot Watch Collector

Magisk boot evidence collector for Android devices.

Boot Watch Collector collects bounded post-boot diagnostics such as Magisk module state, ANR/tombstone/dropbox evidence, logcat red flags, ART/Zygisk hints, storage and memory pressure, and AshLooper health. It exports readable protected result logs to Download and then exits.

No daemon. No network upload. No telemetry.

## Current release

- Version: `v0.2.4`
- Module id: `boot-watch`
- Runtime path: `/data/adb/boot-watch`
- Online update metadata: [`update.json`](./update.json)
- Release ZIP: `magisk-boot-watch-v0.2.4.zip`
- Verified status: Pixel post-reboot PASS with `pbw_result=PASS`, `pbw_version=0.2.4`, `pbw_versionCode=24`, and no active `disable`/`remove` marker.

## Migration from v0.1.x

The first public baseline v0.1.5.1 intentionally kept the verified legacy module id `pixel-boot-watch`. v0.2.0 performs the real id migration:

- old id: `pixel-boot-watch`
- new id: `boot-watch`
- old module folder: `/data/adb/modules/pixel-boot-watch`
- new module folder: `/data/adb/modules/boot-watch`

When a v0.2.x release is flashed, it marks the legacy `pixel-boot-watch` module with `disable` and `remove` if the old module folder exists. The active `boot-watch` module is protected by the v0.2.4 service-time marker self-heal. Existing old runtime evidence under `/data/adb/pixel-boot-watch` is left untouched.

## Result files

On Pixel/Termux setups using Sortify protected names, v0.2.4 writes protected result logs such as:

```text
/storage/emulated/0/Download/pixel_local__boot-watch-<run_id>-result.txt
/storage/emulated/0/Download/pixel_local__boot-watch-last-result.txt
/storage/emulated/0/Download/pixel_local__boot-watch-action-last-result.txt
/storage/emulated/0/Download/pixel_local__boot-watch-status.env
/storage/emulated/0/Download/boot-watch-<run_id>.tar.gz
```

The protected `pixel_local__*` files are intended to stay local on the Pixel and are safe for Sortify hold workflows. The archive contains the detailed run directory from `/data/adb/boot-watch/runs/<run_id>`.

## Collected evidence / log coverage

Boot Watch Collector is bounded and local-only. It captures snapshots after boot and then exits. Current coverage:

| Area | What is collected |
| --- | --- |
| Boot/core state | Time, uptime, uid, SELinux mode, boot-completed props, boot reason, verified boot state, slot suffix, build fingerprint, Android release/SDK, focused boot/debug/system props. |
| Magisk | `magisk -V`, `magisk -v`, `/data/adb/magisk.log`, `/cache/magisk.log`, module matrix with `id`, `name`, `version`, `versionCode`, `disable`, `remove`, `service.sh`, `post-fs-data.sh`, and `action.sh`. |
| Magisk service scripts | `/data/adb/service.d` and `/data/adb/post-fs-data.d` listing plus quick shell syntax checks. |
| Zygisk / ART / LSPosed | Mountinfo focus for Zygisk/Vector/ReZygisk/LSPosed/ART, dex2oat overlay check, `lspd` service status, latest LSPosed logs, ART dexopt/staged-session status, SafetyCore/GMS/GSF/Vending package paths. |
| Binder/service health | `service check` for settings/package/activity/power/dropbox, focused service list, selected safe-volume and SafetyCore checks. |
| Audio safe-volume state | Audio safe-volume service marker and selected global audio safe-volume/CSD settings. |
| AshLooper / AshReXcue | Module presence, version, marker state, service/action scripts, selected config/state/status props, bounded candidate file list, and up to 10 bounded health/log excerpts. |
| Dispatcher | Pixel drop-dispatch runtime presence plus bounded health/dispatch log tails if present. |
| Thermal / power | `dumpsys battery`, `dumpsys thermalservice`, `dumpsys power`, and thermal-related props. |
| Network | `ip addr`, `ip route`, `ip rule`, focused DNS/Wi-Fi/radio/telephony props, `dumpsys connectivity`, and `dumpsys wifi`. |
| Dynamic boot snapshots | At about +20s, +90s, and +240s: logcat tail, logcat red-flag patterns, dmesg tail, dmesg red-flag patterns, storage pressure, memory/PSI pressure, process RSS tail, latest ANR files, tombstones, and dropbox files. |
| Classification/red flags | Bounded classification from ANR/tombstone/dropbox/logcat/dmesg/ART/binder files; summary counts for dex2oat, SafetyCore, permission denials, LMKD, file-name-too-long, SELinux denials, and transaction failures. |
| Export summary | Human-readable protected result text, machine-readable `status.env`, result marker, stages, summary, red flags, module matrix, storage/memory excerpts, latest archived file list, and result marker. |

Privacy note: collected archives may contain package names, process names, device properties, local paths, and crash/log snippets. Review before sharing publicly.

## Not collected yet / vNext candidates

The current release intentionally avoids heavy full-device dumps. Good vNext candidates:

| Candidate | Why it is useful | Risk / default |
| --- | --- | --- |
| Split logcat buffers | Capture `main`, `system`, `crash`, `events`, `kernel`, and optional `radio` separately for cleaner triage. | Low; keep bounded by lines/time. |
| `dumpsys dropbox` index | Adds event timestamps/tags even when copied dropbox files are sparse. | Low-medium; redact or bound output. |
| pstore / ramoops | Helps diagnose kernel panic, watchdog, hard reboot, or early-boot crash. | Low if files exist; bounded copy from `/sys/fs/pstore`. |
| Kernel wakeup/suspend blockers | Helps with reboot, suspend, battery-drain, and wake-lock issues. | Low-medium; sysfs availability varies. |
| `dumpsys activity` focused state | Process/service/provider state around boot stalls, ANRs, or binder pressure. | Medium; output can be large, must be tightly bounded. |
| `dumpsys meminfo` focused summary | Better memory pressure triage than `free`/PSI alone. | Medium; bound total and avoid per-app deep dumps by default. |
| `dumpsys jobscheduler` / `alarm` / `deviceidle` | Useful for post-boot delayed jobs, idle restrictions, and wake behavior. | Medium; bound output. |
| SurfaceFlinger / WindowManager / input | Useful for black screen, lockscreen, keyboard, touch, or UI boot issues. | Medium; collect only focused summaries. |
| Magisk per-module runtime logs | Current module matrix shows state, but not every module's own log files. | Medium; require allowlist or safe path patterns. |
| LSPosed module/config snapshot | Useful for module activation and scope issues. | Medium-high; avoid DB dumps by default, use allowlisted summaries only. |
| App-specific package focus | Targeted package state for SafetyCore, banking, TAN, PixelXpert, etc. | Medium-high; opt-in package allowlist only. |
| Optional full bugreport handoff | Maximum context for rare issues. | High/heavy/privacy-sensitive; never default. |

Proposed vNext direction: keep `standard` profile as-is, add `extended` for bounded extra diagnostics, and add `profile.env`/Action selection so heavy or privacy-sensitive collectors remain opt-in.

## Install

Flash `magisk-boot-watch-v0.2.4.zip` in Magisk, reboot, then check the protected result logs in Download.

## v0.2.1 auto-export fix

v0.2.1 restores automatic Download result export after the v0.2.0 module-id migration.
It also hardens Magisk install permissions and keeps update.json pointed at the latest release.

## v0.2.2 active-module marker fix

v0.2.2 prevents the active boot-watch module from being marked for disable/remove during install.
It keeps the robust v0.2.1 auto-export hook and updates update.json to the latest release.


## v0.2.3 service marker guard

This release fixes service-time active marker cleanup so the active `boot-watch` module cannot be marked with `disable` or `remove` by its own service.


## v0.2.4 versionCode metadata fix

This release aligns the runtime `pbw_versionCode` export with `module.prop` and `update.json` while keeping the active marker self-heal and auto-export path intact.
