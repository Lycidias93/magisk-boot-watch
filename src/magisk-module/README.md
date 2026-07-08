# Boot Watch Collector

Magisk boot evidence collector for Android devices.

This directory contains the installable Magisk module source.

## v0.2.10-webui-runtime-root-hotfix

- Keeps module id `boot-watch`.
- Keeps runtime path `/data/adb/boot-watch`.
- Fixes WebUI run history to read current protected result files:
  - `pixel_local__boot-watch-<run_id>-result.txt`
  - `pixel_local__boot-watch-last-result.txt`
  - `pixel_local__boot-watch-action-last-result.txt`
  - `pixel_local__boot-watch-status.env`
- Removes active WebUI dependency on legacy protected result-file names from the old pixel-boot-watch naming scheme.
- Includes read-only WebUI assets under `webroot/` and exporter helpers under `tools/`.

## Runtime

- Collector runtime: `/data/adb/boot-watch`
- WebUI runtime JSON: `/data/adb/modules/boot-watch/webroot`
- Protected result files stay in Android Download as `pixel_local__boot-watch-*`.
