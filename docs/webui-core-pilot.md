# Shared WebUI core pilot

## Purpose

Boot Watch is the first real module migration pilot for the shared standalone
browser WebUI core. It is not the first Magisk module in this project.

The collector and its persistent runtime paths remain unchanged:

- module id: `boot-watch`
- runtime: `/data/adb/boot-watch`
- runs: `/data/adb/boot-watch/runs`
- protected exports: `/storage/emulated/0/Download/pixel_local__boot-watch-*`

Only the optional WebUI transport and rendering layer changes.

## Pinned core

- source: `Lycidias93/android-root-module-webui-template`
- core version: `0.2.2`
- source commit: `5b8e412428cd04b1cf98a1fc0e03269580b60d71`
- pilot module revision: `0.2.11-webui-core-pilot.2`

Core `0.2.2` includes the Pixel-proven Magisk BusyBox portability fix for UUID
token generation. The earlier `.1` pilot could install successfully but its
Action failed before server startup because `tr -d '-\n'` was parsed as an
invalid BusyBox option.

## Adapter contract

Boot Watch declares:

- config: disabled
- actions: disabled
- jobs: disabled
- logs: enabled
- inventory: enabled

The adapter exposes only fixed Boot Watch paths. It returns bounded, redacted
logs and typed inventories for recent protected result files and evidence-file
state. No browser request can select an arbitrary path or shell command.

## Migration delta

Removed:

- generated JSON snapshots from `webroot/`
- old WebUI exporter scripts
- the wrapper that refreshed JSON before running the original Action
- the module-specific static frontend

Added:

- secure standalone browser launcher
- native ARM64 loopback server
- capability-driven shared frontend
- read-only Boot Watch adapter
- pinned core lock file
- local and CI integration tests
- deterministic pilot package

## Rollback

Reinstall stable Boot Watch `0.2.10-webui-runtime-root-hotfix`. The collector
runtime and evidence under `/data/adb/boot-watch` are retained, so the rollback
does not require deleting diagnostic history.

## Device acceptance gate

The pilot must not be promoted or merged as a stable release until a Pixel test
proves:

1. battery and power preflight passes;
2. the module installs and reboots normally;
3. the boot collector still completes with its normal result marker;
4. Action opens the default browser;
5. the final browser URL no longer contains the bootstrap token;
6. the server listens only on loopback;
7. status, logs and run inventory match the collector runtime;
8. restarting Action replaces only the correct prior WebUI process;
9. WebUI failure does not affect collector startup;
10. rollback to stable preserves evidence.

The required installed-runtime completion marker is:

```text
RESULT: CG_INSTALLED_RUNTIME_VERIFY_DONE outcome=success workflow_exit_code=0
```
