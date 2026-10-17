"""List the modules studio_app imports, for PyInstaller in protected builds.

The Nuitka-compiled studio_app .pyd hides its imports from PyInstaller's analysis,
so the protected spec declares them as hidden imports. Third-party packages are
expanded to all submodules; standard-library modules are listed by name.
"""
import ast
import json
from pathlib import Path
import sys

STUDIO = Path(__file__).resolve().parents[1]
PACKAGE = STUDIO / "backend/studio_app"
THIRD_PARTY = {"flask", "werkzeug", "waitress", "pydantic", "pydantic_core", "lxml", "defusedxml",
               "jinja2", "itsdangerous", "click", "blinker", "markupsafe", "annotated_types",
               "typing_extensions", "truststore"}


def imported_modules(package=PACKAGE):
    found = set()
    for path in package.rglob("*.py"):
        tree = ast.parse(path.read_text(encoding="utf-8"))
        for node in ast.walk(tree):
            if isinstance(node, ast.Import):
                found.update(alias.name for alias in node.names)
            elif isinstance(node, ast.ImportFrom) and node.level == 0 and node.module:
                found.add(node.module)
    return sorted(name for name in found if not name.startswith("studio_app"))


def main(out=None):
    modules = imported_modules()
    roots = sorted({name.split(".")[0] for name in modules})
    unknown = [root for root in roots if root not in THIRD_PARTY and root not in sys.stdlib_module_names]
    if unknown:
        raise SystemExit(f"Unlisted third-party import(s) {unknown}: add them to THIRD_PARTY in collect_imports.py")
    result = {"modules": modules, "packages": sorted(set(roots) & THIRD_PARTY | {"werkzeug", "jinja2"})}
    text = json.dumps(result, indent=2)
    if out:
        Path(out).parent.mkdir(parents=True, exist_ok=True)
        Path(out).write_text(text, encoding="utf-8")
    return result


if __name__ == "__main__":
    print(json.dumps(main(sys.argv[1] if len(sys.argv) > 1 else None), indent=2))
