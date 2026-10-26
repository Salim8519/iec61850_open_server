"""Go release pipeline (ADR-0083): resources, seal, layout, portable key, installer upgrade rule."""
import hashlib
import importlib.util
from pathlib import Path
import struct
import sys
import pytest

STUDIO = Path(__file__).resolve().parents[2]


def load_tool(name):
    sys.path.insert(0, str(STUDIO / "tools"))
    spec = importlib.util.spec_from_file_location(f"studio_tool_{name}", STUDIO / f"tools/{name}.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def coff(symbols, sections=(b".debug$S", b".rsrc$01")):
    """A minimal x64 COFF object: section headers and a symbol table, no data."""
    header_size = 20 + 40 * len(sections)
    data = bytearray(struct.pack("<HHIIIHH", 0x8664, len(sections), 0, header_size, sum(1 + aux for _, _, aux in symbols), 0, 0))
    for name in sections:
        data += name.ljust(8, b"\0") + bytes(32)
    for name, section, aux in symbols:
        data += name.ljust(8, b"\0") + struct.pack("<IhHBB", 0, section, 0, 3, aux)
        data += bytes(18 * aux)
    return bytes(data)


def symbol_sections(data):
    _, _, _, at, count, _, _ = struct.unpack_from("<HHIIIHH", data, 0)
    found, index = [], 0
    while index < count:
        found.append(struct.unpack_from("<h", data, at + 18 * index + 12)[0])
        index += 1 + data[at + 18 * index + 17]
    return found


def test_version_numbers_are_windows_file_versions():
    resources = load_tool("go_release_resources")
    assert resources.version_numbers("0.1.15") == "0,1,15,0"
    assert resources.version_numbers("0.1.14+dev") == "0,1,14,0"
    with pytest.raises(ValueError):
        resources.version_numbers("0.70000.1")


def test_absolute_markers_move_to_debug_section_and_nothing_else_changes():
    resources = load_tool("go_release_resources")
    original = coff([(b"@comp.id", -1, 0), (b"@feat.00", -1, 0), (b".rsrc$01", 2, 1), (b"$R000000", 2, 0)])
    patched = resources.patch_absolute_symbols(original)
    assert symbol_sections(patched) == [1, 1, 2, 2]
    assert len(patched) == len(original)
    assert sum(a != b for a, b in zip(original, patched)) == 4  # two 16-bit section numbers
    with pytest.raises(ValueError):
        resources.patch_absolute_symbols(coff([(b"@feat.00", -1, 0)], sections=(b".rsrc$01",)))


def test_seal_names_every_sealed_worker_with_its_digest(tmp_path):
    steps = load_tool("go_release_steps")
    (tmp_path / "a.exe").write_bytes(b"one")
    (tmp_path / "b.exe").write_bytes(b"two")
    text = steps.seal_text(tmp_path, ("a.exe", "b.exe"))
    assert text == f"a.exe={hashlib.sha256(b'one').hexdigest()},b.exe={hashlib.sha256(b'two').hexdigest()}"


def test_sealed_set_matches_the_python_release():
    steps = load_tool("go_release_steps")
    from protected_source import SEALED
    assert set(SEALED) <= set(steps.WORKERS)


def test_portable_key_follows_content_and_paths(tmp_path):
    steps = load_tool("go_release_steps")
    (tmp_path / "_internal/native").mkdir(parents=True)
    (tmp_path / "ScadaBench.exe").write_bytes(b"engine")
    (tmp_path / "_internal/native/studio_ied.exe").write_bytes(b"ied")
    first = steps.payload_key(tmp_path)
    assert len(first) == 16 and int(first, 16) >= 0
    assert steps.payload_key(tmp_path) == first
    (tmp_path / "_internal/native/studio_ied.exe").write_bytes(b"ied2")
    assert steps.payload_key(tmp_path) != first


def test_go_release_ships_the_python_release_programs_plus_the_go_only_helpers():
    # studio_xsd.exe (ADR-0083), studio_snmp.exe (ADR-0092), scadabench_hostkey.exe (ADR-0099) and
    # scadabench_labnet.exe (ADR-0120) are Go-only.
    steps = load_tool("go_release_steps")
    spec = (STUDIO / "packaging/studio_protected.spec").read_text(encoding="utf-8")
    python_programs = {name for name in steps.WORKERS if f'native / "{name}"' in spec}
    assert python_programs == set(steps.WORKERS) - {"studio_xsd.exe", "studio_snmp.exe", "scadabench_hostkey.exe",
                                                 "scadabench_labnet.exe"}


def test_installer_removes_old_program_files_before_copying():
    text = (STUDIO / "packaging/installer.iss").read_text(encoding="utf-8")
    install_delete = text.split("[InstallDelete]", 1)[1].split("[", 1)[0]
    assert 'Name: "{app}\\_internal"' in install_delete
    assert text.index("[InstallDelete]") < text.index("[Files]")


def test_libiconv_ships_as_replaceable_dll_with_its_source_named():
    steps = load_tool("go_release_steps")
    assert steps.LIBRARY_DLLS == ("libiconv-2.dll",)
    assert steps.LICENSES["studio/packaging/libiconv-NOTICE.txt"] == "licenses/libiconv"
    notice = (STUDIO / "packaging/libiconv-NOTICE.txt").read_text(encoding="utf-8")
    deploy = (STUDIO.parent / "website/deploy.cmd").read_text(encoding="utf-8")
    assert "https://download.scadabench.com/source/libiconv-1.17.tar.gz" in notice
    assert "scadabench-downloads/source/libiconv-1.17.tar.gz" in deploy
    readme = (STUDIO / "native/xsd/README.md").read_text(encoding="utf-8")
    digest = "8f74213b56238c85a50a5329f77e06198771e70dd9a739779f4c02f65d971313"
    assert digest in notice and digest in readme
