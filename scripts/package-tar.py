#!/usr/bin/env python3
"""Create a deterministic gzip-compressed tar archive for a staged tree."""

from __future__ import annotations

import gzip
import pathlib
import sys
import tarfile


def main() -> None:
    if len(sys.argv) != 3:
        raise SystemExit("usage: package-tar.py SOURCE_DIR OUTPUT_ARCHIVE")
    source = pathlib.Path(sys.argv[1]).resolve()
    output = pathlib.Path(sys.argv[2]).resolve()
    root = source.name
    output.parent.mkdir(parents=True, exist_ok=True)
    with output.open("wb") as raw:
        with gzip.GzipFile(fileobj=raw, mode="wb", mtime=0) as compressed:
            with tarfile.open(fileobj=compressed, mode="w") as archive:
                paths = [source, *sorted(source.rglob("*"))]
                for path in paths:
                    relative = path.relative_to(source.parent)
                    info = archive.gettarinfo(str(path), arcname=relative.as_posix())
                    info.mtime = 0
                    info.uid = 0
                    info.gid = 0
                    info.uname = ""
                    info.gname = ""
                    if path.is_file():
                        with path.open("rb") as payload:
                            archive.addfile(info, payload)
                    else:
                        archive.addfile(info)


if __name__ == "__main__":
    main()
