"""One-command protected Windows release (run only when a release is wanted).

    python studio/tools/compile_protected.py              real release -> studio/dist-protected
    python studio/tools/compile_protected.py --sandbox    test build for the local sandbox server -> studio/dist-sandbox
    python studio/tools/compile_protected.py --dry-run    show every step, change nothing
Steps: preflight -> frontend build + frontend.pak -> licensed native workers
(-DSTUDIO_ENFORCE_LICENSE=ON) -> sealed studio_app copy -> Nuitka .pyd (no docstrings) ->
PyInstaller one-folder app (fast start; no studio_app .py/.pyc, no loose UI files) ->
leak check -> optional code signing -> Inno Setup installer.
"""
import argparse
from dataclasses import dataclass
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import time

STUDIO = Path(__file__).resolve().parents[1]
ROOT = STUDIO.parent
NUITKA_VERSION = "4.3"
INSTALLER_TRIES = 8
sys.path.insert(0, str(STUDIO / "backend"))
sys.path.insert(0, str(STUDIO / "tools"))
import protected_source  # noqa: E402


@dataclass
class Mode:
    sandbox: bool
    dry_run: bool

    @property
    def name(self):
        return "protected-sandbox" if self.sandbox else "protected"

    @property
    def build(self):
        return STUDIO / "build" / self.name

    @property
    def native_build(self):
        return STUDIO / "native/build" / self.name

    @property
    def native_bin(self):
        return self.native_build / "Release"

    @property
    def dist(self):
        return STUDIO / ("dist-sandbox" if self.sandbox else "dist-protected")

    def run(self, command, **options):
        printable = " ".join(f'"{part}"' if " " in str(part) else str(part) for part in command)
        print(f"  $ {printable}", flush=True)
        if not self.dry_run:
            subprocess.run([str(part) for part in command], check=True, **options)

    @staticmethod
    def step(title):
        print(f"\n== {title}", flush=True)


def preflight(mode):
    mode.step("Preflight")
    if sys.platform != "win32":
        raise SystemExit("Build the protected release on Windows x64.")
    if shutil.which("cmake") is None or shutil.which("npm") is None:
        raise SystemExit("cmake and npm must be on PATH.")
    if (mode.dist / "ScadaBench/data").exists():
        raise SystemExit(f"{mode.dist} contains user data; move it away before building.")
    if mode.sandbox:
        release = protected_source.sandbox_release()
        print(f"  SANDBOX build: trusts only the local test server {release[0]}")
        return release
    from studio_app.licensing import release_key
    if not release_key.configured():
        raise SystemExit("Release key and server URL are not set. Run: python studio/tools/configure_release.py")
    print(f"  license server: {release_key.DEFAULT_SERVER_URL}, keys: {', '.join(release_key.public_keys())}")
    return None


def ensure_nuitka(mode):
    mode.step("Build tools")
    try:
        import nuitka  # noqa: F401
        print("  Nuitka present")
    except ImportError:
        mode.run([sys.executable, "-m", "pip", "install", f"nuitka=={NUITKA_VERSION}", "ordered-set", "zstandard"])


def build_frontend(mode):
    mode.step("Frontend (no source maps) -> frontend.pak")
    mode.run(["npm.cmd" if os.name == "nt" else "npm", "run", "build"], cwd=STUDIO / "frontend")
    mode.run([sys.executable, STUDIO / "tools/pack_frontend.py", "--out", mode.build / "frontend.pak"])


