from __future__ import annotations

import struct
from pathlib import Path


ICON_SIZES = (16, 32, 48, 64, 128, 256)


def main() -> None:
    app_dir = Path(__file__).resolve().parents[1]
    repo_dir = app_dir.parent
    brand_dir = repo_dir / "docs" / "brand" / "assets" / "png"
    output_path = app_dir / "build" / "windows" / "icon.ico"

    entries: list[tuple[int, bytes]] = []
    for size in ICON_SIZES:
        source_dir = brand_dir / ("favicons" if size <= 64 else "app-icons")
        source_path = source_dir / f"nexusdesk-app-icon-{size}.png"
        entries.append((size, source_path.read_bytes()))

    header_size = 6
    directory_size = 16 * len(entries)
    image_offset = header_size + directory_size

    output = bytearray()
    output += struct.pack("<HHH", 0, 1, len(entries))

    payload = bytearray()
    for size, data in entries:
        width = 0 if size == 256 else size
        height = 0 if size == 256 else size
        output += struct.pack(
            "<BBBBHHII",
            width,
            height,
            0,
            0,
            1,
            32,
            len(data),
            image_offset + len(payload),
        )
        payload += data

    output += payload
    output_path.write_bytes(output)
    print(f"Wrote {output_path} with {len(entries)} sizes")


if __name__ == "__main__":
    main()
