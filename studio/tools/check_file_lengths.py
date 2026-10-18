"""Enforce the 299-physical-line limit for maintained Studio files."""
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
EXCLUDED_DIRS = {
    "node_modules", "dist", "dist-protected", "dist-sandbox", "build", "__pycache__", ".venv", ".pytest_cache",
    "fixtures", "schemas", "ied-templates", "bay-templates", "examples",
    "generated", "data", "reference-screens", "third_party",
}
EXCLUDED_FILES = {"package-lock.json", "uv.lock"}
EXTENSIONS = {".py", ".ts", ".tsx", ".js", ".css", ".md", ".json", ".toml",
              ".yml", ".yaml", ".txt", ".cpp", ".h", ".hpp", ".ps1", ".spec",
              ".go", ".sh", ".html", ".iss", ".service", ".astro", ".mjs", ".cmd"}


def maintained_files():
    yield ROOT / "AGENTS.md"
    for folder in (ROOT / "studio", ROOT / "docs", ROOT / "license-server", ROOT / "website"):
        for path in folder.rglob("*"):
            if not path.is_file() or any(p in EXCLUDED_DIRS for p in path.parts):
                continue
            if path.name not in EXCLUDED_FILES and path.suffix in EXTENSIONS:
                yield path


def check():
    failures = []
    for path in maintained_files():
        count = len(path.read_text(encoding="utf-8-sig").splitlines())
        if count >= 300:
            failures.append(f"{path.relative_to(ROOT)}: {count} lines (maximum 299)")
    return failures


if __name__ == "__main__":
    errors = check()
    print("\n".join(errors) if errors else "File lengths: PASS")
    raise SystemExit(bool(errors))
