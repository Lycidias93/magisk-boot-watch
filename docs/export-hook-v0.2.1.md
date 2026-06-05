# Boot Watch Collector v0.2.1 export-hook fix

v0.2.0 successfully migrated the module id from pixel-boot-watch to boot-watch, but the boot run did not automatically export the readable Download status files.

Root cause: the boot collector no longer contained a visible result-log-export hook, and installed module scripts were not executable after flashing.

v0.2.1 fixes this by:

- executing result-log-export.sh through /system/bin/sh during boot finalization
- logging auto_export_start and auto_export_rc into boot-watch.log
- setting executable permissions for service/action/export scripts in customize.sh
- keeping update.json online-update metadata current