def build_native(mode, release):
    mode.step("Licensed native workers")
    keys = STUDIO / "native/release"
    if release is not None and not mode.dry_run:
        keys = protected_source.write_native_header(mode.build / "release-keys", release[1])
    mode.run(["cmake", "-S", STUDIO / "native", "-B", mode.native_build, "-G", "Visual Studio 16 2019", "-A", "x64",
              "-DSTUDIO_ENFORCE_LICENSE=ON", "-DSTUDIO_GATEWAY=ON",  # explicit: an older cache may hold OFF
              f"-DSTUDIO_RELEASE_KEY_DIR={keys.as_posix()}"])
    mode.run(["cmake", "--build", mode.native_build, "--config", "Release", "--parallel", "8", "--target",
              "studio_ied", "studio_live", "studio_probe", "studio_gateway", "studio_control", "studio_master",
              "studio_license_check"])
    if not mode.dry_run:
        flag = subprocess.run([str(mode.native_bin / "studio_license_check.exe"), "enforced"],
                              capture_output=True, text=True).stdout.strip()
        if flag != "1":
            raise SystemExit("Native workers were built without license enforcement; refusing to package.")
        print("  enforcement: ON")


def c_compiler_flag():
    """Nuitka needs MSVC 14.3+ (VS 2022 C++) for Python 3.12; otherwise use its own MinGW64."""
    vswhere = Path(os.environ.get("ProgramFiles(x86)", "")) / "Microsoft Visual Studio/Installer/vswhere.exe"
    if vswhere.is_file():
        found = subprocess.run([str(vswhere), "-version", "[17.0,)", "-products", "*", "-requires",
                                "Microsoft.VisualStudio.Component.VC.Tools.x86.x64", "-property", "installationPath"],
                               capture_output=True, text=True).stdout.strip()
        if found:
            return "--msvc=latest"
    return "--mingw64"


def build_backend(mode, release, version):
    mode.step("Sealed studio_app copy -> Nuitka native .pyd")
    source = mode.build / "src"
    if not mode.dry_run:
        protected_source.prepare(source, mode.native_bin, release, version)
    print(f"  sealed copy: {source / 'studio_app'} (native hashes{', sandbox key' if release else ''})")
    environment = {**os.environ, "NUITKA_CACHE_DIR": str(STUDIO / "build/nuitka-cache")}
    mode.run([sys.executable, "-m", "nuitka", "--mode=module", source / "studio_app", "--include-package=studio_app",
              f"--output-dir={mode.build / 'nuitka'}", c_compiler_flag(), "--lto=yes", "--no-pyi-file",
              "--python-flag=no_docstrings", "--remove-output", "--assume-yes-for-downloads"],
             env=environment, cwd=source)
    import collect_imports
    if not mode.dry_run:
        collect_imports.main(mode.build / "hidden_imports.json")
        sign_module(mode)


def sign_module(mode):
    """Sign the compiled module so the native workers can verify it at runtime."""
    seed_file = STUDIO / "native/release/module_signing.seed"
    modules = sorted((mode.build / "nuitka").glob("studio_app*.pyd"))
    if not modules:
        raise SystemExit("No compiled studio_app .pyd found to sign.")
    if not seed_file.exists():
        print("  module signature SKIPPED (no module_signing.seed; run studio/tools/module_signing_key.py)")
        return
    import hashlib
    from studio_app.licensing.ed25519_sign import sign
    seed = bytes.fromhex(seed_file.read_text(encoding="utf-8").strip())
    digest = hashlib.sha256(modules[0].read_bytes()).hexdigest().encode("utf-8")
    (modules[0].parent / (modules[0].name + ".sig")).write_bytes(sign(digest, seed))
    print(f"  signed module: {modules[0].name}.sig")


def build_bundle(mode):
    mode.step("PyInstaller one-folder app")
    environment = {**os.environ, "STUDIO_PROTECTED_DIR": str(mode.build), "STUDIO_PROTECTED_NATIVE": str(mode.native_bin)}
    mode.run([sys.executable, "-m", "PyInstaller", "--noconfirm", "--clean", "--distpath", mode.dist,
              "--workpath", mode.build / "pyinstaller", STUDIO / "packaging/studio_protected.spec"],
             cwd=ROOT, env=environment)


def find_leaks(bundle, workpath):
    problems = []
    for path in bundle.rglob("*"):
        relative = path.relative_to(bundle).as_posix()
        if "studio_app" in relative and path.suffix in {".py", ".pyc"}:
            problems.append(relative)
        if path.suffix in {".js", ".map", ".tsx", ".pdb"} or (path.suffix == ".html" and "licenses" not in relative):
            problems.append(relative)
    for toc in workpath.rglob("PYZ-*.toc"):
        if "'studio_app" in toc.read_text(encoding="utf-8", errors="replace"):
            problems.append(f"{toc.name} lists studio_app Python modules")
    return problems


