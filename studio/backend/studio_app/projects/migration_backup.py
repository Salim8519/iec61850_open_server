"""Preserve exact project bytes before a destructive format migration."""
import zipfile
from io import BytesIO
from uuid import uuid4
from studio_app.contracts.errors import StudioError
from .project_files import EXTENSION


def preserve(directory, payload):
    recovery = directory / "recovery"
    if recovery.is_symlink():
        raise StudioError("Recovery directory must not be a symbolic link")
    recovery.mkdir(exist_ok=True)
    destination = recovery / ("before-format-8-migration-" + uuid4().hex + EXTENSION)
    destination.write_bytes(payload)
    if destination.read_bytes() != payload:
        raise StudioError("Project migration recovery verification failed")
    try:
        with zipfile.ZipFile(BytesIO(payload)) as archive:
            if archive.testzip() is not None:
                raise StudioError("Project migration recovery archive is corrupt")
    except zipfile.BadZipFile as error:
        raise StudioError("Project migration recovery archive is invalid") from error
    return str(destination)
