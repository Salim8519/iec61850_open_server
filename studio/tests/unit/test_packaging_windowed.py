"""The released EXE is a windowed desktop app: no console window for users."""
import re
from pathlib import Path

PACKAGING = Path(__file__).resolve().parents[2] / "packaging"


def test_release_specs_build_windowed_exe():
    for name in ("studio.spec", "studio_protected.spec"):
        text = (PACKAGING / name).read_text(encoding="utf-8")
        assert re.search(r"\bconsole=False\b", text), name
        assert "console=True" not in text, name
