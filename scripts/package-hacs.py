#!/usr/bin/env python3
"""Create a deterministic zip of the generated HACS repository tree."""

from __future__ import annotations

import pathlib
import sys
import zipfile


def main() -> None:
    if len(sys.argv) != 4:
        raise SystemExit("usage: package-hacs.py EXPORT_DIR OUTPUT VERSION")
    source = pathlib.Path(sys.argv[1]).resolve()
    output = pathlib.Path(sys.argv[2]).resolve()
    version = sys.argv[3]
    root_name = f"fruit-forwarder-ha-{version}"
    output.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for path in sorted(source.rglob("*")):
            if not path.is_file():
                continue
            relative = path.relative_to(source)
            info = zipfile.ZipInfo(f"{root_name}/{relative.as_posix()}")
            info.date_time = (1980, 1, 1, 0, 0, 0)
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = 0o100644 << 16
            archive.writestr(info, path.read_bytes())
    print(output)


if __name__ == "__main__":
    main()
