"""Use the Python launcher's own lease and record code for the Go launcher tests.

Usage: python -I lease_probe.py <studio/backend> <data dir> try|hold|record <url> <pid>
try: take and release the studio.lock lease, print "ok" or the RuntimeError text.
hold: take the lease, print "locked", keep it until stdin closes.
record: write launcher.json with entrypoint.write_launcher_record.
"""
import pathlib
import sys

sys.path.insert(0, sys.argv[1])
from studio_app.launcher.directory_lease import DirectoryLease  # noqa: E402

folder, mode = pathlib.Path(sys.argv[2]), sys.argv[3]
if mode == "record":
    from studio_app.launcher.entrypoint import write_launcher_record
    write_launcher_record(folder, sys.argv[4], int(sys.argv[5]) or None)
    print("written", flush=True)
    raise SystemExit(0)
lease = DirectoryLease(folder)
try:
    lease.__enter__()
except RuntimeError as error:
    print(str(error), flush=True)
    raise SystemExit(0)
if mode == "hold":
    print("locked", flush=True)
    sys.stdin.read()
else:
    print("ok", flush=True)
lease.__exit__()
