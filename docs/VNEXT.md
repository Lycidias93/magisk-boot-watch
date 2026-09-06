# Boot Watch vNext

Status: repository candidate implemented 2026-09-06.

## Candidate

- module: `0.2.11-vnext.1`
- versionCode: `36`
- shared WebUI Core: `0.6.3`
- core pin: `6791a05be79f162979c76a286f7cdbdd9ce1cc6b`
- stable update channel: unchanged on v0.2.10 until device acceptance

## Collector delta

The `zygisk_stack_support` stage is bounded and read-only. It detects common
LSPosed and Vector module aliases, lspd service/runtime state, current and
rotated log counts, and LSPosed config file counts. It does not read LSPosed
config contents. Manager app-private caches are excluded.

`magisk --denylist ls` is privacy-gated behind `extended` / `debug` or explicit
`PBW_COLLECT_ZYGISK_SUPPORT=1`. Standard mode records metadata only.

## WebUI delta

Boot Watch consumes the shared standalone WebUI Core instead of maintaining a
separate generated-JSON frontend. The module adapter exposes only status,
bounded redacted logs, protected result history, evidence-file inventory and
LSPosed / Vector / Zygisk-stack inventory.

Config, actions and jobs are disabled. The WebUI server is user-triggered,
loopback-only and not a boot dependency.

## Source comparison

`sojiagu/lsposed-bugreport` was used as a design comparison for LSPosed/Vector
path discovery and maintainer-oriented support evidence. Boot Watch deliberately
does not adopt its broad config/module/cache directory-copy model.

## Acceptance

Repository/static/integration checks must pass on exact candidate bytes. Stable
release remains blocked until the candidate is installed on Pixel and both the
post-reboot collector proof and exact-device WebUI audit pass. Any candidate
rebuild invalidates that device audit.
