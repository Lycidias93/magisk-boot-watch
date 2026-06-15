# Boot Watch Collector v0.2.6

Stable diagnostics bundle release.

## Changes

- Adds AshLooper intervention reporting for modules disabled before Boot Watch can collect their runtime logs.
- Adds conservative `module_logs_missing_because_disabled` reporting instead of claiming root cause without explicit AshLooper proof.
- Adds bounded split logcat buffer captures.
- Adds pstore/ramoops snapshot collection when files exist.
- Adds focused dumpsys outputs for boot triage.
- Keeps collection local, bounded, and read-only.
- Keeps protected Pixel-local result file names for Sortify hold workflows.

## Validation

Validated on Pixel with installed `v0.2.6-test.1`:

- First post-reboot proof: PASS.
- Later post-reboot proof: PASS.
- `pbw_result=PASS`.
- `pbw_version=0.2.6-test.1`.
- `pbw_versionCode=27`.
- `pbw_file_name_too_long=absent`.
- `pbw_split_logcat_files=18`.
- `pbw_focused_dumpsys_files=10`.
- `pbw_ashlooper_intervention_possible=yes`.
- `pbw_module_logs_missing_because_disabled=yes`.

## Install

Flash `magisk-boot-watch-v0.2.6.zip` in Magisk and reboot.
