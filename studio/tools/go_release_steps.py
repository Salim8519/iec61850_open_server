"""Steps of the Go release (ADR-0083): engine exe, installed folder layout, signatures, portable exe.

Driven by compile_go_release.py; inputs (frontend.pak, licensed workers) come from the same
steps as the Python release (compile_protected.py), so both releases ship identical workers.
"""
from pathlib import Path
import hashlib
import os
import secrets
import shutil
import subprocess

STUDIO = Path(__file__).resolve().parents[1]
ROOT = STUDIO.parent
ENGINE = STUDIO / "engine"
ICON = STUDIO / "branding/icon.ico"
WORKERS = ("studio_ied.exe", "studio_live.exe", "studio_goose.exe", "studio_probe.exe", "studio_gateway.exe",
           "studio_control.exe", "studio_master.exe", "studio_xsd.exe")
LICENSES = {
    ".local/native/libiec61850/COPYING": "licenses/libiec61850",
    ".local/native/json/LICENSE.MIT": "licenses/nlohmann-json",
    ".local/native/lib60870/COPYING": "licenses/lib60870",
    ".local/native/opendnp3/LICENSE": "licenses/opendnp3",
    ".local/native/opendnp3/NOTICE": "licenses/opendnp3",
    "studio/native/third_party/monocypher/LICENCE.md": "licenses/monocypher",
    ".local/native/libxml2-2.11.9/Copyright": "licenses/libxml2",
    ".local/native/zlib-1.3.1/LICENSE": "licenses/zlib",
    ".local/native/libiconv-1.17/COPYING.LIB": "licenses/libiconv",
    "studio/packaging/libiconv-NOTICE.txt": "licenses/libiconv",
}
PORTABLE_PAYLOAD = ENGINE / "cmd/scadabench_portable/payload"
# The workers' VC++ runtime (MSVCP140, VCRUNTIME140, VCRUNTIME140_1), shipped app-locally
# beside them so a clean PC needs no Visual C++ Redistributable install; the Universal CRT
# (api-ms-win-crt-*) is part of Windows 10 and later.
RUNTIME_DLLS = ("msvcp140.dll", "vcruntime140.dll", "vcruntime140_1.dll")
# GNU libiconv (LGPL-2.1) as the replaceable DLL studio_xsd.exe loads (build_xsd_helper.py).
LIBRARY_DLLS = ("libiconv-2.dll",)


def runtime_folder():
    """The newest x64 Microsoft.VC14x.CRT redistributable folder of the installed MSVC."""
    x86 = Path(os.environ.get("ProgramFiles(x86)", r"C:\Program Files (x86)"))
    programs = Path(os.environ.get("ProgramFiles", r"C:\Program Files"))
    found = []
    for root in (x86 / "Microsoft Visual Studio", programs / "Microsoft Visual Studio"):
        if root.is_dir():
            found += [path for path in root.glob("*/*/VC/Redist/MSVC/14.*/x64/Microsoft.VC14*.CRT")
                      if all((path / name).is_file() for name in RUNTIME_DLLS)]
    if not found:
        raise SystemExit("No MSVC x64 CRT redistributable found (Visual Studio C++ build tools).")
    return sorted(found)[-1]


def go_environment():
    return {**os.environ, "CGO_ENABLED": "0", "GOOS": "windows", "GOARCH": "amd64"}


def seal_text(native_bin, sealed):
    """The -X main.nativeSeal value: name=sha256 of every sealed worker (integrity.EXPECTED)."""
    rows = []
    for name in sealed:
        rows.append(f"{name}={hashlib.sha256((Path(native_bin) / name).read_bytes()).hexdigest()}")
    return ",".join(rows)


def go_build(mode, package, output, ldflags, tags=(), description="ScadaBench"):
    """Build one windowsgui exe with its icon/version resources; the .syso is always removed."""
    import go_release_resources
    from build_goose_engine import go_command
    package_dir = ENGINE / package
    syso = package_dir / "zz_release_windows_amd64.syso"
    print(f"  $ go build {package} -> {output}", flush=True)
    if mode.dry_run:
        return
    try:
        go_release_resources.make_syso(syso, ICON, mode.version, description, Path(output).name)
        command = [go_command(), "build", "-trimpath", *(["-tags", ",".join(tags)] if tags else []),
                   "-ldflags", "-H windowsgui -s -w " + " ".join(ldflags), "-o", str(output), "./" + package]
        subprocess.run(command, cwd=ENGINE, env=go_environment(), check=True)
    finally:
        syso.unlink(missing_ok=True)


def build_engine(mode, folder, sealed):
    """ScadaBench.exe = cmd/studio_server, packaged (license enforced), versioned and sealed."""
    mode.step("Go engine -> ScadaBench.exe")
    seal = "" if mode.dry_run else seal_text(mode.native_bin, sealed)
    flags = ["-X main.packaged=1", f"-X main.version={mode.version}", f"-X main.nativeSeal={seal}"]
    go_build(mode, "cmd/studio_server", folder / "ScadaBench.exe", flags)


