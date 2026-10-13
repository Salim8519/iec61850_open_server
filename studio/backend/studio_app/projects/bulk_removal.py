"""Atomic multi-IED removal with a project-bound impact fingerprint and backup."""
import hashlib
import json
from uuid import uuid4
from studio_app.contracts.errors import StudioError
from .document import compact
from .removal import impacts, without_ieds


def preview(service, keys):
    if (not isinstance(keys, list) or not 1 <= len(keys) <= 20
            or any(not isinstance(key, str) for key in keys) or len(set(keys)) != len(keys)):
        raise StudioError("Select 1 to 20 unique IEDs")
    with service.lock:
        selected = [{"id": key, "name": service.find_ied(key)["name"]} for key in sorted(keys)]
        document = json.dumps({"project": compact(service.project), "ids": sorted(keys)}, sort_keys=True)
        return {"ieds": selected, **impacts(service.project, set(keys)),
                "fingerprint": hashlib.sha256(document.encode()).hexdigest()}


def remove(service, keys, fingerprint):
    with service.lock:
        current = preview(service, keys)
        if fingerprint != current["fingerprint"]:
            raise StudioError("Project changed. Review a fresh deletion preview")
        previous = service.project
        candidate = without_ieds(previous, set(keys))
        recovery = service.directory / "recovery"
        if recovery.is_symlink():
            raise StudioError("Recovery directory must not be a symbolic link")
        recovery.mkdir(exist_ok=True)
        backup = recovery / ("before-ied-delete-" + uuid4().hex + ".subsim")
        payload = service.export()
        with backup.open("xb") as stream:
            stream.write(payload)
        if hashlib.sha256(backup.read_bytes()).digest() != hashlib.sha256(payload).digest():
            raise StudioError("Recovery backup verification failed; nothing deleted")
        service.project = candidate
        try:
            service.save()
            service.changed()
        except Exception:
            service.project = previous
            raise
        return {**current, "recovery_file": str(backup)}
