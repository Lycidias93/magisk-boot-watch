# Archived Heimnetz release: pixel-boot-watch-v0.1.5.1

- Source repo: Lycidias93/heimnetz-geraete
- Source release URL: https://github.com/Lycidias93/heimnetz-geraete/releases/tag/pixel-boot-watch-v0.1.5.1
- Source tag: pixel-boot-watch-v0.1.5.1
- Original name: Pixel Boot Watch v0.1.5.1
- Published: 2026-06-05T09:23:21Z
- Created: 2026-06-05T09:23:13Z
- Draft: False
- Prerelease: False
- Latest flag at migration: True
- Proposed archive repo: Lycidias93/magisk-boot-watch
- Migration note: archived before deleting the Heimnetz release object; source git tag intentionally kept.

## Original changelog

# Pixel Boot Watch v0.1.5.1

Maintenance release after verified Pixel post-reboot run.

## Verification

- Pixel post-reboot run: `20260605_105319_boot`.
- `pbw_result=PASS`.
- `pbw_version=0.1.5.1`.
- `pbw_versionCode=16`.
- `pbw_file_name_too_long=absent`.
- `pbw_file_name_too_long_count=0`.
- `pbw_protected_names=yes`.
- `pbw_sortify_hold_expected=yes`.
- `pbw_ashlooper_present=yes`.
- `pbw_ashlooper_version=9.8`.
- `RESULT: PIXEL_BOOT_WATCH_RESULT_LOG_DONE rc=0`.
- `RESULT: PIXEL_BOOT_WATCH_ACTION_EXPORT_DONE rc=0`.
- No lingering `pixel-boot-watch` process after completion.

## Changes

- Writes protected `pixel_local__pixel-boot-watch-*` result logs so Sortify leaves Pixel-local Boot-Watch logs in place.
- Adds `pixel_local__pixel-boot-watch-status.env` as a machine-readable PASS/status file.
- Adds bounded AshLooper/AshReXcue health evidence.
- Adds Magisk Action before/after export state for result, pointer and status files.
- Fixes file-name-too-long status/count reporting:
  - `pbw_file_name_too_long=absent` when count is `0`.
  - `pbw_file_name_too_long_count=0`.

## Captured external findings

The verified run may still capture external Android/Magisk evidence such as dex2oat/SafetyCore permission failures, settings/package transaction failures, LMKD pressure, app ANRs and app tombstones. These are evidence only and are not Pixel Boot Watch release blockers.

## Scope

Pixel/Magisk boot evidence only. No DNS/HA/VIP/route changes.

## Original assets metadata

- pixel-boot-watch-magisk-v0.1.5.1.zip size=13194
- pixel-boot-watch-v0.1.5.1.sha256 size=113
- pixel-boot-watch-verify-v0.1.5.1.sh size=1404
- README_PIXEL_BOOT_WATCH_V0.1.5.1.md size=1304
