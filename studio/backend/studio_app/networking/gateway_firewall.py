"""Open the station gateway to other computers through Windows Firewall (ADR-0062).

Same contract as the lab network: an exact, fingerprinted preview; nothing changes until the
user confirms; the elevated helper applies it; every change is journaled and the lab-network
clean-up removes ScadaBench's rule and switches disabled Block rules on again."""
import threading
from studio_app.contracts.errors import StudioError
from .change_plan import describe, gateway_plan
from .firewall_rules import blocking_rules, list_gateway_rules


class GatewayFirewall:
    def __init__(self, network, program, rules=None, blocking=None):
        self.network = network          # NetworkService: helper, journal and clean-up
        self.program = program          # path of studio_gateway.exe
        self._rules = rules or list_gateway_rules
        self._blocking = blocking or blocking_rules
        self.lock = threading.Lock()
        self.previews = set()

    def preview(self):
        try:
            plan = gateway_plan(self.program, self._rules(), self._blocking(str(self.program)))
        except (OSError, RuntimeError, ValueError) as error:
            raise StudioError(f"Windows Firewall could not be checked: {error}") from error
        items = [{**item, "description": describe(item)} for item in plan["items"]]
        open_ = all(item["action"] == "keep" for item in items)
        with self.lock:
            self.previews.add(plan["fingerprint"])
        return {"items": items, "fingerprint": plan["fingerprint"], "open": open_, "needs_admin": not open_,
                "blocked": any(item["kind"] == "unblock" for item in items)}

    def apply(self, fingerprint, confirm):
        if confirm is not True:
            raise StudioError("Review the Windows Firewall changes and confirm them first")
        with self.lock:
            known = fingerprint in self.previews
        if not known:
            raise StudioError("This preview has expired. Review the changes again.", code="conflict")
        fresh = self.preview()
        if fresh["fingerprint"] != fingerprint:
            raise StudioError("Windows Firewall changed since you reviewed it. Review the new list.",
                              code="conflict", details={"fingerprint": fresh["fingerprint"]})
        to_add = [item for item in fresh["items"] if item["action"] == "add" and item["owned"]]
        results = self.network._run_helper("apply", to_add, fingerprint) if to_add else []
        failed = [entry for entry in results if not entry.get("ok")]
        return {"results": results, "applied": not failed,
                "message": "Done: Settings › Network › Clean up undoes this."
                if not failed else "Some changes failed; see each line for the reason."}
