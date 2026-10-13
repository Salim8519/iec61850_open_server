"""Check that every feature-specification rule is enforced by a test that exists."""
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SPECS = ROOT / "docs" / "specs"
RULE = re.compile(r"^\|\s*([A-Z]+-\d{2})\s*\|(.+)\|(.*)\|\s*$")
REFERENCE = re.compile(r"`([^`]+?)::([^`]+)`")


def _test_exists(path, name):
    target = ROOT / path
    if not target.is_file():
        return False
    text = target.read_text(encoding="utf-8")
    if target.suffix == ".py":
        return re.search(rf"^\s*def {re.escape(name)}\(", text, re.MULTILINE) is not None
    return any(f"test({quote}{name}{quote}" in text for quote in "\"'`")


def check():
    failures = []
    seen = {}
    for spec in sorted(SPECS.glob("*.md")):
        if spec.name == "README.md":
            continue
        rules = 0
        for number, line in enumerate(spec.read_text(encoding="utf-8").splitlines(), 1):
            match = RULE.match(line)
            if not match:
                continue
            rules += 1
            rule_id, enforced = match.group(1), match.group(3)
            where = f"{spec.relative_to(ROOT)}:{number}"
            if rule_id in seen:
                failures.append(f"{where}: rule {rule_id} duplicates {seen[rule_id]}")
            seen[rule_id] = where
            if "Retired:" in line:
                continue
            references = REFERENCE.findall(enforced)
            if not references:
                failures.append(f"{where}: rule {rule_id} names no enforcing test")
            for path, name in references:
                if not _test_exists(path, name.strip()):
                    failures.append(f"{where}: rule {rule_id} names missing test {path}::{name}")
        if not rules:
            failures.append(f"{spec.relative_to(ROOT)}: specification has no rules table")
    return failures


if __name__ == "__main__":
    errors = check()
    print("\n".join(errors) if errors else "Specifications: PASS")
    raise SystemExit(bool(errors))
