# Boot Watch Collector v0.2.3: service marker guard

v0.2.3 fixes the service-time active-marker regression that remained in v0.2.2.

## Fixed

- `service.sh` now treats only `/data/adb/modules/pixel-boot-watch` as the legacy module path.
- `/data/adb/modules/boot-watch` is treated as the active module and is only self-healed by removing accidental `disable` and `remove` markers.
- Install-time and service-time marker cleanup both avoid disabling or removing the active module.
- The robust boot auto-export hook from v0.2.1/v0.2.2 remains intact.
