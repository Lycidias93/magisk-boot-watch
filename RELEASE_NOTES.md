# Boot Watch Collector v0.1.5.1

First public baseline for the Magisk boot evidence collector.

## Highlights

- Bounded post-boot diagnostics for rooted Android/Magisk devices
- Protected `pixel_local__pixel-boot-watch-*` result files in Download
- Machine-readable `status.env`
- Action export with before/after state
- Bounded AshLooper health evidence when present
- Fixed `file_name_too_long` status/count reporting

## Verified baseline

```text
pbw_result=PASS
pbw_version=0.1.5.1
pbw_versionCode=16
pbw_file_name_too_long=absent
pbw_file_name_too_long_count=0
pbw_protected_names=yes
pbw_sortify_hold_expected=yes
```

## Compatibility note

The public repository name is `magisk-boot-watch` and the public display name is `Boot Watch Collector`.

The current verified ZIP still uses the legacy internal Magisk module id `pixel-boot-watch`. A future release may migrate to `boot-watch` after a separate migration test.
