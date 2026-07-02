# Archived Heimnetz release: pixel-boot-watch-v0.1.4

- Source repo: Lycidias93/heimnetz-geraete
- Source release URL: https://github.com/Lycidias93/heimnetz-geraete/releases/tag/pixel-boot-watch-v0.1.4
- Source tag: pixel-boot-watch-v0.1.4
- Original name: Release Pixel Boot Watch v0.1.4
- Published: 2026-06-02T08:56:22Z
- Created: 2026-06-02T08:56:16Z
- Draft: False
- Prerelease: False
- Latest flag at migration: False
- Proposed archive repo: Lycidias93/magisk-boot-watch
- Migration note: archived before deleting the Heimnetz release object; source git tag intentionally kept.

## Original changelog

# Pixel Boot Watch v0.1.4

Stable release after successful Pixel Android 16 post-reboot verify.

## Verification

- `RESULT: PIXEL_BOOT_WATCH_BOOT_DONE rc=0`
- `version=0.1.4`
- `versionCode=14`
- `RESULT: PIXEL_BOOT_WATCH_RESULT_LOG_DONE rc=0`
- `RESULT: PIXEL_BOOT_WATCH_ACTION_EXPORT_DONE rc=0`
- `classification.txt`, `red_flags.txt`, `summary.txt`, `result.marker` present
- boot archive present
- no `File name too long`
- no lingering `pixel-boot-watch` process after completion

## Changes

- Comprehensive one-shot collector with bounded local snapshots.
- Adds module matrix, Zygisk/ART/LSPosed, binder/settings health, audio-safe status, rescue/dispatcher, thermal/power/network, storage/memory/kernel/logcat, ANR/tombstone/dropbox classification.
- Magisk Action Button exports latest run to Download.
- Still no daemon, no network, no upload after successful collection.

## Captured external findings

The verified run captured separate operational issues: dex2oat `Permission denied` for `org.lsposed.dirtysepolicy` and `com.google.android.safetycore`, settings transaction failure, LMKD pressure, app ANRs and app tombstones. These are evidence, not Boot Watch release blockers.

## Scope

Pixel/Magisk boot evidence only. No DNS/HA/VIP/route changes.

## Original assets metadata

- pixel-boot-watch-magisk-v0.1.4.zip size=24934
- pixel-boot-watch-v0.1.4.sha256 size=301
- pixel-boot-watch-verify-v0.1.4.sh size=1213
- README_PIXEL_BOOT_WATCH_V0.1.4.md size=2410
