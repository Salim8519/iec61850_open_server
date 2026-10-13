"""Source-checkout bootstrap for the elevated helper (frozen builds use --network-helper).

Windows starts elevated processes with a fresh environment, so this script adds
the backend folder itself before running the same helper code as the package.
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[3]))

from studio_app.networking.helper.runner import main  # noqa: E402

if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
