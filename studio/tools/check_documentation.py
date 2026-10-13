"""Check local Markdown links in the maintained documentation."""
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


def check():
    errors = []
    for path in (ROOT / "docs").rglob("*.md"):
        for target in re.findall(r"\]\(([^)]+)\)", path.read_text(encoding="utf-8")):
            if "://" in target or target.startswith("#"):
                continue
            if not (path.parent / target.split("#")[0]).exists():
                errors.append(f"{path.relative_to(ROOT)}: missing link {target}")
    return errors
