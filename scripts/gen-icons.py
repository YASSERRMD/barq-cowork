#!/usr/bin/env python3
"""
gen-icons.py — Generate placeholder app icons for Barq Cowork.

Usage:
    python3 scripts/gen-icons.py

Requirements:
    pip install Pillow

The script generates a lightning bolt icon in Barq's brand palette and
writes all required sizes to apps/desktop/src-tauri/icons/.
Replace these placeholders with professionally designed artwork before
publishing a release.
"""

import os
import sys
import struct
import zlib
import math

ICONS_DIR = os.path.join(
    os.path.dirname(__file__), "..", "apps", "desktop", "src-tauri", "icons"
)

# Brand colours
BG       = (18,  18,  27,  255)   # near-black
ACCENT   = (249, 115,  22, 255)   # orange-500
GLOW     = (251, 146,  60, 180)   # orange-400 semi-transparent
WHITE    = (255, 255, 255, 255)
TRANSP   = (0,   0,   0,   0)


def _write_chunk(chunk_type: bytes, data: bytes) -> bytes:
    chunk = chunk_type + data
    crc   = zlib.crc32(chunk) & 0xFFFFFFFF
    return struct.pack(">I", len(data)) + chunk + struct.pack(">I", crc)


def create_rgba_png(width: int, height: int, filepath: str) -> None:
    """Render a lightning bolt icon and save as RGBA PNG."""
    sig  = b"\x89PNG\r\n\x1a\n"
    ihdr = _write_chunk(b"IHDR", struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0))

    cx, cy = width / 2, height / 2
    # Bolt vertices (normalised 0-1, then scaled)
    bolt = [
        (0.50, 0.05), (0.28, 0.52), (0.48, 0.52),
        (0.35, 0.95), (0.72, 0.42), (0.52, 0.42),
    ]
    bolt_px = [(x * width, y * height) for x, y in bolt]

    def point_in_polygon(px, py, polygon):
        n     = len(polygon)
        inside = False
        j     = n - 1
        for i in range(n):
            xi, yi = polygon[i]
            xj, yj = polygon[j]
            if ((yi > py) != (yj > py)) and (px < (xj - xi) * (py - yi) / (yj - yi) + xi):
                inside = not inside
            j = i
        return inside

    rows = []
    for y in range(height):
        row = bytearray(b"\x00")
        for x in range(width):
            dx, dy  = x - cx, y - cy
            dist    = math.hypot(dx, dy)
            r_outer = min(width, height) * 0.46
            r_inner = min(width, height) * 0.42

            if dist > r_outer:
                row += bytearray(TRANSP)
            elif dist > r_inner:
                # Circular border
                t    = (dist - r_inner) / (r_outer - r_inner)
                a    = int(255 * (1 - t))
                row += bytearray((ACCENT[0], ACCENT[1], ACCENT[2], a))
            elif point_in_polygon(x, y, bolt_px):
                row += bytearray(ACCENT)
            else:
                row += bytearray(BG)
        rows.append(bytes(row))

    compressed = zlib.compress(b"".join(rows), 9)
    idat = _write_chunk(b"IDAT", compressed)
    iend = _write_chunk(b"IEND", b"")

    os.makedirs(os.path.dirname(filepath), exist_ok=True)
    with open(filepath, "wb") as fh:
        fh.write(sig + ihdr + idat + iend)
    print(f"  ✓ {os.path.basename(filepath)}  ({width}×{height})")


def create_ico(png_32_path: str, ico_path: str) -> None:
    """Wrap the 32×32 PNG inside a minimal .ico container."""
    with open(png_32_path, "rb") as fh:
        png_data = fh.read()

    # ICO header (reserved=0, type=1, count=1)
    header  = struct.pack("<HHH", 0, 1, 1)
    # Directory entry for a 32×32 PNG image
    img_size   = len(png_data)
    img_offset = 6 + 16          # header + one directory entry
    entry = struct.pack("<BBBBHHII", 32, 32, 0, 0, 1, 32, img_size, img_offset)
    with open(ico_path, "wb") as fh:
        fh.write(header + entry + png_data)
    print(f"  ✓ {os.path.basename(ico_path)}")


def create_icns(png_128_path: str, icns_path: str) -> None:
    """Wrap a 128×128 PNG inside a minimal .icns container (ic07 type)."""
    with open(png_128_path, "rb") as fh:
        png_data = fh.read()

    # Each ICNS chunk: 4-byte OSType + 4-byte (chunk length including header) + data
    icns_type   = b"ic07"           # 128×128 PNG
    chunk_len   = 8 + len(png_data)
    chunk       = icns_type + struct.pack(">I", chunk_len) + png_data
    file_len    = 8 + len(chunk)    # 'icns' magic + total size + chunks
    header      = b"icns" + struct.pack(">I", file_len)
    with open(icns_path, "wb") as fh:
        fh.write(header + chunk)
    print(f"  ✓ {os.path.basename(icns_path)}")


def main() -> None:
    d = os.path.abspath(ICONS_DIR)
    print(f"Generating icons → {d}")

    p32  = os.path.join(d, "32x32.png")
    p128 = os.path.join(d, "128x128.png")
    p256 = os.path.join(d, "128x128@2x.png")
    ico  = os.path.join(d, "icon.ico")
    icns = os.path.join(d, "icon.icns")

    create_rgba_png(32,  32,  p32)
    create_rgba_png(128, 128, p128)
    create_rgba_png(256, 256, p256)
    create_ico(p32, ico)
    create_icns(p128, icns)

    print("\nDone. Replace these placeholders with professional artwork before release.")


if __name__ == "__main__":
    main()
