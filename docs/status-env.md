# status.env

The module exports a machine-readable status file to Download:

```text
pixel_local__pixel-boot-watch-status.env
```

Core fields:

```text
pbw_result=PASS
pbw_version=<version>
pbw_versionCode=<versionCode>
pbw_run_id=<run_id>
pbw_mode=boot|action
pbw_archive_path=<path>
pbw_result_path=<path>
pbw_last_result_path=<path>
pbw_action_result_path=<path>
pbw_status_env_path=<path>
pbw_file_name_too_long=absent|present
pbw_file_name_too_long_count=<number>
pbw_protected_names=yes
pbw_sortify_hold_expected=yes
```
