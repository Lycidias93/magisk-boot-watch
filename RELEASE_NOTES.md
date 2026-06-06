# Boot Watch Collector v0.2.4

v0.2.4 is a focused metadata fix for the v0.2.3 runtime export.

## Fixed

- Corrected `boot-watch.sh` so `VERSION_CODE=24` is exported consistently.
- Aligns `module.prop`, `update.json`, result marker, and `pixel_local__boot-watch-status.env`.
- Keeps the v0.2.3 active-module marker self-heal unchanged.
- Keeps the v0.2.1+ automatic boot result export path intact.

## Expected post-reboot state

- `id=boot-watch`
- `version=0.2.4`
- `versionCode=24`
- `new_disable=absent`
- `new_remove=absent`
- `pbw_versionCode=24`
- `pbw_result=PASS`
