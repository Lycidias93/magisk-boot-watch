# Changelog

## v0.2.0 - 2026-06-05

- Rename internal module id from `pixel-boot-watch` to `boot-watch`.
- Move runtime path from `/data/adb/pixel-boot-watch` to `/data/adb/boot-watch`.
- Mark legacy `pixel-boot-watch` module for removal during install/service start.
- Add Magisk `updateJson` support through root `update.json`.
- Rename public result artifacts to `boot-watch-*` while preserving the Pixel-local protected prefix for Sortify holds.
- Publish release asset `magisk-boot-watch-v0.2.0.zip`.

## v0.1.5.1 - 2026-06-05

- Public baseline release as Boot Watch Collector.
- Verified baseline build still used legacy internal id `pixel-boot-watch`.

## v0.2.1 - 2026-06-05

- Fixes automatic boot export after the v0.2.0 module-id migration.
- Executes result-log-export.sh through /system/bin/sh and logs auto_export_rc.
- Hardens Magisk install permissions for module scripts.
