# Boot Watch Collector

Magisk boot evidence collector for Android devices.

Boot Watch Collector collects bounded post-boot diagnostics such as Magisk module state, ANR/tombstone/dropbox evidence, logcat red flags, ART/Zygisk hints, storage and memory pressure, and AshLooper health. It exports readable protected result logs to Download and then exits.

No daemon. No network upload. No telemetry.

## Current release

- Version: `v0.2.0`
- Module id: `boot-watch`
- Runtime path: `/data/adb/boot-watch`
- Online update metadata: [`update.json`](./update.json)
- Release ZIP: `magisk-boot-watch-v0.2.0.zip`

## Migration from v0.1.x

The first public baseline v0.1.5.1 intentionally kept the verified legacy module id `pixel-boot-watch`. v0.2.0 performs the real id migration:

- old id: `pixel-boot-watch`
- new id: `boot-watch`
- old module folder: `/data/adb/modules/pixel-boot-watch`
- new module folder: `/data/adb/modules/boot-watch`

When v0.2.0 is flashed, it marks the legacy module with `disable` and `remove` if the old module folder exists. Existing old runtime evidence under `/data/adb/pixel-boot-watch` is left untouched.

## Result files

On Pixel/Termux setups using Sortify protected names, v0.2.0 writes protected result logs such as:

```text
/storage/emulated/0/Download/pixel_local__boot-watch-last-result.txt
/storage/emulated/0/Download/pixel_local__boot-watch-action-last-result.txt
/storage/emulated/0/Download/pixel_local__boot-watch-status.env
```

## Install

Flash `magisk-boot-watch-v0.2.0.zip` in Magisk, reboot, then check the protected result logs in Download.

## v0.2.1 auto-export fix

v0.2.1 restores automatic Download result export after the v0.2.0 module-id migration.
It also hardens Magisk install permissions and keeps update.json pointed at the latest release.
