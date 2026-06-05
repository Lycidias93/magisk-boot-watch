# Boot Watch Collector v0.2.3

v0.2.3 is a focused stability fix for the service-time active marker guard.

## Fixed

- Corrected `service.sh` so it no longer marks `/data/adb/modules/boot-watch` with `disable` and `remove`.
- Legacy cleanup is scoped to `/data/adb/modules/pixel-boot-watch` only.
- Added service-time active self-heal that removes accidental `disable` and `remove` markers from the active `boot-watch` module.
- Kept the v0.2.1+ automatic boot result export path intact.

## Expected post-reboot state

- `id=boot-watch`
- `version=0.2.3`
- `versionCode=23`
- `new_disable=absent`
- `new_remove=absent`
- `auto_export_rc=0`
- `pbw_result=PASS`
