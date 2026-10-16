"""An OS-held Windows lock prevents concurrent launchers overwriting one autosave."""
import msvcrt


class DirectoryLease:
    def __init__(self, directory):
        self.directory = directory
        self.handle = None

    def __enter__(self):
        self.directory.mkdir(parents=True, exist_ok=True)
        path = self.directory / "studio.lock"
        if path.is_symlink():
            raise RuntimeError("ScadaBench lock file is a link; inspect the data folder before starting")
        handle = path.open("a+b")
        try:
            handle.seek(0, 2)
            if handle.tell() == 0:
                handle.write(b"0")
                handle.flush()
            handle.seek(0)
            msvcrt.locking(handle.fileno(), msvcrt.LK_NBLCK, 1)
        except OSError as error:
            handle.close()
            raise RuntimeError("This ScadaBench data folder is already open or cannot be locked. "
                               "Close its other launcher before opening it again.") from error
        self.handle = handle
        return self

    def __exit__(self, *_args):
        if self.handle:
            try:
                self.handle.seek(0)
                msvcrt.locking(self.handle.fileno(), msvcrt.LK_UNLCK, 1)
            finally:
                self.handle.close()
                self.handle = None
