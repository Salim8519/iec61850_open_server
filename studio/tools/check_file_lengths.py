"""Enforce the 299-physical-line limit for maintained Studio files."""
import os
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
EXCLUDED_DIRS = {
    "node_modules", "dist", "dist-protected", "dist-sandbox", "build", "__pycache__", ".venv", ".pytest_cache",
    "fixtures", "schemas", "ied-templates", "bay-templates", "examples",
    "generated", "data", "reference-screens", "third_party",
}
EXCLUDED_FILES = {"package-lock.json", "uv.lock"}
# Verbatim IETF/IANA MIB modules (imported engineering data, ADR-0092).
EXCLUDED_PATHS = ("studio/engine/internal/snmp/mib/builtin/",)
EXTENSIONS = {".py", ".ts", ".tsx", ".js", ".css", ".md", ".json", ".toml",
              ".yml", ".yaml", ".txt", ".cpp", ".h", ".hpp", ".ps1", ".spec",
              ".go", ".sh", ".html", ".iss", ".service", ".astro", ".mjs", ".cmd"}


def maintained_files():
    yield ROOT / "AGENTS.md"
    for folder in (ROOT / "studio", ROOT / "docs", ROOT / "license-server", ROOT / "website"):
        # Prune excluded folders before descending: walking node_modules, build and .venv
        # only to discard them took minutes (change 0203).
        for directory, folders, names in os.walk(folder):
            folders[:] = [name for name in folders if name not in EXCLUDED_DIRS]
            for name in names:
                path = Path(directory, name)
                if path.relative_to(ROOT).as_posix().startswith(EXCLUDED_PATHS):
                    continue
                if name not in EXCLUDED_FILES and path.suffix in EXTENSIONS:
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
