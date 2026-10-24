"""Build the app copy the tutorial videos record:  python marketing/videos/video_app.py

The videos must show the product as it ships, so this builds the Go engine (the released
engine, ADR-0083), the production frontend bundle and the native workers from the current
source into studio/build/video-app. It never touches the developer's running app
(studio/build/engine, studio/native/build): natives are compiled in their own CMake folder
(studio/build/video-native, or STUDIO_VIDEO_NATIVE_BUILD). Layout of the result:
studio_server.exe, frontend/ and native/, as in the packaged app.
"""
from __future__ import annotations

import os
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
STUDIO = ROOT / "studio"
APP = STUDIO / "build" / "video-app"
NATIVE_BUILD = Path(os.environ.get("STUDIO_VIDEO_NATIVE_BUILD", STUDIO / "build" / "video-native"))
OWNER_NATIVE = STUDIO / "native" / "build" / "Release"
# Workers built from source here; the other native programs are copied from the developer build.
BUILT = ("studio_ied", "studio_control", "studio_live", "studio_probe", "studio_gateway", "studio_master")
LOCAL = ROOT / ".local" / "native"


def go_server() -> None:
    sys.path.insert(0, str(STUDIO / "tools"))
    from build_goose_engine import go_command

    environment = {**os.environ, "CGO_ENABLED": "0", "GOOS": "windows", "GOARCH": "amd64"}
    subprocess.run([go_command(), "build", "-o", str(APP / "studio_server.exe"), "./cmd/studio_server"],
                   cwd=STUDIO / "engine", env=environment, check=True)


def frontend() -> None:
    vite = STUDIO / "frontend" / "node_modules" / "vite" / "bin" / "vite.js"
    subprocess.run(["node", str(vite), "build", "--outDir", str(APP / "frontend"), "--emptyOutDir",
                    "--logLevel", "warn"], cwd=STUDIO / "frontend", check=True)


def natives() -> None:
    if not (NATIVE_BUILD / "CMakeCache.txt").exists():
        deps = LOCAL / "opendnp3-deps"
        subprocess.run(["cmake", "-S", str(STUDIO / "native"), "-B", str(NATIVE_BUILD), "-G", "Visual Studio 16 2019",
                        "-A", "x64", f"-DNPCAP_SDK={LOCAL / 'npcap-sdk'}",
                        f"-DFETCHCONTENT_SOURCE_DIR_ASIO={deps / 'asio'}",
                        f"-DFETCHCONTENT_SOURCE_DIR_EXE4CPP={deps / 'exe4cpp'}",
                        f"-DFETCHCONTENT_SOURCE_DIR_SER4CPP={deps / 'ser4cpp'}"], check=True)
    subprocess.run(["cmake", "--build", str(NATIVE_BUILD), "--config", "Release", "--parallel", "6",
                    "--target", *BUILT], check=True)
    target = APP / "native"
    shutil.rmtree(target, ignore_errors=True)
    shutil.copytree(OWNER_NATIVE, target, ignore=shutil.ignore_patterns("*.prev", "*.lib"))
    for name in BUILT:
        shutil.copy2(NATIVE_BUILD / "Release" / f"{name}.exe", target / f"{name}.exe")


if __name__ == "__main__":
    APP.mkdir(parents=True, exist_ok=True)
    natives()
    go_server()
    frontend()
    print(APP)
