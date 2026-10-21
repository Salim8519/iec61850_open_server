"""Apply "Where GOOSE goes": pick the adapter, check the risk and confirm sharing (ADR-0073)."""
from studio_app.contracts.errors import StudioError
from studio_app.domain.goose_reach import (
    advanced_choices, pick_uplink, reach_status, share_hold, share_risk)
from studio_app.networking.adapters import list_adapters
from studio_app.networking.npcap import npcap_status
from studio_app.projects.goose_reach import save_reach

CONFIRM = "Read the warning and confirm before sharing GOOSE with the network"
MOVED = "The network adapter changed. Review the warning again before sharing GOOSE."


class GooseShare:
    def __init__(self, manager, adapters=list_adapters, npcap=npcap_status):
        self.manager = manager
        self.adapters = adapters
        self.npcap = npcap
        self.current = {"uplink": {}, "hold": ""}

    def _resolve(self, network):
        try:
            adapters = self.adapters()
        except (OSError, RuntimeError):
            adapters = []
        return adapters, pick_uplink(adapters, network)

    def _heard_real_goose(self):
        return bool(self.manager.engine.stats().get("delivered_wire")) if self.manager.engine.running else False

    def preview(self, project):
        """Everything the "Where GOOSE goes" panel and its warning need."""
        network = project["network"]
        adapters, uplink = self._resolve(network)
        installed = self.npcap()["installed"]
        engine = self.manager.engine.snapshot()
        state, message = reach_status(network, uplink, share_hold(network, uplink, installed), engine)
        return {"reach": network.get("goose_reach", "inside"), "uplink": uplink,
                "uplink_guid": network.get("goose_uplink_guid", ""), "choices": advanced_choices(adapters),
                "risk": share_risk(network, uplink, project["ieds"], self._heard_real_goose()),
                "npcap": installed, "running": engine["running"], "state": state, "message": message,
                "conflicts": engine["conflicts"]}

    def apply(self, projects, body):
        """Save the user's choice; sharing needs confirm:true and the adapter they saw (GOOSE-19)."""
        reach = body.get("reach", "inside")
        uplink_guid = body.get("uplink_guid")
        network = {**projects.project["network"], "goose_reach": reach}
        if uplink_guid is not None:
            network["goose_uplink_guid"] = str(uplink_guid).upper()
        share_guid = ""
        if reach == "outside":
            if body.get("confirm") is not True:
                raise StudioError(CONFIRM)
            _, uplink = self._resolve(network)
            if not self.npcap()["installed"]:
                raise StudioError(self.npcap()["message"])
            if not uplink["guid"]:
                raise StudioError(uplink["reason"])
            if str(body.get("adapter_guid") or "").upper() != uplink["guid"]:
                raise StudioError(MOVED)
            share_guid = uplink["guid"]
        save_reach(projects, reach, network.get("goose_uplink_guid", ""), share_guid)
        if self.manager.engine.running:
            self.start(projects.project)
        return self.preview(projects.project)

    def start(self, project):
        """At Run, or after a change while running: listen on the adapter, share when allowed."""
        network = project["network"]
        _, uplink = self._resolve(network)
        installed = self.npcap()["installed"]
        hold = share_hold(network, uplink, installed)
        self.current = {"uplink": uplink, "hold": hold}
        listen = uplink["guid"] if installed else ""
        share = network.get("goose_reach") == "outside" and not hold
        self.manager.engine.uplink(listen, share)

    def status(self, project):
        """(state, message) for the GOOSE traffic view while running."""
        uplink = self.current.get("uplink") or {}
        return reach_status(project["network"], uplink, self.current.get("hold", ""), self.manager.engine.snapshot())

    def choose(self, body):
        """The user's answer to a conflict: stop this IED's GOOSE outside, or continue (confirmed)."""
        choice = body.get("choice")
        if choice not in {"stop", "continue"}:
            raise StudioError("Choose 'stop' or 'continue'")
        if choice == "continue" and body.get("confirm") is not True:
            raise StudioError("Confirm that ScadaBench may keep sending this IED's GOOSE next to the real device")
        try:
            return self.manager.engine.gate(str(body.get("ied_id") or ""), choice)
        except (RuntimeError, ValueError) as error:
            raise StudioError(str(error)) from error
