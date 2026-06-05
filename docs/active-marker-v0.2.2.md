# Boot Watch Collector v0.2.2 active-module marker fix

v0.2.1 restored the automatic boot export hook, but its installer still used the wrong legacy module path.

Root cause: customize.sh pointed LEGACY_MOD at /data/adb/modules/boot-watch instead of /data/adb/modules/pixel-boot-watch. This could mark the active module with disable/remove during flashing.

v0.2.2 fixes this by:

- only marking the real legacy pixel-boot-watch module for removal
- refusing to mark a path whose module.prop already says id=boot-watch
- clearing accidental disable/remove markers from MODPATH during install
- keeping the v0.2.1 robust auto-export hook
- correcting the installer banner to v0.2.2
