# Boot Watch Collector v0.2.10

Release: `0.2.10-webui-runtime-root-hotfix`
Version code: `35`

## Summary

This release fixes the WebUI run-history exporter after the module moved to the current `boot-watch` naming and protected result-file scheme.

## Fixed

- WebUI run history now reads `pixel_local__boot-watch-<run_id>-result.txt` files.
- WebUI status/log exporters now read the current `pixel_local__boot-watch-*` protected files.
- Legacy `pixel-boot-watch` protected result-file names are no longer used by active WebUI exporters.

## Added to reproducible source

- `src/magisk-module/tools/boot-watch-webui-log-export.sh`
- `src/magisk-module/tools/boot-watch-webui-status-export.sh`
- `src/magisk-module/tools/action.original-before-webui-wrapper.sh`
- `src/magisk-module/webroot/` read-only WebUI assets

## Verified

- ZIP verify PASS.
- Installed on Pixel via Magisk.
- Post-reboot runtime PASS.
- Active module version: `0.2.10-webui-runtime-root-hotfix`.
- Active old-prefix count: `0`.
- Latest verified boot run: `20260707_174749_boot`.
- WebUI run history lists current `pixel_local__boot-watch-*` result files.
