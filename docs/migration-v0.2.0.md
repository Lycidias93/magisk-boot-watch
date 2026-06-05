# Boot Watch Collector v0.2.0

This release performs the real public id migration and enables Magisk online updates.

## Changed

- New module id: `boot-watch`
- New runtime path: `/data/adb/boot-watch`
- Legacy module id `pixel-boot-watch` is marked for removal when present.
- `updateJson` now points to the public repository `update.json`.
- Public release asset: `magisk-boot-watch-v0.2.0.zip`

## Migration behavior

Flash v0.2.0 in Magisk and reboot. Do not manually delete the old module before flashing. The installer/service marks `/data/adb/modules/pixel-boot-watch` with `disable` and `remove` if present.

Existing old evidence under `/data/adb/pixel-boot-watch` is not deleted.

## Expected post-reboot status

```text
id=boot-watch
version=0.2.0
versionCode=20
runtime=/data/adb/boot-watch
legacy /data/adb/modules/pixel-boot-watch/remove=present or legacy module absent
```
