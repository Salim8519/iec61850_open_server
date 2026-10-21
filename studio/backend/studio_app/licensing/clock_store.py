"""Keep each license clock value in three places so deleting one copy changes nothing.

Places: a dot-file in the license folder, a REG_SZ value under HKCU\\Software\\ScadaBench,
and a string field of license_state.json. Readers decide on the copies that remain;
every read or write failure (antivirus, Controlled Folder Access, a locked registry) is
ignored silently, so a blocked place can never stop ScadaBench. The native guard reads
the same places (license_clock.cpp). Values: time_anchor, server_time, grace_start and
revoked_ids (the cancelled license IDs, see revoked_ids.py).
"""
import json
import os
from pathlib import Path

REGISTRY_KEY = r"Software\ScadaBench"
REGISTRY_NAMES = {"time_anchor": "TimeAnchor", "server_time": "ServerTime", "grace_start": "GraceStart",
                  "revoked_ids": "RevokedLicenses"}


class WindowsRegistry:
    """HKCU\\Software\\ScadaBench string values (per user, no administrator rights)."""

    def __init__(self, key=REGISTRY_KEY):
        self.key = key

    def get(self, name):
        try:
            import winreg
            with winreg.OpenKey(winreg.HKEY_CURRENT_USER, self.key) as key:
                value, kind = winreg.QueryValueEx(key, name)
            return str(value) if kind == winreg.REG_SZ else None
        except (OSError, ImportError):
            return None

    def set(self, name, text):
        try:
            import winreg
            with winreg.CreateKeyEx(winreg.HKEY_CURRENT_USER, self.key, 0, winreg.KEY_SET_VALUE) as key:
                winreg.SetValueEx(key, name, 0, winreg.REG_SZ, str(text))
        except (OSError, ImportError):
            pass


def _write_text(path, text):
    try:
        path.parent.mkdir(parents=True, exist_ok=True)
        temporary = path.with_name(path.name + ".tmp")
        temporary.write_text(text, encoding="utf-8")
        os.replace(temporary, path)
    except OSError:
        pass


class ClockValue:
    """One named value kept in the file, the registry and license_state.json."""

    def __init__(self, name, home, registry=None):
        self.name = name
        self.file = Path(home) / f".{name}"
        self.state_file = Path(home) / "license_state.json"
        self.registry = registry
        self.registry_name = REGISTRY_NAMES[name]

    def _state(self):
        try:
            state = json.loads(self.state_file.read_text(encoding="utf-8"))
            return state if isinstance(state, dict) else {}
        except (OSError, ValueError):
            return {}

    def read_all(self):
        """The copies that exist, as text (missing or unreadable places are skipped)."""
        copies = []
        try:
            copies.append(self.file.read_text(encoding="utf-8").strip())
        except (OSError, ValueError):
            pass
        if self.registry is not None:
            copies.append(self.registry.get(self.registry_name))
        state_value = self._state().get(self.name)
        copies.append(state_value if isinstance(state_value, str) else None)
        return [copy for copy in copies if copy]

    def places(self):
        return 3 if self.registry is not None else 2

    def read_ints(self):
        numbers = []
        for copy in self.read_all():
            try:
                numbers.append(int(copy))
            except ValueError:
                continue
        return numbers

    def write(self, text):
        text = str(text)
        _write_text(self.file, text)
        if self.registry is not None:
            self.registry.set(self.registry_name, text)
        state = self._state()
        if state.get(self.name) != text:
            state[self.name] = text
            _write_text(self.state_file, json.dumps(state))
