# One-folder developer preview; run from the workspace using build_portable.py.
from pathlib import Path
import os

studio = Path(SPECPATH).parent
workspace = studio.parent
native = Path(os.environ.get("STUDIO_BUILD_NATIVE", studio / "native/build/Release"))
frontend = Path(os.environ.get("STUDIO_BUILD_FRONTEND", studio / "frontend/dist"))
required = [native / "studio_ied.exe", native / "studio_live.exe", native / "studio_probe.exe",
            frontend / "index.html"]
for path in required:
    if not path.exists():
        raise RuntimeError(f"Build prerequisite missing: {path}")

analysis = Analysis(
    [str(studio / "packaging/entrypoint.py")],
    pathex=[str(studio / "backend")],
    binaries=[(str(native / "studio_ied.exe"), "native"),
              (str(native / "studio_live.exe"), "native"),
              (str(native / "studio_probe.exe"), "native")],
    datas=[(str(frontend), "frontend"),
           (str(studio / "schemas"), "schemas"),
           (str(studio / "packaging/README.md"), "."),
           (str(workspace / ".local/native/libiec61850/COPYING"), "licenses/libiec61850"),
           (str(workspace / ".local/native/json/LICENSE.MIT"), "licenses/nlohmann-json")],
    hiddenimports=[], hookspath=[], runtime_hooks=[], excludes=[], noarchive=False,
)
python_archive = PYZ(analysis.pure)
executable = EXE(python_archive, analysis.scripts, [], exclude_binaries=True,
                 name="SubstationStudio", debug=False, strip=False, upx=False, console=True)
collection = COLLECT(executable, analysis.binaries, analysis.datas, strip=False,
                     upx=False, name="SubstationStudio")
