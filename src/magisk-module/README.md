# Pixel Boot Watch v0.1.5.1

Comprehensive one-shot post-boot evidence collector for rooted Pixel/Magisk systems.

- bounded local-only collector
- no daemon after completion
- readable Download result log after boot
- Magisk Action Button exports latest run
- no network/upload behavior


## v0.1.5.1 protected result logs

Readable result logs are written with the `pixel_local__` prefix so Sortify should treat them as Pixel-local protected hold files:

- `pixel_local__pixel-boot-watch-<run_id>-boot-result.txt`
- `pixel_local__pixel-boot-watch-last-result.txt`
- `pixel_local__pixel-boot-watch-action-last-result.txt`
- `pixel_local__pixel-boot-watch-status.env`

The boot archive remains `pixel-boot-watch-<run_id>.tar.gz`.

v0.1.5.1 also adds bounded AshLooper/AshReXcue health evidence and a compact machine-readable `status.env`.
