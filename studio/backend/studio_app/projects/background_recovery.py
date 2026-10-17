"""Reopen the last autosave on a background thread so the app window can appear at once.

Recovery never starts IEDs. Until it finishes, engineering requests wait (see
api/startup_gate.py); a recovery that stopped to protect the autosave keeps them refused.
"""
import logging
import threading
from .startup_recovery import recover_startup


class BackgroundRecovery:
    def __init__(self, projects):
        self.projects = projects
        self.result = None
        self.error = None
        self.done = threading.Event()

    def start(self):
        threading.Thread(target=self._run, name="project-restore", daemon=True).start()
        return self

    def _run(self):
        try:
            self.result = recover_startup(self.projects)
        except Exception as error:  # Reported to every waiting request; never lost.
            logging.getLogger(__name__).exception("Startup recovery stopped")
            self.error = error
        finally:
            self.done.set()

    def wait(self, timeout=None):
        """True once recovery has finished (successfully or not)."""
        return self.done.wait(timeout)