def leak_check(mode):
    mode.step("Leak check")
    if mode.dry_run:
        print("  (skipped in dry run)")
        return
    problems = find_leaks(mode.dist / "ScadaBench", mode.build / "pyinstaller")
    if problems:
        raise SystemExit("Protected build leaks source:\n  " + "\n  ".join(problems))
    print("  no studio_app source, bytecode, symbols or loose UI files")


def sign(mode):
    mode.step("Code signing")
    certificate, signtool = os.environ.get("STUDIO_SIGN_PFX"), shutil.which("signtool")
    if not certificate or not signtool:
        print("  skipped: set STUDIO_SIGN_PFX (and STUDIO_SIGN_PASSWORD) with signtool on PATH to sign.")
        print("  Unsigned builds show a Windows SmartScreen warning on first start.")
        return
    targets = [mode.dist / "ScadaBench/ScadaBench.exe",
               *(mode.dist / "ScadaBench/_internal/native").glob("*.exe")]
    mode.run([signtool, "sign", "/fd", "SHA256", "/tr", "http://timestamp.digicert.com", "/td", "SHA256",
              "/f", certificate, "/p", os.environ.get("STUDIO_SIGN_PASSWORD", ""), *targets])


def installer(mode, version):
    mode.step("Installer")
    candidates = [Path(os.environ.get("ProgramFiles(x86)", "")) / "Inno Setup 6/ISCC.exe",
                  Path(os.environ.get("ProgramFiles", "")) / "Inno Setup 6/ISCC.exe"]
    iscc = next((path for path in candidates if path.is_file()), None) or shutil.which("ISCC")
    if not iscc:
        print("  skipped: install Inno Setup 6 (jrsoftware.org) to get ScadaBench-Setup.exe")
        return
    # Inno writes the setup outside the scanned release folder, then it is moved in: an antivirus
    # watching studio/dist-protected locked every icon update in the 0.1.8 build (change 0090).
    staging = Path(tempfile.mkdtemp(prefix="scadabench-setup-"))
    command = [iscc, f"/DSourceDir={mode.dist / 'ScadaBench'}", f"/DOutputDir={staging}",
               f"/DAppVersion={version}{'-sandbox' if mode.sandbox else ''}", STUDIO / "packaging/installer.iss"]
    # Antivirus scanners briefly lock freshly built files ("used by another process", seen in the
    # 0.1.7 build, change 0089); Inno Setup does not retry by itself. Same rule as make_portable.py.
    for attempt in range(1, INSTALLER_TRIES + 1):
        try:
            mode.run(command)
            for built in staging.glob("*.exe"):
                shutil.move(str(built), str(mode.dist / built.name))
            shutil.rmtree(staging, ignore_errors=True)
            return
        except subprocess.CalledProcessError:
            if attempt == INSTALLER_TRIES:
                raise
            print(f"  Inno Setup try {attempt} failed (often an antivirus scan); retrying in 10 s", flush=True)
            time.sleep(10)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--sandbox", action="store_true", help="test build that trusts the local sandbox server")
    parser.add_argument("--version", default="0.1.0")
    parser.add_argument("--skip-installer", action="store_true")
    args = parser.parse_args(argv)
    mode = Mode(sandbox=args.sandbox, dry_run=args.dry_run)
    release = preflight(mode)
    ensure_nuitka(mode)
    if not mode.dry_run:
        mode.build.mkdir(parents=True, exist_ok=True)
    build_frontend(mode)
    build_native(mode, release)
    build_backend(mode, release, args.version)
    build_bundle(mode)
    leak_check(mode)
    sign(mode)
    if not args.skip_installer:
        installer(mode, args.version)
    print(f"\nDone{' (dry run)' if mode.dry_run else ''}: {mode.dist}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
