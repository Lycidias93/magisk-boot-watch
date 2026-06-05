# Boot Watch Collector

Magisk boot evidence collector for Android devices.

Boot Watch Collector collects bounded post-boot diagnostics such as Magisk module state, ANR/tombstone/dropbox evidence, logcat red flags, ART/Zygisk hints, storage and memory pressure, and AshLooper health. It exports protected readable result logs to Download and then exits.

- No daemon after the collection run finishes
- No network access
- No upload
- No cloud dependency
- Designed for rooted Android devices using Magisk

## Current public baseline

This public repository starts from the verified `v0.1.5.1` build that was tested on a Pixel device. The collector itself is intended to become broader than Pixel-only, but the first public baseline keeps the verified internal module id and runtime paths for compatibility.

Current package details:

```text
Public name: Boot Watch Collector
Repository: magisk-boot-watch
Current Magisk module id: pixel-boot-watch
Verified version: 0.1.5.1
Verified versionCode: 16
```

A future compatibility release may rename the Magisk module id to `boot-watch` after a separate migration/upgrade test.

## What it captures

The module runs once after boot and collects bounded evidence from areas that are useful when diagnosing post-boot instability:

- boot completion and stage timings
- Magisk module state
- ANR, tombstone and dropbox counts
- selected logcat red flags
- ART/dex2oat/Zygisk hints
- storage and memory pressure hints
- AshLooper health when present
- result marker and machine-readable `status.env`

## Output files

The collector writes protected local result files to Download:

```text
pixel_local__pixel-boot-watch-<run_id>-result.txt
pixel_local__pixel-boot-watch-last-result.txt
pixel_local__pixel-boot-watch-action-last-result.txt
pixel_local__pixel-boot-watch-status.env
```

The `pixel_local__` prefix is intentional. It marks the result files as local/protected for environments that automatically sort or archive files from Download.

## Install

Flash the Magisk ZIP from the GitHub Release assets:

```text
magisk-boot-watch-v0.1.5.1.zip
```

Reboot once after flashing.

## Verify

After reboot, open the module action or check Download for:

```text
pixel_local__pixel-boot-watch-status.env
```

Expected core status:

```text
pbw_result=PASS
pbw_version=0.1.5.1
pbw_versionCode=16
pbw_file_name_too_long=absent
pbw_file_name_too_long_count=0
pbw_protected_names=yes
pbw_sortify_hold_expected=yes
```

You can also run the packaged verifier against the ZIP:

```sh
bash tools/verify-magisk-boot-watch-v0.1.5.1.sh releases/v0.1.5.1/magisk-boot-watch-v0.1.5.1.zip
```

## Privacy notes

Boot Watch Collector is a diagnostic tool. Result logs can contain package names, process names, paths, stack traces, tombstone snippets and local device state. Review logs before sharing them publicly.

The module itself does not upload anything.

## License

No open-source license has been selected yet. Until a license is added, all rights are reserved by the repository owner.
