"""One-command Go release (ADR-0083; run only when a release is wanted).

    python studio/tools/compile_go_release.py --version 0.1.15            -> studio/dist-go
    python studio/tools/compile_go_release.py --version 0.1.15 --dry-run  show every step
Steps: preflight -> frontend build + frontend.pak -> licensed native workers + studio_goose
(shared with compile_protected.py) -> studio_xsd -> studio_snmp -> scadabench_hostkey -> Go engine
ScadaBench.exe (packaged, sealed)
-> installed folder layout -> leak check -> code signing -> module signature -> portable exe
embedding that exact folder -> Inno Setup installer (same installer.iss and AppId as Python).
The Python release (compile_protected.py) is unchanged and stays the fallback.
"""
import argparse
from pathlib import Path
import shutil
import sys

STUDIO = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(STUDIO / "backend"))
sys.path.insert(0, str(STUDIO / "tools"))
import compile_protected  # noqa: E402
import go_release_steps as steps  # noqa: E402
from protected_source import SEALED  # noqa: E402


class GoMode(compile_protected.Mode):
    """compile_protected's Mode with the Go release's own build and output folders; the
    licensed native build folder (native/build/protected) is shared with the Python release."""

    def __init__(self, dry_run, version):
        super().__init__(sandbox=False, dry_run=dry_run)
        self.version = version

    @property
    def build(self):
        return STUDIO / "build/go-release"

    @property
    def dist(self):
        return STUDIO / "dist-go"

    @property
    def native_build(self):
        return STUDIO / "native/build/protected"


def build_xsd(mode):
    """studio_xsd.exe (pinned libxml2) beside the workers; the Go engine's schema validator."""
    mode.step("Schema helper -> studio_xsd.exe")
    import build_xsd_helper
    target = mode.native_bin / "studio_xsd.exe"
    print(f"  $ build_xsd_helper -> {target}", flush=True)
    if not mode.dry_run:
        build_xsd_helper.build(output=target, build_id=f"scadabench-{mode.version}")


def build_snmp(mode):
    """studio_snmp.exe (ADR-0092) beside the workers, built from the engine module."""
    mode.step("SNMP engine -> studio_snmp.exe")
    import build_snmp_engine
    target = mode.native_bin / "studio_snmp.exe"
    print(f"  $ go build ./cmd/studio_snmp -> {target}", flush=True)
    if not mode.dry_run:
        build_snmp_engine.build(output=target)


def build_hostkey(mode):
    """scadabench_hostkey.exe (ADR-0099) beside the workers: the helper that lets the PC's virtual machines
    use its lifetime license. Built without build tags, so the test seam is never in a release."""
    mode.step("Virtual machine helper -> scadabench_hostkey.exe")
    import build_hostkey_helper
    target = mode.native_bin / "scadabench_hostkey.exe"
    print(f"  $ go build ./cmd/scadabench_hostkey -> {target}", flush=True)
    if not mode.dry_run:
        build_hostkey_helper.build(output=target)


def preflight(mode):
    release = compile_protected.preflight(mode)
    if not mode.version or not all(part.isdigit() for part in mode.version.split(".")):
        raise SystemExit("Pass the release version, for example --version 0.1.15")
    if (STUDIO / "engine/cmd/scadabench_portable/payload").exists() and not mode.dry_run:
        shutil.rmtree(STUDIO / "engine/cmd/scadabench_portable/payload")
    return release


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--version", required=True)
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--skip-installer", action="store_true")
    args = parser.parse_args(argv)
    mode = GoMode(dry_run=args.dry_run, version=args.version)
    release = preflight(mode)
    if not mode.dry_run:
        mode.build.mkdir(parents=True, exist_ok=True)
        mode.dist.mkdir(parents=True, exist_ok=True)
    compile_protected.build_frontend(mode)
    compile_protected.build_native(mode, release)
    build_xsd(mode)
    build_snmp(mode)
    build_hostkey(mode)
    folder = steps.fresh_folder(mode)
    steps.build_engine(mode, folder, SEALED)
    steps.lay_out(mode, folder, mode.build / "frontend.pak")
    steps.leak_check(mode, folder)
    mode.step("Code signing")
    executables = [folder / "ScadaBench.exe", *((folder / "_internal/native").glob("*.exe"))]
    steps.code_sign(mode, executables)
    steps.sign_module(mode, folder / "ScadaBench.exe")
    portable = mode.dist / f"ScadaBench-Portable-{mode.version}.exe"
    steps.build_portable(mode, folder, portable)
    steps.code_sign(mode, [portable])
    if not args.skip_installer:
        compile_protected.installer(mode, mode.version)
    print(f"\nDone{' (dry run)' if mode.dry_run else ''}: {mode.dist}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
