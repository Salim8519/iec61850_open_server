"""Run fast maintained-file, architecture, and documentation checks."""
from concurrent.futures import ThreadPoolExecutor

from check_file_lengths import check as lengths
from check_architecture import check as architecture
from check_documentation import check as documentation
from history_log import check as history
from check_specs import check as specifications
from build_goose_engine import test as goose_engine
from engine_tests import test as engine

CHECKS = [("File lengths", lengths), ("Architecture", architecture),
          ("Documentation", documentation), ("Engineering history", history),
          ("Specifications", specifications), ("GOOSE engine (go test)", goose_engine),
          ("Go engine (go test, pyjson vs CPython)", engine)]


def main():
    failures = []
    # The checks are independent and the Go suites mostly wait on child processes, so all
    # start at once; results still print in this fixed order (change 0203).
    with ThreadPoolExecutor(max_workers=len(CHECKS)) as pool:
        running = [(name, pool.submit(check)) for name, check in CHECKS]
        for name, future in running:
            errors = future.result()
            print(f"{name}: {'FAIL' if errors else 'PASS'}", flush=True)
            failures.extend(errors)
    for error in failures:
        print(error)
    return bool(failures)


if __name__ == "__main__":
    raise SystemExit(main())
