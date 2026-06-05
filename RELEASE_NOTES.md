# Boot Watch Collector v0.2.2

v0.2.2 fixes the active-module marker regression found after v0.2.1.

## Fixed

- Corrects the installer legacy cleanup path from boot-watch to pixel-boot-watch.
- Prevents the active boot-watch module from being marked with disable/remove.
- Clears accidental disable/remove markers from the active MODPATH during install.
- Keeps the v0.2.1 robust automatic boot export hook.
- Corrects the installer banner to v0.2.2.

## Expected output

After reboot, Boot Watch Collector should stay enabled and create:

- /storage/emulated/0/Download/pixel_local__boot-watch-status.env
- /storage/emulated/0/Download/pixel_local__boot-watch-last-result.txt
- /storage/emulated/0/Download/boot-watch-<run_id>.tar.gz

## Update channel

Online updates remain enabled through update.json.
