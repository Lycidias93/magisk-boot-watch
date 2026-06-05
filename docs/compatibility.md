# Compatibility

The first public baseline is verified as `v0.1.5.1` on a Pixel device with Magisk.

The project is intended to grow beyond Pixel devices, but the first public release keeps the verified module id and runtime paths:

```text
id=pixel-boot-watch
runtime=/data/adb/pixel-boot-watch
```

A future migration release may move to:

```text
id=boot-watch
runtime=/data/adb/boot-watch
```

That rename needs a dedicated migration and upgrade test before release.
