# WebUI credits and provenance

Boot Watch Collector consumes the shared standalone WebUI Core from:

- `Lycidias93/android-root-module-webui-template`
- core version `0.6.3`
- pinned source commit `6791a05be79f162979c76a286f7cdbdd9ce1cc6b`
- core manifest SHA-256 `94600c81b15571571e175f8a16e92177a77269541165f19886c86c0c332e1119`

The synchronized common core retains its upstream design/license provenance.
Retained third-party license texts are packaged under `third_party/licenses/`.

Boot Watch-specific collector, status, log and inventory semantics remain in
this repository. The LSPosed/Vector support layer was informed by the public
MIT project `sojiagu/lsposed-bugreport`; no source files from that collector are
vendored into Boot Watch.
