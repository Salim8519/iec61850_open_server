"""Guided lab-network workflow: preview, apply through the helper, and clean up."""
import hashlib
import json
import threading
import time
from pathlib import Path
from uuid import uuid4
from studio_app.contracts.errors import StudioError
from studio_app.projects.ied_settings import update_network
from . import adapters as adapter_list
from .address_plan import on_link, plan_addresses, validate_manual
from .arp_probe import addresses_in_use
from .change_plan import build_plan, describe, describe_removal
from .elevation import ElevationCancelled, helper_command, run_elevated
from .journal import NetworkJournal, item_key


class NetworkService:
    def __init__(self, projects, directory, program, adapters=None, rules=None, elevate=None, probe=None):
        self.projects = projects
        self.directory = Path(directory)
        self.program = Path(program)
        self.journal = NetworkJournal(self.directory)
        self._list_adapters = adapters or adapter_list.list_adapters
        self._list_rules = rules or adapter_list.list_studio_rules
        self._elevate = elevate or (lambda request, digest: run_elevated(
            *helper_command(request, digest, self.program)))
        self._probe = probe or addresses_in_use
        self.lock = threading.Lock()
        self.cache = (0.0, None)
        self.previews = {}

    def adapters(self, refresh=False):
        stamp, value = self.cache
        if refresh or value is None or time.monotonic() - stamp > 10:
            value = self._list_adapters()
            self.cache = (time.monotonic(), value)
        return value

    def adapter(self, guid):
        found = next((item for item in self.adapters() if item["guid"] == str(guid).upper()), None)
        if found is None:
            raise StudioError("That network adapter is no longer available. Refresh the adapter list.")
        return found

    def preview(self, guid, strategy="scd", manual=None):
        adapter = self.adapter(guid)
        ieds = [ied for ied in self.projects.project["ieds"] if ied.get("role", "simulated") == "simulated"]
        if not ieds:
            raise StudioError("There are no simulated IEDs to put on the network")
        rows = plan_addresses(ieds, adapter, self.adapters(), strategy)
        for row in rows:
            override = (manual or {}).get(row["ied_id"])
            if override:
                try:
                    row["proposed"], row["prefix"] = validate_manual(override.get("ip"), override.get("prefix"))
                    row.update(status="ok", hint="Address typed by you.")
                except ValueError as error:
                    row.update(status="invalid_manual", hint=str(error))
        own = {item["ip"] for item in adapter["addresses"]}
        candidates = [row["proposed"] for row in rows if row["status"] == "ok" and row["proposed"] not in own]
        reachable = [address for address in candidates if on_link(address, adapter)]
        in_use = self._probe(reachable)
        for row in rows:
            if row["proposed"] in in_use:
                row.update(status="in_use_on_network", hint=f"Another device already answers on {row['proposed']}. "
                           "Pick a different address so the two do not clash.")
            elif row["status"] == "ok" and not row["hint"]:
                row["hint"] = ("No other device answered on this address." if row["proposed"] in reachable else
                               "Not used on this PC. Devices in this new network can only be checked after the "
                               "address is added; Windows reports a clash then.")
        ports = {ied.get("port", 102) for ied in ieds}
        plan = build_plan(adapter, rows, ports, self.program, self._list_rules())
        blocked = [row for row in rows if row["status"] not in {"ok", "already_on_adapter"}]
        result = {"adapter": adapter, "rows": rows, "items": [{**item, "description": describe(item)}
                                                              for item in plan["items"]],
                  "fingerprint": plan["fingerprint"], "ready": not blocked,
                  "needs_admin": any(item["action"] == "add" for item in plan["items"])}
        with self.lock:
            self.previews[plan["fingerprint"]] = {"guid": adapter["guid"], "strategy": strategy,
                                                  "manual": manual or {}, "rows": rows}
        return result

    def apply(self, fingerprint, confirm):
        if confirm is not True:
            raise StudioError("Review the change list and confirm it before applying")
        with self.lock:
            stored = self.previews.get(fingerprint)
        if stored is None:
            raise StudioError("This preview has expired. Review the changes again.", code="conflict")
        self.cache = (0.0, None)
        fresh = self.preview(stored["guid"], stored["strategy"], stored["manual"])
        if fresh["fingerprint"] != fingerprint:
            raise StudioError("The network changed since you reviewed it. Review the new change list.",
                              code="conflict", details={"fingerprint": fresh["fingerprint"]})
        if not fresh["ready"]:
            raise StudioError("Fix the highlighted address problems before applying")
        to_add = [item for item in fresh["items"] if item["action"] == "add" and item["owned"]]
        results = self._run_helper("apply", to_add, fingerprint) if to_add else []
        failed = [entry for entry in results if not entry["ok"]]
        if not failed:
            self._enter_lab_mode(fresh)
        self.cache = (0.0, None)
        return {"results": results, "applied": not failed,
                "message": "Lab network is ready. Press Run to start the IEDs on the network." if not failed
                else "Some changes failed; nothing else was switched. See each line for the reason."}

    def _enter_lab_mode(self, preview):
        with self.projects.lock:
            for row in preview["rows"]:
                ied = self.projects.find_ied(row["ied_id"])
                ied["lab_ip"] = row["proposed"]
            adapter = preview["adapter"]
            update_network(self.projects, {"mode": "lab", "adapter_guid": adapter["guid"],
                                           "adapter_label": adapter["name"]})

    def _run_helper(self, action, items, fingerprint):
        folder = self.directory / "network" / "requests"
        folder.mkdir(parents=True, exist_ok=True)
        request = folder / f"{action}-{uuid4().hex}.json"
        clean = [{key: value for key, value in item.items() if key != "description"} for item in items]
        payload = json.dumps({"action": action, "fingerprint": fingerprint, "items": clean}).encode("utf-8")
        request.write_bytes(payload)
        try:
            self._elevate(request, hashlib.sha256(payload).hexdigest())
        except ElevationCancelled as error:
            raise StudioError(str(error)) from error
        result_path = request.with_suffix(".result.json")
        if not result_path.exists():
            raise StudioError("The network helper finished without a result; nothing was recorded as changed")
        outcome = json.loads(result_path.read_text(encoding="utf-8"))
        if outcome.get("refused"):
            raise StudioError("The network helper refused the request: " + outcome.get("reason", ""))
        by_key = {entry["key"]: entry for entry in outcome["results"]}
        for item in clean:
            entry = by_key.get(item_key(item), {"ok": False, "message": "No result"})
            event = ("applied" if action == "apply" else "removed") if entry["ok"] else "failed"
            self.journal.append(event, item, fingerprint, entry.get("message", ""))
        label = describe if action == "apply" else describe_removal
        return [{**by_key.get(item_key(item), {}), "description": label(item)} for item in clean]

    def _present(self, item):
        if item["kind"] == "alias":
            return any(address["ip"] == item["ip"] for adapter in self.adapters(refresh=True)
                       if adapter["guid"] == item["adapter_guid"] for address in adapter["addresses"])
        return item["name"] in self._list_rules()

    def leftovers(self):
        return self.journal.leftovers(self._present)

    def cleanup_preview(self):
        items = [{**item, "action": "remove", "description": describe_removal(item)} for item in self.leftovers()]
        digest = hashlib.sha256(json.dumps(items, sort_keys=True).encode()).hexdigest()
        return {"items": items, "fingerprint": digest, "needs_admin": bool(items)}

    def cleanup(self, fingerprint, confirm):
        if confirm is not True:
            raise StudioError("Confirm the clean-up list first")
        preview = self.cleanup_preview()
        if preview["fingerprint"] != fingerprint:
            raise StudioError("The list of ScadaBench changes is different now. Review it again.", code="conflict")
        results = self._run_helper("remove", preview["items"], fingerprint) if preview["items"] else []
        if self.projects.project["network"]["mode"] == "lab":
            update_network(self.projects, {"mode": "loopback"})
        self.cache = (0.0, None)
        return {"results": results, "message": "ScadaBench's network changes were removed. IEDs are back on "
                "'This PC only' mode."}
