# Protected release: Nuitka-compiled studio_app (.pyd), packed UI, licensed native workers.
# Run only through studio/tools/compile_protected.py, which prepares every input below.
from pathlib import Path
import json
import os
from PyInstaller.utils.hooks import collect_submodules

studio = Path(SPECPATH).parent
workspace = studio.parent
protected = Path(os.environ["STUDIO_PROTECTED_DIR"])
native = Path(os.environ["STUDIO_PROTECTED_NATIVE"])
compiled = protected / "nuitka"
imports = json.loads((protected / "hidden_imports.json").read_text(encoding="utf-8"))
modules = sorted(compiled.glob("studio_app*.pyd"))
required = [native / "studio_ied.exe", native / "studio_live.exe", native / "studio_probe.exe",
            protected / "frontend.pak"]
if len(modules) != 1:
    raise RuntimeError(f"Expected one compiled studio_app .pyd in {compiled}, found {modules}")
for path in required:
    if not path.exists():
        raise RuntimeError(f"Build prerequisite missing: {path}")

hidden = list(imports["modules"])
for package in imports["packages"]:
    hidden += collect_submodules(package)

analysis = Analysis(
    [str(studio / "packaging/entrypoint.py")],
    pathex=[str(compiled)],
    binaries=[(str(modules[0]), "."),
              (str(native / "studio_ied.exe"), "native"),
              (str(native / "studio_live.exe"), "native"),
              (str(native / "studio_probe.exe"), "native")],
    datas=[(str(protected / "frontend.pak"), "."),
           (str(studio / "schemas"), "schemas"),
           (str(studio / "packaging/README.md"), "."),
           (str(workspace / ".local/native/libiec61850/COPYING"), "licenses/libiec61850"),
           (str(workspace / ".local/native/json/LICENSE.MIT"), "licenses/nlohmann-json"),
           (str(studio / "native/third_party/monocypher/LICENCE.md"), "licenses/monocypher")],
    hiddenimports=hidden, hookspath=[], runtime_hooks=[],
    excludes=["tkinter"], noarchive=False,
)
# The compiled .pyd is the only studio_app code shipped: drop any pure-Python copy.
analysis.pure = [entry for entry in analysis.pure if not entry[0].startswith("studio_app")]
python_archive = PYZ(analysis.pure)
executable = EXE(python_archive, analysis.scripts, [], exclude_binaries=True,
                 name="ScadaBench", debug=False, strip=False, upx=False, console=False,
                 icon=os.environ.get("STUDIO_ICON") or str(studio / "branding/icon.ico"))
collection = COLLECT(executable, analysis.binaries, analysis.datas, strip=False,
                     upx=False, name="ScadaBench")
