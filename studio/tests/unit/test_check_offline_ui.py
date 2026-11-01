import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "tools"))
import check_offline_ui  # noqa: E402


def _frontend(tmp_path, css="", tsx=""):
    (tmp_path / "src").mkdir()
    (tmp_path / "index.html").write_text("<!doctype html><title>ScadaBench</title>", encoding="utf-8")
    (tmp_path / "src" / "app.css").write_text(css, encoding="utf-8")
    (tmp_path / "src" / "App.tsx").write_text(tsx, encoding="utf-8")
    (tmp_path / "vite.config.ts").write_text(check_offline_ui.VITE.read_text(encoding="utf-8"), encoding="utf-8")
    return tmp_path


def test_the_real_interface_loads_nothing_online():
    assert check_offline_ui.check() == []


def test_a_web_font_is_refused(tmp_path, monkeypatch):
    folder = _frontend(tmp_path, css="@import url('https://fonts.googleapis.com/css2?family=Inter');")
    monkeypatch.setattr(check_offline_ui, "FRONTEND", folder)
    monkeypatch.setattr(check_offline_ui, "VITE", folder / "vite.config.ts")
    failures = check_offline_ui.check()
    assert any("loads something from the internet" in failure for failure in failures)


def test_an_unlisted_address_is_refused_but_a_listed_link_is_fine(tmp_path, monkeypatch):
    folder = _frontend(tmp_path, tsx="const a = 'https://scadabench.com/tutorials/';\n"
                                     "const b = 'https://example.com/logo.png';\n")
    monkeypatch.setattr(check_offline_ui, "FRONTEND", folder)
    monkeypatch.setattr(check_offline_ui, "VITE", folder / "vite.config.ts")
    failures = check_offline_ui.check()
    assert len(failures) == 1 and "https://example.com/logo.png" in failures[0]
