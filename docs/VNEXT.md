# Boot Watch vNext roadmap

Status: 2026-08-02

This roadmap separates historical ideas, already delivered collector work and
new WebUI follow-ups. The stable collector remains read-only and bounded.

## Already delivered

The earlier `log-coverage-vnext.md` plan has partly landed:

- bounded extended module-runtime logs;
- split logcat buffers;
- pstore/ramoops snapshot;
- focused dumpsys summaries;
- AshLooper intervention reporting;
- protected exports and run history;
- read-only WebUI status and logs.

These items should be documented as current behavior rather than repeatedly
listed as future work.

## Current pilot

### Shared WebUI core

- replace generated JSON snapshots with live typed APIs;
- keep the collector independent from the browser server;
- expose only read-only logs and inventories;
- validate Action → browser → session on a real Pixel;
- feed only domain-neutral findings back into the shared template.

## Collector vNext candidates

Priority order:

1. `dumpsys dropbox` index and bounded tag summary;
2. wakeup-source and suspend-blocker snapshots;
3. focused process/service/provider state;
4. focused memory summary;
5. bounded jobscheduler, alarm and device-idle summaries;
6. SurfaceFlinger, WindowManager and input summaries;
7. conservative combined Zygisk-stack health;
8. opt-in app/package focus with explicit privacy gate.

## Explicit non-goals

- no automatic full bugreport;
- no network upload or telemetry;
- no unrestricted recursive copies;
- no app-private data by default;
- no automatic module enable/disable or repair;
- no deep LSPosed database dump;
- no DNS, route, VPN or root-manager mutation.

## WebUI vNext after device proof

Only after the pilot is installed and reboot-verified:

- stale-data age indicators;
- selectable allowlisted log sources;
- a richer run-detail inventory;
- reusable empty/loading/error states;
- optional export-file open/share intents;
- localization and theme tokens;
- accessibility review on small screens;
- session restart and browser-back behavior tests.

## Shared core vNext candidates

Cross-module candidates, not Boot Watch-only features:

- cancellable jobs;
- typed readiness reasons;
- standardized diagnostics bundle metadata;
- reusable confirmation and preview components;
- stale core-lock detection in CI;
- optional manager-host renderer without making a companion app mandatory;
- additional ABIs only when a real supported device requires them.

Each item enters the core only after at least two modules need it or the Boot
Watch pilot proves it is transport/security infrastructure rather than domain
logic.
