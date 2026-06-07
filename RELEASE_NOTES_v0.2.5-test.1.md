# Boot Watch Collector v0.2.5-test.1

v0.2.5-test.1 is a manual prerelease for the extended module runtime log collector.

## Added

- Opt-in extended module runtime log collection.
- Bounded Frosty log tails:
  - `/data/adb/modules/Frosty/logs/kernel_tweaks.log`
  - `/data/adb/modules/Frosty/logs/ram.log`
  - `/data/adb/modules/Frosty/logs/services.log`
- Bounded LSPosed/lspd rotated log tails:
  - `/data/adb/lspd/log.old/kmsg.log`
  - `/data/adb/lspd/log.old/props.txt`
  - latest `modules_*.log`
  - latest `verbose_*.log`

## Guards

- `standard` profile remains unchanged.
- Content collection is gated by `PBW_PROFILE=extended`, `PBW_PROFILE=debug`, or `PBW_COLLECT_MODULE_LOGS=1`.
- No LSPosed database dumps.
- No app-private data collection.
- No network upload and no telemetry.
- Stable update channel remains on v0.2.4 because this is a manual prerelease.

## Verified before release

- PR #11 implementation merged.
- PR #12 Pixel runtime proof documented.
- Runtime proof showed:
  - `profile=extended`
  - `module_runtime_logs_enabled=1`
  - `db_leak=absent`
  - `verify_extended_runtime=pass`
  - active stable scripts restored after test.

## Expected post-install check

After flashing manually and rebooting, expected values are:

```text
id=boot-watch
version=0.2.5-test.1
versionCode=25
pbw_version=0.2.5-test.1
pbw_versionCode=25
```

For extended collection, run with an explicit extended profile/test flow; normal boot collection should remain standard unless configured otherwise.
