# Boot Watch Collector

> **Development source candidate:** `0.2.11-vnext.1` / versionCode `36` is
> repository scope only. Stable `update.json` remains on
> `0.2.10-webui-runtime-root-hotfix` until exact-device post-reboot and WebUI
> release audits pass. The candidate pins shared WebUI Core `0.6.3` at
> `6791a05be79f162979c76a286f7cdbdd9ce1cc6b`.
**Magisk module for Android boot evidence collection, protected result exports, read-only WebUI status, and boot-crash handoff bundles.**

Stable **0.2.10-webui-runtime-root-hotfix** fixes the WebUI run-history exporter after the module moved to the current boot-watch runtime and protected result-file scheme. It is runtime-verified on Pixel after reboot with active old-prefix count 0, a PASS boot run, and populated WebUI run history.

[Download latest release](https://github.com/Lycidias93/magisk-boot-watch/releases/latest) · [Telegram](https://t.me/lycidias93) · [Issues](https://github.com/Lycidias93/magisk-boot-watch/issues) · [Release notes](RELEASE_NOTES.md) · [Changelog](CHANGELOG.md) · [Stable update metadata](update.json)

---

## vNext source candidate

The vNext candidate adds a bounded LSPosed / Vector support layer informed by
`sojiagu/lsposed-bugreport` without adopting broad directory copies:

- detects `zygisk_lsposed`, `LSPosed`, `vector` and `zygisk_vector` aliases;
- records lspd service/runtime state and bounded current/rotated log metadata;
- counts LSPosed config files but never reads or exports their contents;
- excludes LSPosed Manager app-private caches;
- collects `magisk --denylist ls` only in `extended` / `debug` or explicit
  `PBW_COLLECT_ZYGISK_SUPPORT=1` mode;
- exports machine-readable Zygisk-stack state through protected `pixel_local__*`
  results;
- exposes the same bounded state in a read-only shared WebUI Core inventory.

The WebUI is loopback-only, user-triggered and not a boot dependency.

## What this module does

Boot Watch Collector installs a guarded boot-time evidence collector through Magisk.

Main functions:

- **Boot evidence collection:** collects focused boot diagnostics after Android reaches boot-complete and stores each run under /data/adb/boot-watch/runs/<run_id>_boot.
- **Protected result exports:** writes human-readable summaries and archives to Android Download using the current protected names pixel_local__boot-watch-*.
- **Read-only WebUI:** exposes status, logs, and run history through bundled WebUI assets under webroot/.
- **Action refresh path:** the module Action refreshes WebUI status and result visibility without changing device routing, DNS, boot image, or root configuration.
- **Migration guard:** keeps the active module id boot-watch and runtime path /data/adb/boot-watch while avoiding legacy protected result-file names.

It is **not** a bootloop fixer, root manager, SafetyNet/Play Integrity tweak, SELinux policy patch, or thermal/performance module.

---

## Stable v0.2.10 highlights

| Area | v0.2.10 |
|---|---|
| WebUI run history | Reads current pixel_local__boot-watch-<run_id>-result.txt files |
| Result-file prefix | Removes active dependency on the old protected naming scheme |
| Source reproducibility | Tracks tools/ exporters and webroot/ assets in the module source |
| Runtime artifacts | Commits build-time skeleton JSON only, not live Base64/log snapshots |
| Pixel runtime | Post-reboot PASS with active version 0.2.10-webui-runtime-root-hotfix |
| Version code | 35 |

---

## Runtime and verification status

Runtime PASS:

- Active module: 0.2.10-webui-runtime-root-hotfix
- Active old-prefix count: 0
- Service launch marker: version=0.2.10-webui-runtime-root-hotfix
- Latest verified boot run: 20260707_174749_boot
- Boot result marker: RESULT: BOOT_WATCH_BOOT_DONE rc=0
- WebUI run history lists current pixel_local__boot-watch-* result files

Runtime paths:

| Path | Purpose |
|---|---|
| /data/adb/modules/boot-watch | active Magisk module |
| /data/adb/boot-watch | collector runtime |
| /data/adb/boot-watch/runs | per-boot evidence runs |
| /data/adb/modules/boot-watch/webroot | read-only WebUI files and generated JSON |
| /storage/emulated/0/Download/pixel_local__boot-watch-* | protected user-visible result files |

---

## Compatibility

| Device / root setup | Status |
|---|---|
| Pixel with Magisk-compatible module manager | Runtime verified |
| KernelSU / APatch style managers | May work, but report manager/version and install path when debugging |
| Unsupported or heavily modified ROMs | Evidence useful, but runtime behavior must be verified per device |

A PASS on one device does **not** automatically verify every root manager, ROM, or Android build.

---

## Install

### Requirements

- Magisk-compatible module installation path.
- Working root shell for manual verification.
- Reboot after install/update.
- Wait several minutes after boot before judging a boot run.

### Install/update

1. Download the latest ZIP from [Releases](https://github.com/Lycidias93/magisk-boot-watch/releases/latest).
2. Install it in Magisk or your compatible module manager.
3. Reboot.
4. Wait until the collector finishes.
5. Open the module Action/WebUI or run the verification commands below.

Stable update channel: [update.json](update.json)

Verify module identity:

    su -c 'cat /data/adb/modules/boot-watch/module.prop'

Expected healthy markers:

    id=boot-watch
    version=0.2.10-webui-runtime-root-hotfix
    versionCode=35

Check the latest service/run markers:

    su -c 'tail -120 /data/adb/boot-watch/service-launch.log'
    su -c 'find /data/adb/boot-watch/runs -maxdepth 1 -type d -name "*_boot" | sort | tail -5'

---

## WebUI and Action

The module includes a read-only WebUI and Action wrapper.

The WebUI displays:

- current status JSON
- recent boot/result logs
- run history from protected pixel_local__boot-watch-* result files

The Action path refreshes the generated WebUI JSON. It does not disable modules, patch boot images, change DNS/routes, or alter root-manager settings.

Generated user-visible files:

    /storage/emulated/0/Download/pixel_local__boot-watch-<run_id>-result.txt
    /storage/emulated/0/Download/pixel_local__boot-watch-last-result.txt
    /storage/emulated/0/Download/pixel_local__boot-watch-action-last-result.txt
    /storage/emulated/0/Download/pixel_local__boot-watch-status.env
    /storage/emulated/0/Download/boot-watch-<run_id>.tar.gz

---

## Reporting issues

When reporting boot or WebUI issues, include:

- device model and codename
- Android version, build ID, incremental and fingerprint
- root solution and version
- module version and install/update path
- latest Boot Watch run id
- result text file and archive, if safe to share
- Magisk install log or screenshot when relevant

Review generated output before posting it publicly.

Do not post raw tokens, private hostnames, private IPs, MAC addresses, personal paths, unrelated logs, or files from unrelated apps.

---

## Safety notes

Boot Watch Collector intentionally does **not**:

- change DNS, routes, VPN, MagicDNS, subnet routes, or firewall policy
- change thermal, CPU, memory, battery, or performance settings
- disable or enable other Magisk modules
- patch boot images
- change SELinux policy
- remove runtime evidence automatically

The module prefers explicit evidence collection over risky automatic repair.
