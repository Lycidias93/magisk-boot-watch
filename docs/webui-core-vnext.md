# Boot Watch shared WebUI Core vNext

## Pin

- repository: `Lycidias93/android-root-module-webui-template`
- core: `0.6.3`
- commit: `6791a05be79f162979c76a286f7cdbdd9ce1cc6b`
- manifest SHA-256: `94600c81b15571571e175f8a16e92177a77269541165f19886c86c0c332e1119`
- upstream Core 0.6.3 CI: GitHub Actions run `34037562507` PASS on the merged PR head

The `webui-core/core-source-blobs-*.tsv` manifests bind every synchronized
consumer file to its exact upstream Git blob. `tools/verify-webui-core-pin.py`
recomputes those Git blob identities during verification.

## Classification

The LSPosed/Vector presentation is `module_specific`. Existing generic Core
summary cards, logs and typed inventories are sufficient, so no Boot-Watch-only
logic is added to the shared template.

## Security boundary

- loopback only;
- one-time bootstrap token exchanged for an HttpOnly session cookie;
- no token in server argv;
- no generic shell or arbitrary-path API;
- config/actions/jobs disabled by the Boot Watch adapter;
- no boot-time WebUI server;
- no LSPosed config contents or manager app-private cache exposed.

## Release gate

`webui_release_audit_required=yes`. Repository acceptance combines exact Core
blob pinning, the upstream successful Core CI, consumer static release audit and
real loopback HTTP integration. Stable publication additionally requires the
exact installed candidate device audit.

## Rollback

Reinstall stable Boot Watch v0.2.10. Persistent evidence under
`/data/adb/boot-watch` remains intact.
