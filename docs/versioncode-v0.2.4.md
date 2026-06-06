# Boot Watch Collector v0.2.4: versionCode metadata fix

v0.2.4 fixes the release metadata inconsistency that remained in v0.2.3.

## Fixed

- `boot-watch.sh` now reports `VERSION_CODE=24` in result marker and `status.env`.
- `module.prop`, `update.json`, runtime marker, and exported `pbw_versionCode` are aligned.
- The v0.2.3 service-time active marker guard remains unchanged.
- The robust boot auto-export hook remains intact.
