#!/usr/bin/env python3
from pathlib import Path
import os, stat, sys, zipfile
if len(sys.argv)!=3: print(f'Usage: {sys.argv[0]} <source-directory> <output.zip>',file=sys.stderr); sys.exit(2)
source=Path(sys.argv[1]).resolve(); output=Path(sys.argv[2]).resolve()
if not source.is_dir(): print(f'FAIL source_not_directory path={source}',file=sys.stderr); sys.exit(1)
output.parent.mkdir(parents=True,exist_ok=True); temporary=output.with_suffix(output.suffix+'.tmp')
if temporary.exists(): temporary.unlink()
files=sorted(p for p in source.rglob('*') if p.is_file())
with zipfile.ZipFile(temporary,'w',compression=zipfile.ZIP_DEFLATED,compresslevel=9,strict_timestamps=True) as z:
    for path in files:
        if path.is_symlink(): print(f'FAIL symlink_not_allowed path={path}',file=sys.stderr); sys.exit(1)
        relative=path.relative_to(source).as_posix(); mode=stat.S_IMODE(path.stat().st_mode)
        info=zipfile.ZipInfo(relative,date_time=(1980,1,1,0,0,0)); info.create_system=3; info.external_attr=(stat.S_IFREG|mode)<<16; info.compress_type=zipfile.ZIP_DEFLATED; info.flag_bits|=0x800
        z.writestr(info,path.read_bytes(),compress_type=zipfile.ZIP_DEFLATED,compresslevel=9)
os.replace(temporary,output); print(f'RESULT: DETERMINISTIC_PACKAGE_PASS files={len(files)} output={output}')
