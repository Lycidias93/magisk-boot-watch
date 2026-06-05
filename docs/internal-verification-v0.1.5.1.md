# Pixel Boot Watch v0.1.5.1 Runbook

## Goal

Release Pixel Boot Watch v0.1.5.1 after post-reboot proof.

## Verified Pixel runtime

```text
run_id=20260605_105319_boot
pbw_result=PASS
pbw_version=0.1.5.1
pbw_versionCode=16
pbw_file_name_too_long=absent
pbw_file_name_too_long_count=0
pbw_protected_names=yes
pbw_sortify_hold_expected=yes
pbw_ashlooper_present=yes
pbw_ashlooper_version=9.8
RESULT: PIXEL_BOOT_WATCH_ACTION_EXPORT_DONE rc=0
RESULT: PIXEL_BOOT_WATCH_V0151_POST_REBOOT_CHECK_DONE rc=0
```

## Protected output files

```text
/storage/emulated/0/Download/pixel_local__pixel-boot-watch-<run_id>-result.txt
/storage/emulated/0/Download/pixel_local__pixel-boot-watch-last-result.txt
/storage/emulated/0/Download/pixel_local__pixel-boot-watch-action-last-result.txt
/storage/emulated/0/Download/pixel_local__pixel-boot-watch-status.env
```

## Release gates

- `module.prop` reports `version=0.1.5.1` and `versionCode=16`.
- `status.env` reports `pbw_result=PASS`.
- Protected names and Sortify hold expectation are `yes`.
- File-name-too-long count is `0` and status is `absent`.
- AshLooper health is present and bounded.
- No lingering `pixel-boot-watch` process remains.

## Out of scope

No Zygisk/ART, SafetyCore, Thermal, DNS/HA/VIP/route or Sortify code changes.
