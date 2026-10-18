"""Build the installer pictures from studio/branding/icon.svg (Windows, no extra packages).

    python studio/tools/build_installer_art.py
Edge (headless) draws the welcome panel and the small header icon once at high resolution;
Windows System.Drawing writes the 24-bit BMP sizes Inno Setup picks per display scaling
(100 % to 250 %). Outputs in studio/branding/installer/ are committed; run again only after
changing the brand. Used by packaging/installer.iss (change 0090; the portable self-extractor was withdrawn in 0092).
"""
from pathlib import Path
import subprocess
import tempfile

from build_icons import SOURCE, STUDIO, edge

OUT = STUDIO / "branding/installer"
# Inno Setup 6 sizes for 100, 125, 150, 175, 200, 225 and 250 % scaling.
PANEL_SIZES = ((164, 314), (192, 386), (246, 459), (273, 556), (328, 628), (355, 669), (410, 797))
SMALL_SIZES = ((55, 55), (64, 68), (83, 80), (92, 97), (110, 106), (119, 123), (138, 140))
PANEL_MASTER = (820, 1594)
SMALL_MASTER = (552, 560)

PANEL_HTML = """<html><body style="margin:0;width:820px;height:1594px;overflow:hidden;
background:linear-gradient(160deg,#16274a 0%,#0b1630 55%,#070d1a 100%);font-family:'Segoe UI',Arial,sans-serif;
color:#e8eef7;display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center">
<img src="{icon}" style="width:300px;height:300px">
<div style="margin-top:56px;font-size:96px;font-weight:700;letter-spacing:-1px">ScadaBench</div>
<div style="width:120px;height:10px;border-radius:5px;background:#2be4b0;margin:40px 0"></div>
<div style="font-size:46px;line-height:1.35;color:#b8c6dc;padding:0 90px">IEC 61850 IED simulator<br>&amp; SCADA test bench</div>
<div style="margin-top:120px;font-size:36px;line-height:1.6;color:#7f93b3">IEC 61850 &middot; GOOSE &middot; SV<br>
IEC 104 &middot; DNP3 &middot; Modbus</div></body></html>"""

SMALL_HTML = """<html><body style="margin:0;width:552px;height:560px;background:#ffffff;display:flex;
align-items:center;justify-content:center"><img src="{icon}" style="width:480px;height:480px"></body></html>"""


def render(html, size, work, name):
    page = work / f"{name}.html"
    page.write_text(html.format(icon=SOURCE.as_uri()), encoding="utf-8")
    image = work / f"{name}.png"
    subprocess.run([str(edge()), "--headless=new", "--disable-gpu", "--hide-scrollbars",
                    f"--window-size={size[0]},{size[1]}", f"--screenshot={image}", page.as_uri()],
                   check=True, capture_output=True)
    if not image.is_file():
        raise SystemExit(f"Edge did not produce {name}.png")
    return image


def write_bmps(master, sizes, prefix):
    """Scale to cover each size (centre crop keeps the proportions) and save 24-bit BMP."""
    pairs = ",".join(f"@({width},{height})" for width, height in sizes)
    script = (
        "Add-Type -AssemblyName System.Drawing;"
        f"$src=[Drawing.Image]::FromFile('{master}');"
        f"foreach($p in @({pairs})){{$w=$p[0];$h=$p[1];"
        "$k=[Math]::Max($w/$src.Width,$h/$src.Height);$dw=$src.Width*$k;$dh=$src.Height*$k;"
        "$b=New-Object Drawing.Bitmap $w,$h,([Drawing.Imaging.PixelFormat]::Format24bppRgb);"
        "$g=[Drawing.Graphics]::FromImage($b);$g.InterpolationMode='HighQualityBicubic';"
        "$g.SmoothingMode='HighQuality';$g.PixelOffsetMode='HighQuality';"
        "$g.DrawImage($src,($w-$dw)/2,($h-$dh)/2,$dw,$dh);$g.Dispose();"
        f"$b.Save('{OUT}\\{prefix}-'+$w+'x'+$h+'.bmp',[Drawing.Imaging.ImageFormat]::Bmp);$b.Dispose()}};"
        "$src.Dispose()")
    subprocess.run(["powershell", "-NoProfile", "-NonInteractive", "-Command", script], check=True)


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="scadabench-installer-art-") as temporary:
        work = Path(temporary)
        write_bmps(render(PANEL_HTML, PANEL_MASTER, work, "panel"), PANEL_SIZES, "wizard")
        write_bmps(render(SMALL_HTML, SMALL_MASTER, work, "small"), SMALL_SIZES, "small")
    print(f"{len(PANEL_SIZES)} welcome panels and {len(SMALL_SIZES)} header icons written to {OUT}.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
