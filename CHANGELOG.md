# Changelog


## v0.2.5 - 2026-06-08

- Promoted `v0.2.5` stable after installed `v0.2.5-test.1` validation on Pixel.
- Enabled stable online update metadata with `versionCode=26` and release asset `magisk-boot-watch-v0.2.5.zip`.
- Kept `standard` profile bounded and unchanged for foreign module log contents.
- Added opt-in `extended` module runtime tails for Frosty logs and LSPosed/lspd rotated logs.
- Preserved privacy guards: no LSPosed DB dumps, no app-private data, and installed proof showed `db_leak=absent` plus `file_name_too_long=absent`.

## v0.2.5-test.1 installed proof - 2026-06-08

- Documented installed Pixel post-reboot standard PASS for `v0.2.5-test.1`.
- Documented installed extended module-runtime PASS with Frosty logs and LSPosed/lspd rotated logs.
- Confirmed `db_leak=absent` for LSPosed DB guard and `file_name_too_long=absent`.

## Unreleased

- Planned AshLooper intervention coverage for disabled-module events where protection disables a module before Boot Watch can collect its runtime logs.


## v0.2.5-test.1 - 2026-06-07

- Test prerelease for opt-in `extended` module runtime log collectors.
- Adds bounded Frosty log tails and LSPosed/lspd `log.old` tails when `PBW_PROFILE=extended` or `PBW_COLLECT_MODULE_LOGS=1`.
- Keeps stable `update.json` on v0.2.4; this test ZIP is manual/pre-release only.
- Fixes `action.sh --status` module id output from the old `pixel-boot-watch` label to `boot-watch`.

- Document runtime proof for the extended module runtime log collector; test passed with `db_leak=absent` and stable scripts restored.
- Add opt-in extended module-runtime log collectors for Frosty logs and rotated LSPosed/lspd logs, keeping standard profile unchanged.

- Fixed `update.json` changelog metadata to use raw Markdown instead of a GitHub HTML release page.
- Document ReZygisk / Treat Wheel / Vector as a dedicated Zygisk-stack vNext collector target.
- Clarify that the next implementation should use a gated `extended` profile, keeping v0.2.4 `standard` behavior unchanged.

## 2026-06-06 - README log coverage and vNext plan

- Documented current Boot Watch Collector v0.2.4 log/evidence coverage in README.
- Added a public vNext candidate list for additional bounded diagnostics.
- Added `docs/log-coverage-vnext.md` with the detailed coverage and risk plan.

## v0.2.4

- Fix runtime versionCode metadata mismatch from v0.2.3.
- Align `boot-watch.sh` exported `VERSION_CODE` with `module.prop` and `update.json`.
- Keep active marker self-heal and auto-export behavior unchanged.


## v0.2.3

- Fix service-time active marker regression from v0.2.2.
- Scope legacy cleanup to `/data/adb/modules/pixel-boot-watch` only.
- Add active `boot-watch` marker self-heal during service start.


## v0.2.0 - 2026-06-05

- Rename internal module id from `pixel-boot-watch` to `boot-watch`.
- Move runtime path from `/data/adb/pixel-boot-watch` to `/data/adb/boot-watch`.
- Mark legacy `pixel-boot-watch` module for removal during install/service start.
- Add Magisk `updateJson` support through root `update.json`.
- Rename public result artifacts to `boot-watch-*` while preserving the Pixel-local protected prefix for Sortify holds.
- Publish release asset `magisk-boot-watch-v0.2.0.zip`.

## v0.1.5.1 - 2026-06-05

- Public baseline release as Boot Watch Collector.
- Verified baseline build still used legacy internal id `pixel-boot-watch`.

## v0.2.1 - 2026-06-05

- Fixes automatic boot export after the v0.2.0 module-id migration.
- Executes result-log-export.sh through /system/bin/sh and logs auto_export_rc.
- Hardens Magisk install permissions for module scripts.

## v0.2.2 - 2026-06-05

- Fixes active-module disable/remove marker regression after v0.2.1.
- Limits legacy cleanup to /data/adb/modules/pixel-boot-watch and clears accidental MODPATH markers.
- Keeps v0.2.1 robust auto-export and executable permission hardening.
