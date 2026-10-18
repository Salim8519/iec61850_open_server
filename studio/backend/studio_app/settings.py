"""Resolve installed and development application paths."""
from dataclasses import dataclass
import os
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

    @staticmethod
    def packaged_data_directory(executable_folder, local_app_data):
        """A data folder beside the exe wins (portable copies and existing users);
        otherwise installed copies keep data per user, because Program Files is read-only."""
        portable = Path(executable_folder) / "data"
        if portable.is_dir() or not local_app_data:
            return portable
        return Path(local_app_data) / "ScadaBench" / "data"

    @classmethod
    def defaults(cls):
        if getattr(sys, "frozen", False):
            root = Path(sys.executable).parent
            resources = Path(sys._MEIPASS)
            packed = resources / "frontend.pak"
            frontend = packed if packed.is_file() else resources / "frontend"
            from studio_app.launcher import portable_runtime
            if portable_runtime.is_single_file(root, resources):
                directory = portable_runtime.data_directory(root)
                native = portable_runtime.stage_native(resources / "native", directory)
                return cls(directory, native / "studio_ied.exe", frontend)
            directory = cls.packaged_data_directory(root, os.environ.get("LOCALAPPDATA"))
            return cls(directory, resources / "native/studio_ied.exe", frontend)
        root = Path(__file__).resolve().parents[2]
        return cls(root / "data", root / "native/build/Release/studio_ied.exe", root / "frontend/dist")
