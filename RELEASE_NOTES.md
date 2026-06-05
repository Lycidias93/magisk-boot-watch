# Boot Watch Collector v0.2.1

v0.2.1 fixes the first post-migration export issue found after v0.2.0.

## Fixed

- Restores automatic boot result export for the new boot-watch namespace.
- Runs result-log-export.sh through /system/bin/sh instead of depending on the executable bit.
- Adds explicit auto_export_start and auto_export_rc markers to the boot log.
- Hardens Magisk install permissions for service, action, manual collection and export scripts.

## Expected output

After reboot, Boot Watch Collector should create:

- /storage/emulated/0/Download/pixel_local__boot-watch-status.env
- /storage/emulated/0/Download/pixel_local__boot-watch-last-result.txt
- /storage/emulated/0/Download/boot-watch-<run_id>.tar.gz

## Update channel

Online updates remain enabled through update.json.
