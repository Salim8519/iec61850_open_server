"""Remember which licenses the server cancelled, so a renamed certificate can never come back.

When the license server's signed verdict is "revoked" (or "moved"), the license ID goes
into three places beside the trial clock (clock_store.py): the dot-file `.revoked_ids`,
the string `RevokedLicenses` under HKCU\\Software\\ScadaBench and the `revoked_ids` field
of license_state.json. Each copy is the IDs sorted and joined by ",". Evaluation refuses
any certificate whose ID is in ANY copy, and restores the copies that were deleted from
the ones that remain. Nothing here ever removes an ID: a cancelled license stays
cancelled. Native twin: license_clock.cpp (read_revoked_ids).
"""
from .clock_store import ClockValue


class RevokedIds:
    def __init__(self, home, registry=None):
        self.value = ClockValue("revoked_ids", home, registry)

    @staticmethod
    def parse(text):
        return {part.strip() for part in str(text).split(",") if part.strip()}

    def all(self):
        """The union of every copy that can be read."""
        found = set()
        for copy in self.value.read_all():
            found |= self.parse(copy)
        return found

    def contains(self, license_id):
        return isinstance(license_id, str) and license_id.strip() in self.all()

    def add(self, license_id):
        """Remember a cancelled license everywhere (an ID that is not text is ignored)."""
        if not isinstance(license_id, str) or not self.parse(license_id) or "," in license_id:
            return
        self.value.write(",".join(sorted(self.all() | {license_id.strip()})))

    def restore(self):
        """Rewrite any copy that is missing, empty or behind the union of the others."""
        ids = self.all()
        if not ids:
            return
        text = ",".join(sorted(ids))
        copies = self.value.read_all()
        if len(copies) < self.value.places() or any(copy != text for copy in copies):
            self.value.write(text)
