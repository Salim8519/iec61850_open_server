"""Resolve installed and development application paths."""
from dataclasses import dataclass
from pathlib import Path
import sys


@dataclass
class Settings:
    directory: Path
    native_executable: Path
    frontend: Path

    @property
    def live_executable(self):
        """The read-only live client ships beside the IED worker in both modes."""
        return self.native_executable.with_name("studio_live.exe")

    @classmethod
    def defaults(cls):
        if getattr(sys, "frozen", False):
            root = Path(sys.executable).parent
            resources = Path(sys._MEIPASS)
            return cls(root / "data", resources / "native/studio_ied.exe", resources / "frontend")
        root = Path(__file__).resolve().parents[2]
        return cls(root / "data", root / "native/build/Release/studio_ied.exe", root / "frontend/dist")
