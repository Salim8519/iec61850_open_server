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


def test_sv_is_subscribe_only():
    top = (NATIVE / "CMakeLists.txt").read_text(encoding="utf-8")
    support = (NATIVE / "cmake/sv_support.cmake").read_text(encoding="utf-8")
    code = "\n".join(line for line in support.splitlines() if not line.startswith("#"))
    assert "sv_subscriber.c" in code and "sv_publisher.c" not in code
    # The publisher source is compiled only into the test-only target, never into a product executable.
    assert top.count("sv_publisher.c") == 1
    test_target = top[top.index("add_executable(studio_sv_test_publisher"):]
    assert "sv_publisher.c" in test_target.split(")\n", 1)[0]
    for source in (NATIVE / "live").glob("*.[ch]pp"):
        assert "SVPublisher" not in source.read_text(encoding="utf-8"), source.name
    spec = (NATIVE.parent / "packaging/studio.spec").read_text(encoding="utf-8")
    assert "sv_test_publisher" not in spec
    options = (NATIVE / "cmake/goose_options.cmake").read_text(encoding="utf-8")
    assert "set(CONFIG_IEC61850_L2_SMV ON" in options and "set(CONFIG_IEC61850_R_SMV OFF" in options


def test_ethernet_hal_routes_udp_to_the_engine_and_numbers_to_npcap():
    """GOOSE-26: "udp:" ids reach the GOOSE engine; numeric ids keep the vendor Npcap HAL (SV)."""
    support = (NATIVE / "cmake/goose_support.cmake").read_text(encoding="utf-8")
    for name in ("ethernet_dispatch.c", "ethernet_pcap.c", "ethernet_udp.c"):
        assert f"hal/{name}" in support
    pcap = (NATIVE / "hal/ethernet_pcap.c").read_text(encoding="utf-8")
    assert '#include "ethernet_win32.c"' in pcap  # the vendor file compiles unmodified
    dispatch = (NATIVE / "hal/ethernet_dispatch.c").read_text(encoding="utf-8")
    assert "StudioUdp_isInterface(interfaceId)" in dispatch and "Pcap_Ethernet_createSocket" in dispatch
    worker = (NATIVE / "worker/goose/publishing.cpp").read_text(encoding="utf-8")
    assert "\"udp:\"" in worker and "npcap" not in worker.lower()
