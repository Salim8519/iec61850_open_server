"""Poll Windows every few milliseconds and log when things appear and disappear.

Watched: addresses on the Lab card (winaddr), TCP listeners (winlisten) and the instance's
worker processes (its Job object; handles held so exact kernel creation/exit times survive).
Every change is one record (moment, what, key, change) on the winclock scale.

    watch = Watcher(engine.job, prefix="198.18.20.").start()
    watch.first("address", "added")    # {ip: moment}
    watch.stop()
"""
import os
import threading
import time

import winaddr
import winlisten
from winclock import now
from winproc import Tracker

WORKERS = ("studio_ied.exe", "studio_goose.exe")


class Watcher:
    def __init__(self, job, prefix=None, interval=0.002):
        prefix = prefix or os.environ.get("SCADABENCH_TRIAL_RANGE", "198.18.20") + "."
        self.job, self.prefix, self.interval = job, prefix, interval
        self.card = winaddr.lab_card_index()
        self.tracker = Tracker()
        self.records, self.lock = [], threading.Lock()
        self.addresses, self.listening, self.processes, self.names = {}, {}, {}, {}
        self.running = False
        self.thread = None

    def start(self):
        self._poll()  # the starting state is not a change
        with self.lock:
            self.records.clear()
        self.running = True
        self.thread = threading.Thread(target=self._loop, daemon=True)
        self.thread.start()
        return self

    def _loop(self):
        while self.running:
            self._poll()
            time.sleep(self.interval)

    def _note(self, moment, what, key, change):
        with self.lock:
            self.records.append((moment, what, key, change))

    def _diff(self, what, before, after, moment):
        for key in after.keys() - before.keys():
            self._note(moment, what, key, "added")
        for key in before.keys() - after.keys():
            self._note(moment, what, key, "removed")

    def _poll(self):
        moment = now()
        addresses = winaddr.addresses(self.prefix, self.card)
        listening = winlisten.listeners(self.prefix)
        processes = {pid: name for pid, name in self.job.processes().items() if name.lower() in WORKERS}
        for pid in processes.keys() - self.processes.keys():
            self.tracker.hold(pid)
            self.names[pid] = processes[pid].lower()
        self._diff("address", self.addresses, addresses, moment)
        self._diff("listen", self.listening, listening, moment)
        self._diff("process", self.processes, processes, moment)
        for ip, row in addresses.items():
            previous = self.addresses.get(ip)
            if previous and previous.state != row.state:
                self._note(moment, "address_state", ip, row.state)
        self.addresses, self.listening, self.processes = addresses, listening, processes

    def mark(self):
        """Forget earlier records (call just before the action being timed)."""
        with self.lock:
            self.records.clear()

    def first(self, what, change):
        """{key: first moment} of this kind of change since the last mark()."""
        found = {}
        with self.lock:
            for moment, kind, key, kind_change in self.records:
                if kind == what and kind_change == change and key not in found:
                    found[key] = moment
        return found

    def changes(self, what):
        """[(moment, key, change)] of this kind since the last mark(), oldest first."""
        with self.lock:
            return [(moment, key, change) for moment, kind, key, change in self.records if kind == what]

    def kernel_times(self, name="studio_ied.exe", since=0.0):
        """{pid: (created, exited)} of every worker with this image name seen by the watcher
        that was still alive at `since` (exited None while it runs)."""
        found = {}
        for pid, image in list(self.names.items()):
            if image == name:
                created, exited = self.tracker.times(pid)
                if created and (exited is None or exited >= since):
                    found[pid] = (created, exited)
        return found

    def windows_created(self):
        """{ip: Windows' own creation time} of the Lab-card addresses present now."""
        return {ip: row.created for ip, row in self.addresses.items()}

    def stop(self):
        self.running = False
        if self.thread:
            self.thread.join(2)
        self.tracker.close()
