"""GOOSE is part of every native build and Npcap stays optional at run time."""
from pathlib import Path

NATIVE = Path(__file__).resolve().parents[2] / "native"


def test_goose_requires_pinned_sdk():
    options = (NATIVE / "cmake/goose_options.cmake").read_text(encoding="utf-8")
    assert "FATAL_ERROR" in options and "fetch_npcap_sdk.py" in options
    assert "set(CONFIG_INCLUDE_GOOSE_SUPPORT ON" in options and "set(CONFIG_IEC61850_L2_GOOSE ON" in options
    top = (NATIVE / "CMakeLists.txt").read_text(encoding="utf-8")
    assert top.index("goose_options.cmake") < top.index("add_subdirectory")
    assert "goose_support.cmake" in top and "OFF CACHE BOOL" not in top.split("goose_options")[1].split("\n")[0]


def test_npcap_is_delay_loaded():
    support = (NATIVE / "cmake/goose_support.cmake").read_text(encoding="utf-8")
    assert "/DELAYLOAD:wpcap.dll" in support and "/DELAYLOAD:Packet.dll" in support
    top = (NATIVE / "CMakeLists.txt").read_text(encoding="utf-8")
    loop = top[top.index("foreach(studio_target"):]
    assert "studio_ied studio_live studio_probe" in loop and "studio_delay_load_npcap" in loop
    loader = (NATIVE / "common/npcap.cpp").read_text(encoding="utf-8")
    assert "GetFileAttributesW" in loader and "SetDllDirectoryW" in loader
