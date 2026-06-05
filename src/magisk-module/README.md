# Boot Watch Collector

Magisk boot evidence collector for Android devices.

This is the installable Magisk module source for Boot Watch Collector.

## v0.2.0 migration

- New module id: `boot-watch`
- New runtime path: `/data/adb/boot-watch`
- Legacy v0.1.x module id `pixel-boot-watch` is marked with `disable` and `remove` during install/service start when present.
- Existing legacy runtime data under `/data/adb/pixel-boot-watch` is not deleted automatically.
