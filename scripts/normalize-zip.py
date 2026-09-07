#!/usr/bin/env python3
"""Normalize ZIP entry metadata while preserving payloads and executable modes."""

from __future__ import annotations

import os
import pathlib
import sys
import tempfile
import zipfile


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("usage: normalize-zip.py ARCHIVE")
    archive = pathlib.Path(sys.argv[1]).resolve()
    fd, temporary = tempfile.mkstemp(prefix=f".{archive.name}.", dir=archive.parent)
    os.close(fd)
    temporary_path = pathlib.Path(temporary)
    try:
        with zipfile.ZipFile(archive, "r") as source, zipfile.ZipFile(
            temporary_path, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9
        ) as target:
            for original in sorted(source.infolist(), key=lambda item: item.filename):
                normalized = zipfile.ZipInfo(original.filename)
                normalized.date_time = (1980, 1, 1, 0, 0, 0)
                normalized.compress_type = zipfile.ZIP_DEFLATED
                normalized.external_attr = original.external_attr
                normalized.create_system = original.create_system
                normalized.flag_bits = original.flag_bits
                target.writestr(normalized, source.read(original))
        os.replace(temporary_path, archive)
    finally:
        temporary_path.unlink(missing_ok=True)


if __name__ == "__main__":
    main()
