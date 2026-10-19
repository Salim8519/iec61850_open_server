"""Run fast maintained-file, architecture, and documentation checks."""
from check_file_lengths import check as lengths
from check_architecture import check as architecture
from check_documentation import check as documentation
from history_log import check as history
from check_specs import check as specifications
from build_goose_engine import test as goose_engine


def main():
    failures = []
    for name, check in [("File lengths", lengths), ("Architecture", architecture),
                        ("Documentation", documentation), ("Engineering history", history),
                        ("Specifications", specifications), ("GOOSE engine (go test)", goose_engine)]:
        errors = check()
        print(f"{name}: {'FAIL' if errors else 'PASS'}")
        failures.extend(errors)
    for error in failures:
        print(error)
    return bool(failures)


if __name__ == "__main__":
    raise SystemExit(main())
