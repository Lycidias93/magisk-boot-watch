# Archived Heimnetz release: pixel-boot-watch-v0.1.3

- Source repo: Lycidias93/heimnetz-geraete
- Source release URL: https://github.com/Lycidias93/heimnetz-geraete/releases/tag/pixel-boot-watch-v0.1.3
- Source tag: pixel-boot-watch-v0.1.3
- Original name: Pixel Boot Watch v0.1.3
- Published: 2026-05-31T21:32:41Z
- Created: 2026-05-31T21:32:39Z
- Draft: False
- Prerelease: False
- Latest flag at migration: False
- Proposed archive repo: Lycidias93/magisk-boot-watch
- Migration note: archived before deleting the Heimnetz release object; source git tag intentionally kept.

## Original changelog

Pixel Boot Watch v0.1.3

Stable release after successful Pixel post-reboot verify.

Verification:
- RESULT: PIXEL_BOOT_WATCH_BOOT_DONE rc=0
- RESULT: PIXEL_BOOT_WATCH_RESULT_LOG_DONE rc=0
- PBW_V013_RELEASE_GATE=PASS
- no File name too long in final result log

Fixes:
- safe AshReXcue copy excludes old Pixel Boot Watch runs/backups
- hashed/truncated filenames prevent recursive filename growth
- readable Download result log after each boot

Captured external issue:
- SafetyCore/dex2oat Permission denied was captured as evidence and remains a separate Zygisk/ART/SafetyCore path, not a Boot Watch failure.

Scope: Pixel/Magisk boot evidence only. No DNS/HA/VIP/route change.

## Original assets metadata

- pixel-boot-watch-magisk-v0.1.3.zip size=6648
- pixel-boot-watch-v0.1.3.sha256 size=301
- pixel-boot-watch-verify-v0.1.3.sh size=1466
- README_PIXEL_BOOT_WATCH_V0.1.3.md size=1482
