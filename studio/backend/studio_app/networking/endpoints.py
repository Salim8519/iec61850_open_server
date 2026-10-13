"""Check simulated-IED listener endpoints before Run, in loopback or lab mode."""
import ipaddress
import socket
from studio_app.contracts.errors import StudioError
from studio_app.domain.network_settings import effective_endpoint


def preflight(ieds, network, adapter_addresses=()):
    """Raise a plain, fixable error unless every endpoint can be bound safely."""
    endpoints = set()
    allowed = set(adapter_addresses)
    for ied in ieds:
        host, port = effective_endpoint(ied, network)
        address = ipaddress.ip_address(host)
        if address.version != 4:
            raise StudioError(f"{ied['name']}: only IPv4 addresses are supported")
        if not address.is_loopback:
            if network["mode"] != "lab":
                raise StudioError(f"{ied['name']} uses {address}, but the project is in 'This PC only' "
                                  "mode. Set up the lab network first (Network tab).")
            if str(address) not in allowed:
                label = network.get("adapter_label") or "the selected adapter"
                raise StudioError(f"{ied['name']} needs address {address}, which is not set up on {label}. "
                                  "Open Network > Set up lab network and apply the address plan.")
        port = int(port)
        if not 1 <= port <= 65535:
            raise StudioError("MMS port must be between 1 and 65535")
        endpoint = str(address), port
        if endpoint in endpoints:
            raise StudioError(f"Two IEDs use {address}:{port}; give each IED its own address")
        endpoints.add(endpoint)
        try:
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
                if hasattr(socket, "SO_EXCLUSIVEADDRUSE"):
                    listener.setsockopt(socket.SOL_SOCKET, socket.SO_EXCLUSIVEADDRUSE, 1)
                listener.bind(endpoint)
        except OSError as error:
            raise StudioError(f"{ied['name']} cannot listen on {address}:{port} because another "
                              f"program is using it ({error}). Close that program or change the port.") from error
