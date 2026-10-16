"""Build every ScadaBench icon file from studio/branding/icon.svg (Windows, no extra packages).

    python studio/tools/build_icons.py
Edge (headless) renders a 1024 px master with a transparent background, Windows
System.Drawing resizes it, and this script packs the PNG frames into icon.ico.
Outputs are committed; run again only after changing icon.svg.
"""
import os
from pathlib import Path
import shutil
import struct
import subprocess
import tempfile

STUDIO = Path(__file__).resolve().parents[1]
ROOT = STUDIO.parent
SOURCE = STUDIO / "branding/icon.svg"
ICO_SIZES = (16, 20, 24, 32, 40, 48, 64, 128, 256)
APP_ASSETS = STUDIO / "frontend/public/assets/brand"
SITE_ASSETS = ROOT / "website/public"
EDGE_PATHS = ("ProgramFiles(x86)", "ProgramFiles")


def edge():
    for variable in EDGE_PATHS:
        candidate = Path(os.environ.get(variable, "")) / "Microsoft/Edge/Application/msedge.exe"
        if candidate.is_file():
            return candidate
    raise SystemExit("Microsoft Edge was not found; it is used to draw the SVG.")


def render_master(work, size=1024):
    page = work / "icon.html"
    page.write_text(f'<html><body style="margin:0;background:transparent"><img src="{SOURCE.as_uri()}" '
                    f'style="width:{size}px;height:{size}px;display:block"></body></html>', encoding="utf-8")
    master = work / "master.png"
    subprocess.run([str(edge()), "--headless=new", "--disable-gpu", "--hide-scrollbars",
                    "--default-background-color=00000000", f"--window-size={size},{size}",
                    f"--screenshot={master}", page.as_uri()], check=True, capture_output=True)
    if not master.is_file():
        raise SystemExit("Edge did not produce the icon image.")
    return master


def resize(master, sizes, work):
    """Write work/icon-<size>.png for each size with high-quality bicubic scaling."""
    script = (
        "Add-Type -AssemblyName System.Drawing;"
        f"$src=[Drawing.Image]::FromFile('{master}');"
        f"foreach($s in @({','.join(map(str, sizes))})){{"
        "$b=New-Object Drawing.Bitmap $s,$s;$g=[Drawing.Graphics]::FromImage($b);"
        "$g.InterpolationMode='HighQualityBicubic';$g.SmoothingMode='HighQuality';"
        "$g.PixelOffsetMode='HighQuality';$g.CompositingQuality='HighQuality';"
        "$g.DrawImage($src,0,0,$s,$s);$g.Dispose();"
        f"$b.Save('{work}\\icon-'+$s+'.png',[Drawing.Imaging.ImageFormat]::Png);$b.Dispose()}};$src.Dispose()")
    subprocess.run(["powershell", "-NoProfile", "-NonInteractive", "-Command", script], check=True)
    return {size: (work / f"icon-{size}.png").read_bytes() for size in sizes}


def ico_bytes(frames):
    """Pack PNG frames into an .ico (PNG-compressed entries, Vista and later)."""
    sizes = sorted(frames)
    header = struct.pack("<HHH", 0, 1, len(sizes))
    offset = 6 + 16 * len(sizes)
    entries, blobs = b"", b""
    for size in sizes:
        data = frames[size]
        dimension = 0 if size >= 256 else size
        entries += struct.pack("<BBBBHHII", dimension, dimension, 0, 0, 1, 32, len(data), offset)
        blobs += data
        offset += len(data)
    return header + entries + blobs


def main():
    with tempfile.TemporaryDirectory(prefix="scadabench-icons-") as temporary:
        work = Path(temporary)
        frames = resize(render_master(work), sorted(set(ICO_SIZES) | {180, 192, 512}), work)
        ico = ico_bytes({size: frames[size] for size in ICO_SIZES})
    (STUDIO / "branding/icon.ico").write_bytes(ico)
    APP_ASSETS.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(SOURCE, APP_ASSETS / "favicon.svg")
    (APP_ASSETS / "favicon-32.png").write_bytes(frames[32])
    (APP_ASSETS / "icon-180.png").write_bytes(frames[180])
    SITE_ASSETS.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(SOURCE, SITE_ASSETS / "favicon.svg")
    (SITE_ASSETS / "favicon.ico").write_bytes(ico)
    (SITE_ASSETS / "apple-touch-icon.png").write_bytes(frames[180])
    (SITE_ASSETS / "icon-192.png").write_bytes(frames[192])
    (SITE_ASSETS / "icon-512.png").write_bytes(frames[512])
    print(f"icon.ico ({len(ICO_SIZES)} sizes, {len(ico) // 1024} KiB), app favicons, website icons written.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
