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
