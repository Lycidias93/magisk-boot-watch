# Extended module runtime log collector test proof

Status: 2026-06-06
Baseline: main after PR #11 (`8d3cf1a`)
Scope: runtime verification of the vNext extended module runtime log collectors.

## Result

PASS.

The extended collector was temporarily deployed over the active Boot Watch v0.2.4 installation, executed with `PBW_PROFILE=extended`, verified, and then the active stable scripts were restored from backup.

## Verified signals

- `profile=extended`
- `module_runtime_logs_enabled=1`
- `verify_extended_runtime=pass`
- `db_leak=absent`
- archive size observed: `2042941` bytes
- run id observed: `20260606_135009_boot`

## Collected extended module runtime files

Frosty:

- `module_runtime/frosty/kernel_tweaks.log.txt`
- `module_runtime/frosty/ram.log.txt`
- `module_runtime/frosty/services.log.txt`

LSPosed/lspd rotated logs:

- `module_runtime/lsposed_log_old/kmsg.log.txt`
- `module_runtime/lsposed_log_old/props.txt.txt`
- `module_runtime/lsposed_log_old/modules_2026-06-06T08:53:11.034382.log.txt`
- `module_runtime/lsposed_log_old/verbose_2026-06-06T08:53:11.033165.log.txt`

Summary:

- `module_runtime/known_modules.txt`

## Privacy / risk guard

- No LSPosed SQLite database was collected.
- No `modules_config.db`, `modules_config.db-shm`, `modules_config.db-wal`, or `modules_config.db-journal` leaked into the archive.
- Collection stayed bounded to explicit runtime log files.
- Standard profile behavior remains unchanged.

## Restore proof

The active stable scripts were restored after the test:

- `/data/adb/modules/boot-watch/boot-watch.sh`
- `/data/adb/modules/boot-watch/result-log-export.sh`

The backup location was:

- `/data/adb/boot-watch/backups/vnext_module_runtime_logs_test_20260606_135009_8d3cf1a`

## Result markers

```text
RESULT: MAGISK_BOOT_WATCH_EXTENDED_RUNTIME_TEST_DONE rc=0
RESULT: MAGISK_BOOT_WATCH_EXTENDED_RUNTIME_TEST_RUN_DONE
RESULT: CHAT_BLOCK_DONE rc=0
```
