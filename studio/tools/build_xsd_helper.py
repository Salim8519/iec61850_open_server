"""Build studio_xsd.exe (libxml2 2.11.9 + zlib 1.3.1) and libiconv-2.dll (GNU libiconv 1.17).

    python studio/tools/build_xsd_helper.py [--output PATH] [--build-id ID]
The steps of studio/native/xsd/README.md ("Build on Windows"). Nothing is downloaded: the
three unmodified source trees must already be in .local/native (see that README for the
archives and their SHA-256). The result is checked with the helper's own hello reply.
libiconv is LGPL-2.1 and ships as a replaceable DLL beside the helper (ADR-0083).
"""
from pathlib import Path
import argparse
import json
import shutil
import struct
import subprocess

STUDIO = Path(__file__).resolve().parents[1]
ROOT = STUDIO.parent
SOURCES = ROOT / ".local/native"
DEPS_BUILD = STUDIO / "native/build/xsd-deps"
HELPER_BUILD = STUDIO / "native/build/xsd"
OUTPUT = STUDIO / "native/build/Release/studio_xsd.exe"
GENERATOR = ["-G", "Visual Studio 16 2019", "-A", "x64", "-DCMAKE_SYSTEM_VERSION=10.0.19041.0"]
LIBXML2 = "2.11.9"
ICONV_DLL = "libiconv-2.dll"


def run(command):
    print("  $ " + " ".join(str(part) for part in command), flush=True)
    subprocess.run([str(part) for part in command], check=True)


def source(name):
    folder = SOURCES / name
    if not folder.is_dir():
        raise SystemExit(f"Missing {folder}. Unpack the pinned source archive there (studio/native/xsd/README.md).")
    return folder.as_posix()


def hello(executable):
    """Ask the helper for its versions (one framed request, then stop)."""
    frames = b""
    for request in ({"version": 1, "id": "1", "op": "hello"}, {"version": 1, "id": "2", "op": "stop"}):
        body = json.dumps(request).encode("utf-8")
        frames += struct.pack("<I", len(body)) + body
    output = subprocess.run([str(executable)], input=frames, capture_output=True, timeout=30,
                            creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0)).stdout
    size, = struct.unpack_from("<I", output, 0)
    return json.loads(output[4:4 + size])["result"]


def build(output=OUTPUT, build_id="dev"):
    if shutil.which("cmake") is None:
        raise SystemExit("cmake must be on PATH.")
    install = (DEPS_BUILD / "install").as_posix()
    run(["cmake", "-S", STUDIO / "native/xsd/cmake/deps", "-B", DEPS_BUILD, *GENERATOR,
         f"-DZLIB_SOURCE={source('zlib-1.3.1')}", f"-DLIBICONV_SOURCE={source('libiconv-1.17')}",
         f"-DCMAKE_INSTALL_PREFIX={install}"])
    run(["cmake", "--build", DEPS_BUILD, "--config", "Release", "--parallel", "4"])
    run(["cmake", "--install", DEPS_BUILD, "--config", "Release"])
    run(["cmake", "-S", STUDIO / "native/xsd", "-B", HELPER_BUILD, *GENERATOR,
         f"-DLIBXML2_SOURCE={source('libxml2-' + LIBXML2)}",
         f"-DIconv_INCLUDE_DIR={install}/include", f"-DIconv_LIBRARY={install}/lib/libiconv.lib",
         f"-DZLIB_INCLUDE_DIR={install}/include", f"-DZLIB_LIBRARY={install}/lib/zlib.lib",
         f"-DSTUDIO_XSD_BUILD={build_id}"])
    run(["cmake", "--build", HELPER_BUILD, "--config", "Release", "--parallel", "4", "--target", "studio_xsd"])
    built = HELPER_BUILD / "Release/studio_xsd.exe"
    shutil.copy2(DEPS_BUILD / "install/bin" / ICONV_DLL, built.with_name(ICONV_DLL))
    reply = hello(built)
    if reply.get("libxml2") != LIBXML2 or reply.get("libxml2_compiled") != LIBXML2 or reply.get("build") != build_id:
        raise SystemExit(f"studio_xsd.exe reports {reply}, expected libxml2 {LIBXML2} and build {build_id}.")
    Path(output).parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(built, output)
    shutil.copy2(built.with_name(ICONV_DLL), Path(output).with_name(ICONV_DLL))
    print(f"Built {output} (libxml2 {LIBXML2}, build {build_id})", flush=True)
    return Path(output)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--output", type=Path, default=OUTPUT)
    parser.add_argument("--build-id", default="dev")
    options = parser.parse_args(argv)
    build(options.output, options.build_id)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
