"""Check forbidden domain dependencies using the Python syntax tree."""
import ast
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FORBIDDEN = {"flask", "subprocess", "socket", "pathlib", "os", "api", "runtime",
             "projects", "networking", "observability"}


def check():
    errors = []
    for path in (ROOT / "backend/studio_app/domain").rglob("*.py"):
        tree = ast.parse(path.read_text(encoding="utf-8"))
        for node in ast.walk(tree):
            modules = []
            if isinstance(node, ast.Import):
                modules = [item.name for item in node.names]
            elif isinstance(node, ast.ImportFrom):
                modules = [node.module or ""]
            for module in modules:
                if set(module.split(".")) & FORBIDDEN:
                    errors.append(f"{path.name}: forbidden domain dependency {module}")
    return errors
