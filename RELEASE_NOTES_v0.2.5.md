# Boot Watch Collector v0.2.5

Stable release for the opt-in extended module runtime log collector previously validated as `v0.2.5-test.1`.

## Added

- Stable online update metadata for `v0.2.5` with `versionCode=26`.
- Opt-in extended module runtime log collection:
  - Frosty bounded tails: `kernel_tweaks.log`, `ram.log`, and `services.log`.
  - LSPosed/lspd rotated bounded tails: `kmsg.log`, `props.txt`, latest `modules_*.log`, and latest `verbose_*.log`.

## Guards

- `standard` profile remains bounded and unchanged for foreign module log contents.
- Extended runtime log collection stays gated by `PBW_PROFILE=extended`, `PBW_PROFILE=debug`, or `PBW_COLLECT_MODULE_LOGS=1`.
- No LSPosed database dumps.
- No app-private data collection.
- No network upload and no telemetry.

## Installed validation

Promoted after Pixel installed validation documented in `docs/v0.2.5-test.1-post-reboot-and-extended-proof-20260608.md`:

```text
standard_post_reboot=PASS
installed_extended=PASS
version=0.2.5-test.1
versionCode=25
profile=extended
module_runtime_logs_enabled=1
frosty_log_files=3
lsposed_old_log_files=4
db_leak=absent
file_name_too_long=absent
```

## Update path

`update.json` now points the stable channel to:

```text
version=0.2.5
versionCode=26
zip=magisk-boot-watch-v0.2.5.zip
```