def lay_out(mode, folder, frontend_pak):
    """The installed layout of the Python release: ScadaBench.exe + _internal resources."""
    mode.step("Folder layout")
    internal = folder / "_internal"
    print(f"  {folder} (ScadaBench.exe, _internal/native, frontend.pak, schemas, licenses, README.md)")
    if mode.dry_run:
        return
    (internal / "native").mkdir(parents=True)
    for name in WORKERS + LIBRARY_DLLS:
        source = mode.native_bin / name
        if not source.is_file():
            raise SystemExit(f"Build prerequisite missing: {source}")
        shutil.copy2(source, internal / "native" / name)
    runtime = runtime_folder()
    print(f"  VC++ runtime from {runtime}")
    for name in RUNTIME_DLLS:
        shutil.copy2(runtime / name, internal / "native" / name)
    shutil.copy2(frontend_pak, internal / "frontend.pak")
    shutil.copytree(STUDIO / "schemas", internal / "schemas")
    shutil.copy2(STUDIO / "packaging/README.md", internal / "README.md")
    for source, target in LICENSES.items():
        (internal / target).mkdir(parents=True, exist_ok=True)
        shutil.copy2(ROOT / source, internal / target / Path(source).name)


def code_sign(mode, targets):
    """Authenticode signing with STUDIO_SIGN_PFX (as compile_protected.sign); skipped without it."""
    certificate, signtool = os.environ.get("STUDIO_SIGN_PFX"), shutil.which("signtool")
    if not certificate or not signtool:
        print("  code signing skipped: set STUDIO_SIGN_PFX (and STUDIO_SIGN_PASSWORD) with signtool on PATH.")
        return
    mode.run([signtool, "sign", "/fd", "SHA256", "/tr", "http://timestamp.digicert.com", "/td", "SHA256",
              "/f", certificate, "/p", os.environ.get("STUDIO_SIGN_PASSWORD", ""), *targets])


def sign_module(mode, executable):
    """ScadaBench.exe.sig: Ed25519 over the exe's SHA-256 hex, verified by the licensed workers."""
    seed_file = STUDIO / "native/release/module_signing.seed"
    if not seed_file.exists():
        print("  module signature SKIPPED (no module_signing.seed; run studio/tools/module_signing_key.py)")
        return
    print(f"  signed module: {executable.name}.sig")
    if mode.dry_run:
        return
    from studio_app.licensing.ed25519_sign import sign
    seed = bytes.fromhex(seed_file.read_text(encoding="utf-8").strip())
    digest = hashlib.sha256(executable.read_bytes()).hexdigest().encode("utf-8")
    executable.with_name(executable.name + ".sig").write_bytes(sign(digest, seed))


def payload_key(folder):
    """16 hex digits over every file's relative path and SHA-256: the portable's app folder name."""
    digest = hashlib.sha256()
    for path in sorted(p for p in folder.rglob("*") if p.is_file()):
        digest.update(path.relative_to(folder).as_posix().encode("utf-8") + b"\0")
        digest.update(hashlib.sha256(path.read_bytes()).digest())
    return digest.hexdigest()[:16]


def build_portable(mode, folder, output):
    """ScadaBench-Portable-<v>.exe: the launcher embedding exactly the installed folder."""
    mode.step("Portable exe")
    if not mode.dry_run:
        shutil.rmtree(PORTABLE_PAYLOAD, ignore_errors=True)
        shutil.copytree(folder, PORTABLE_PAYLOAD)
        (PORTABLE_PAYLOAD / "payload.key").write_text(payload_key(folder) + "\n", encoding="ascii")
    try:
        go_build(mode, "cmd/scadabench_portable", output, [], tags=("release",), description="ScadaBench Portable")
    finally:
        if not mode.dry_run:
            shutil.rmtree(PORTABLE_PAYLOAD, ignore_errors=True)


def leak_check(mode, folder):
    """No loose UI or source files, no debug symbols, only the expected executables."""
    mode.step("Leak check")
    if mode.dry_run:
        return
    problems = []
    for path in folder.rglob("*"):
        relative = path.relative_to(folder).as_posix()
        if path.suffix in {".js", ".map", ".tsx", ".ts", ".go", ".py", ".pyc", ".pdb", ".syso"}:
            problems.append(relative)
        if path.suffix == ".html" and "licenses" not in relative:
            problems.append(relative)
        if path.suffix == ".exe" and relative not in {"ScadaBench.exe", *("_internal/native/" + n for n in WORKERS)}:
            problems.append(relative)
    if problems:
        raise SystemExit("Go release folder holds unexpected files:\n  " + "\n  ".join(sorted(problems)))
    print("  no source, symbols, loose UI files or unexpected programs")


def fresh_folder(mode):
    """A new empty dist-go/ScadaBench; the previous build is replaced, never merged (preflight
    has refused a folder holding user data)."""
    folder = mode.dist / "ScadaBench"
    if mode.dry_run:
        return folder
    if folder.exists():
        stale = mode.dist / f"ScadaBench.old-{secrets.token_hex(4)}"
        folder.rename(stale)
        shutil.rmtree(stale, ignore_errors=True)
    folder.mkdir(parents=True)
    return folder
