"""Save where GOOSE goes. Allowed while running: it never changes addresses or IEDs."""
from studio_app.contracts.errors import StudioError
from studio_app.domain.network_settings import validate_network

LAB_TITLE = "GOOSE now stays inside ScadaBench unless you share it."
LAB_NOTICE = ("GOOSE now stays inside ScadaBench by default. Before, a lab-network project also sent "
              "its GOOSE to the network. To keep doing that, open Settings > Network & GOOSE and "
              "switch on \"Also share with the network\".")


def lab_notice(raw_network, network):
    """True once for a lab project saved before "Where GOOSE goes" existed (ADR-0073)."""
    return network.get("mode") == "lab" and "goose_reach" not in (raw_network or {})


def save_reach(service, reach, uplink_guid, share_guid):
    with service.lock:
        network = {**service.project["network"], "goose_reach": reach,
                   "goose_uplink_guid": uplink_guid or "", "goose_share_guid": share_guid or ""}
        try:
            candidate = validate_network(network)
        except ValueError as error:
            raise StudioError(str(error)) from error
        service.project["network"] = candidate
        service.save()
        service.changed()
        return candidate
