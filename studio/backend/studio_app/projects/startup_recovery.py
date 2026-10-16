"""Recover engineering state without starting IEDs or overwriting unreadable archives."""
import shutil
from uuid import uuid4
from .archive import MAX_UNPACKED


def recover_startup(projects):
    source = projects.directory / "autosave.subsim"
    if not source.exists():
        return None
    if source.is_symlink() or not source.is_file():
        raise RuntimeError("Autosave is not a regular local file; inspect it before starting ScadaBench")
    try:
        if source.stat().st_size > MAX_UNPACKED:
            raise ValueError("Saved archive exceeds the 100 MB project limit")
        projects.restore(source.read_bytes(), persist=False)
        return {"status": "restored", "message": "Recovered the last saved substation. All IEDs remain stopped."}
    except Exception as error:
        recovery = projects.directory / "recovery"
        if recovery.is_symlink():
            raise RuntimeError("Recovery folder is a link; cannot safely preserve unreadable autosave") from error
        recovery.mkdir(exist_ok=True)
        backup = recovery / ("unreadable-autosave-" + uuid4().hex + ".subsim")
        try:
            shutil.copyfile(source, backup)
        except OSError as backup_error:
            raise RuntimeError("Cannot preserve unreadable autosave; startup stopped to protect it") from backup_error
        return {"status": "unreadable", "message": "The last autosave could not be reopened. "
                "A separate recovery copy was preserved before allowing new work.",
                "details": str(error), "backup": str(backup)}
