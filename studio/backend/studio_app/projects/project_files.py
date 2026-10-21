"""Names of ScadaBench project files and the refusal of the old .subsim extension (ADR-0096).

Project files are ZIP archives named *.scadabench. The app-owned copies (the autosave and the
recovery backups) use the same extension. The Go engine mirrors these names byte for byte
(engine/internal/store/projectfile).
"""
from studio_app.contracts.errors import StudioError

EXTENSION = ".scadabench"
OLD_EXTENSION = ".subsim"
AUTOSAVE_NAME = "autosave" + EXTENSION
DOWNLOAD_NAME = "substation" + EXTENSION
OLD_PROJECT_CODE = "old_project_file"
OLD_PROJECT_MESSAGE = ("This is an old project file (.subsim). "
                       "ScadaBench now uses .scadabench project files.")


def refuse_old_project(filename):
    """Raise the plain-English refusal when the user picked a file with the old extension."""
    if (filename or "").lower().endswith(OLD_EXTENSION):
        raise StudioError(OLD_PROJECT_MESSAGE, code=OLD_PROJECT_CODE)
