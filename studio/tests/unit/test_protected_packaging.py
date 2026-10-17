"""Protected release pipeline: packed UI, data folder, app window, build steps, leaks."""
import importlib.util
from pathlib import Path
from flask import Flask
from studio_app.launcher import app_window
from studio_app.launcher.frontend_assets import FrontendAssets, pack, unpack
from studio_app.settings import Settings

STUDIO = Path(__file__).resolve().parents[2]


def load_tool(name):
    spec = importlib.util.spec_from_file_location(f"studio_tool_{name}", STUDIO / f"tools/{name}.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def test_packed_frontend_serves_identical_bytes(tmp_path):
    dist = tmp_path / "dist"
    (dist / "assets").mkdir(parents=True)
    files = {"index.html": b"<html>studio</html>", "assets/app.js": b"console.log(1)" * 500,
             "assets/app.css": b"body{}"}
    for name, content in files.items():
        (dist / name).write_bytes(content)
    blob = pack(files)
    assert unpack(blob) == files and b"console.log" not in blob
    (tmp_path / "frontend.pak").write_bytes(blob)
    app = Flask(__name__)
    with app.test_request_context():
        packed, folder = FrontendAssets(tmp_path / "frontend.pak"), FrontendAssets(dist)
        assert packed.available() and folder.available()
        for served in (packed.asset("app.js"), folder.asset("app.js")):
            served.direct_passthrough = False
            assert served.get_data() == files["assets/app.js"] and "javascript" in served.mimetype
        assert packed.index().get_data() == files["index.html"]
        assert packed.asset("missing.js").status_code == 404


def test_installed_copy_keeps_data_per_user_but_portable_copy_stays_beside_exe(tmp_path):
    installed = Settings.packaged_data_directory(tmp_path / "Program Files/Studio", str(tmp_path / "Local"))
    assert installed == tmp_path / "Local/ScadaBench/data"
    portable = tmp_path / "portable"
    (portable / "data").mkdir(parents=True)
    assert Settings.packaged_data_directory(portable, str(tmp_path / "Local")) == portable / "data"


def test_old_user_folder_moves_to_scadabench_once_and_never_merges(tmp_path):
    from studio_app.launcher.legacy_folder import migrate
    (tmp_path / "SubstationStudio/data").mkdir(parents=True)
    (tmp_path / "SubstationStudio/license.cert").write_text("x")
    assert migrate(str(tmp_path)) == "moved"
    assert (tmp_path / "ScadaBench/license.cert").exists() and not (tmp_path / "SubstationStudio").exists()
    (tmp_path / "SubstationStudio").mkdir()
    assert migrate(str(tmp_path)) == "kept-both" and (tmp_path / "SubstationStudio").exists()
    assert migrate(None) == "nothing"


def test_app_window_is_chromeless_and_isolated():
    command = app_window.app_window_command(Path("C:/Edge/msedge.exe"), "http://127.0.0.1:1/#session=x", Path("P"))
    assert "--app=http://127.0.0.1:1/#session=x" in command and "--user-data-dir=P" in command
    assert app_window.find_browser({}) is None


def test_protected_build_refuses_without_release_key_and_dry_run_lists_steps(monkeypatch, capsys):
    build = load_tool("compile_protected")
    from studio_app.licensing import release_key
    monkeypatch.setattr(release_key, "configured", lambda: False)
    try:
        build.main(["--dry-run"])
    except SystemExit as error:
        assert "configure_release.py" in str(error)
    monkeypatch.setattr(release_key, "configured", lambda: True)
    assert build.main(["--dry-run"]) == 0
    output = capsys.readouterr().out
    for marker in ("-DSTUDIO_ENFORCE_LICENSE=ON", "-DSTUDIO_GATEWAY=ON", "studio_control", "nuitka", "--mode=module", "pack_frontend.py",
                   "studio_protected.spec", "Leak check"):
        assert marker in output, marker


def test_leak_check_finds_source_bytecode_and_loose_ui(tmp_path):
    build = load_tool("compile_protected")
    bundle = tmp_path / "ScadaBench"
    (bundle / "_internal/studio_app").mkdir(parents=True)
    (bundle / "_internal/studio_app/service.pyc").write_bytes(b"x")
    (bundle / "_internal/index.js").write_text("x")
    (bundle / "_internal/licenses").mkdir()
    (bundle / "_internal/licenses/notice.html").write_text("ok")
    problems = build.find_leaks(bundle, tmp_path / "work")
    assert any("service.pyc" in item for item in problems) and any("index.js" in item for item in problems)
    assert not any("notice.html" in item for item in problems)


def test_release_configuration_round_trips(tmp_path, monkeypatch):
    tool = load_tool("configure_release")
    key = "2152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12"
    generated = tmp_path / "release_key.py"
    generated.write_text(tool.python_source("https://license.example.com", [key]), encoding="utf-8")
    assert tool.current_values(generated) == ("https://license.example.com", [key])
    assert key not in generated.read_text() and "kKeyCount = 1" in tool.native_source([key])
    namespace = {}
    exec(generated.read_text(), namespace)
    assert list(namespace["public_keys"]().values()) == [bytes.fromhex(key)] and namespace["configured"]()


def test_protected_copy_is_sealed_and_repository_source_untouched(tmp_path):
    tool = load_tool("protected_source")
    native = tmp_path / "native"
    native.mkdir()
    for name in tool.SEALED:
        (native / name).write_bytes(name.encode())
    key = "2152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12"
    before = (STUDIO / "backend/studio_app/licensing/integrity.py").read_bytes()
    copy = tool.prepare(tmp_path / "src", native, ("http://127.0.0.1:8090", [key]))
    sealed = {}
    exec((copy / "licensing/integrity.py").read_text(), sealed)
    assert set(sealed["EXPECTED"]) == set(tool.SEALED) and sealed["problem"](native) is None
    (native / "studio_ied.exe").write_bytes(b"patched")
    assert sealed["problem"](native) == sealed["TAMPERED"]
    (native / "studio_ied.exe").write_bytes(b"studio_ied.exe")
    (native / "studio_gateway.exe").write_bytes(b"swapped gateway")
    assert sealed["problem"](native) == sealed["TAMPERED"]
    assert 'DEFAULT_SERVER_URL = "http://127.0.0.1:8090"' in (copy / "licensing/release_key.py").read_text()
    assert not list(copy.rglob("__pycache__"))
    assert (STUDIO / "backend/studio_app/licensing/integrity.py").read_bytes() == before


def test_hidden_imports_cover_only_declared_packages():
    result = load_tool("collect_imports").main()
    assert {"flask", "waitress", "pydantic", "lxml"} <= set(result["packages"])
    assert "winreg" in result["modules"] and not any(name.startswith("studio_app") for name in result["modules"])


def test_native_enforcement_is_off_unless_release_build_asks():
    cmake = (STUDIO / "native/CMakeLists.txt").read_text(encoding="utf-8")
    assert 'option(STUDIO_ENFORCE_LICENSE "Release builds only' in cmake and 'license" OFF)' in cmake
    assert "PRIVATE iec61850 studio_license" in cmake
    for spec in ("studio.spec", "studio_protected.spec"):
        assert "license_check" not in (STUDIO / "packaging" / spec).read_text(encoding="utf-8")
    guard = (STUDIO / "native/common/license_guard.cpp").read_text(encoding="utf-8")
    assert "getenv(\"STUDIO_ENFORCE" not in guard and "native/build" not in guard
