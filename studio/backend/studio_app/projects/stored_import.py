"""Rebuild stored IED models, parsing and validating each distinct SCL source only once.

Projects often hold several IEDs from one large SCD. Importing that document once with
all of its stored IED names selected yields the same per-IED models as importing it once
per IED, without repeating the XML parse and XSD validation for every IED.
"""
import copy
import re
from studio_app.contracts.errors import StudioError
from studio_app.scl.importer import import_scl


def import_stored(items):
    """Return {stored IED id: freshly built model}, keyed so callers keep stored order."""
    groups = {}
    for item in items:
        if not isinstance(item, dict):
            raise StudioError("Invalid stored IED")
        if not re.fullmatch(r"[0-9a-f]{32}", item.get("id", "")):
            raise StudioError("Invalid stored IED identifier")
        key = (item["source"], item.get("filename", "stored.scd"))
        groups.setdefault(key, []).append(item)
    if len({item["id"] for item in items}) != len(items):
        raise StudioError("Project contains too many or duplicate IEDs")
    models = {}
    for (source, filename), members in groups.items():
        names = [member["name"] for member in members]
        if len(set(names)) != len(names):
            raise StudioError("Project contains too many or duplicate IEDs")
        built = {model["name"]: model for model in import_scl(source, filename, names)}
        for member in members:
            model = built.get(member["name"])
            if model is None:
                raise StudioError("Stored IED is missing from its source XML")
            # The validation report is shared by every IED of one import; keep each IED's own.
            model["validation"] = copy.deepcopy(model["validation"])
            models[member["id"]] = model
    return models
