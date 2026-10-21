"""Windows icon + version resources for the Go release executables (ADR-0083).

Go links resources from a COFF object named *.syso in the package folder. The Windows SDK
resource compiler (rc.exe) and MSVC's cvtres.exe make that object; cvtres also adds two
absolute marker symbols (@comp.id, @feat.00) that Go's linker rejects ("sectnum < 0"). They
carry no resource data, so they are pointed at the object's .debug$S section, which the
linker ignores. Nothing else in the object is changed.
"""
from pathlib import Path
import os
import struct
import subprocess
import tempfile

SYMBOL_SIZE = 18
SECTION_HEADER_SIZE = 40
IMAGE_SYM_ABSOLUTE = -1

RC_TEMPLATE = """1 ICON "{icon}"
1 VERSIONINFO
FILEVERSION {numbers}
PRODUCTVERSION {numbers}
FILEOS 0x40004
FILETYPE 0x1
BEGIN
  BLOCK "StringFileInfo"
  BEGIN
    BLOCK "040904B0"
    BEGIN
      VALUE "CompanyName", "ScadaBench"
      VALUE "ProductName", "ScadaBench"
      VALUE "FileDescription", "{description}"
      VALUE "FileVersion", "{version}"
      VALUE "ProductVersion", "{version}"
      VALUE "OriginalFilename", "{filename}"
      VALUE "LegalCopyright", "ScadaBench"
    END
  END
  BLOCK "VarFileInfo"
  BEGIN
    VALUE "Translation", 0x409, 1200
  END
END
"""


def version_numbers(version):
    """'0.1.15' -> '0,1,15,0' (four 16-bit parts; anything after '+' or '-' is dropped)."""
    core = version.split("+")[0].split("-")[0]
    parts = [int(part) for part in core.split(".")][:4]
    if not parts or any(part < 0 or part > 65535 for part in parts):
        raise ValueError(f"version {version!r} cannot be a Windows file version")
    return ",".join(str(part) for part in parts + [0] * (4 - len(parts)))


def rc_text(icon, version, description, filename):
    return RC_TEMPLATE.format(icon=Path(icon).as_posix(), numbers=version_numbers(version), version=version,
                              description=description, filename=filename)


def _newest(pattern_roots, relative):
    found = []
    for root in pattern_roots:
        if root.is_dir():
            found += sorted(root.glob(relative))
    if not found:
        raise SystemExit(f"Cannot find {relative.rsplit('/', 1)[-1]}: install the Windows SDK and MSVC build tools.")
    return found[-1]


def tools():
    """(rc.exe, cvtres.exe) of the newest Windows SDK and MSVC toolset on this machine."""
    x86 = Path(os.environ.get("ProgramFiles(x86)", r"C:\Program Files (x86)"))
    programs = Path(os.environ.get("ProgramFiles", r"C:\Program Files"))
    rc = _newest([x86 / "Windows Kits/10/bin"], "10.*/x64/rc.exe")
    cvtres = _newest([x86 / "Microsoft Visual Studio", programs / "Microsoft Visual Studio"],
                     "*/*/VC/Tools/MSVC/*/bin/Hostx64/x64/cvtres.exe")
    return rc, cvtres


def patch_absolute_symbols(data):
    """Point cvtres' absolute marker symbols at its .debug$S section; return the new bytes."""
    data = bytearray(data)
    machine, sections, _, symbols_at, symbol_count, optional, _ = struct.unpack_from("<HHIIIHH", data, 0)
    if machine != 0x8664:
        raise ValueError("resource object is not x64")
    debug = None
    for index in range(sections):
        name = bytes(data[20 + optional + SECTION_HEADER_SIZE * index:][:8]).rstrip(b"\0")
        if name == b".debug$S":
            debug = index + 1  # section numbers are 1-based
    index = 0
    while index < symbol_count:
        offset = symbols_at + SYMBOL_SIZE * index
        section, = struct.unpack_from("<h", data, offset + 12)
        if section == IMAGE_SYM_ABSOLUTE:
            if debug is None:
                raise ValueError("resource object has absolute symbols but no .debug$S section")
            struct.pack_into("<h", data, offset + 12, debug)
        index += 1 + data[offset + 17]  # skip auxiliary records
    return bytes(data)


def make_syso(output, icon, version, description, filename):
    """Write output (a *_windows_amd64.syso) holding the icon and version information."""
    rc, cvtres = tools()
    with tempfile.TemporaryDirectory(prefix="scadabench-rc-") as work:
        source, compiled, coff = Path(work) / "app.rc", Path(work) / "app.res", Path(work) / "app.obj"
        source.write_text(rc_text(Path(icon).resolve(), version, description, filename), encoding="utf-8")
        subprocess.run([str(rc), "/nologo", "/fo", str(compiled), str(source)], check=True)
        subprocess.run([str(cvtres), "/nologo", "/machine:x64", f"/out:{coff}", str(compiled)], check=True)
        Path(output).write_bytes(patch_absolute_symbols(coff.read_bytes()))
    return Path(output)
