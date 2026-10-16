"""Move the pre-ScadaBench user folder once, so licenses and projects survive the rename.

%LOCALAPPDATA%\\SubstationStudio becomes %LOCALAPPDATA%\\ScadaBench when only the old folder
exists. If both exist nothing is merged or deleted; the old folder is left for the user.
"""
import os
from pathlib import Path

OLD_NAME = "SubstationStudio"
NEW_NAME = "ScadaBench"


def migrate(local_app_data):
    """Return "moved", "kept-both" or "nothing" (packaged launches only)."""
    if not local_app_data:
        return "nothing"
    old, new = Path(local_app_data) / OLD_NAME, Path(local_app_data) / NEW_NAME
    if not old.is_dir():
        return "nothing"
    if new.exists():
        return "kept-both"
    try:
        os.replace(old, new)
    except OSError:
        return "nothing"
    return "moved"
