"""Validate the address range Find IEDs may scan: one host or at most a /24 network."""
import ipaddress

MAX_HOSTS = 256


def plan_range(text, own_ip=None):
    """Return {"range", "hosts", "count"} for "192.168.1.0/24", "192.168.1.7" or "192.168.1.20-40"."""
    text = str(text or "").strip()
    if not text:
        raise ValueError("Type a network such as 192.168.1.0/24, one address, or a range such as 192.168.1.20-40")
    try:
        if "-" in text:
            start_text, end_text = text.split("-", 1)
            start = ipaddress.IPv4Address(start_text.strip())
            end_part = end_text.strip()
            end = (ipaddress.IPv4Address(end_part) if "." in end_part
                   else ipaddress.IPv4Address(f"{str(start).rsplit('.', 1)[0]}.{int(end_part)}"))
            if int(end) < int(start):
                raise ValueError("The range must go from the lower to the higher address")
            hosts = [ipaddress.IPv4Address(value) for value in range(int(start), int(end) + 1)]
        else:
            network = ipaddress.IPv4Network(text, strict=False)
            if network.prefixlen < 24:
                raise ValueError("Scan at most 256 addresses (a /24 network) so the scan stays quick and polite")
            hosts = list(network.hosts()) if network.prefixlen < 31 else list(network)
    except ValueError as error:
        if "at most" in str(error) or "lower to the higher" in str(error):
            raise
        raise ValueError(f"'{text}' is not a valid address, range or network") from error
    if len(hosts) > MAX_HOSTS:
        raise ValueError("Scan at most 256 addresses (a /24 network) so the scan stays quick and polite")
    if any(host.is_multicast or host.is_unspecified for host in hosts):
        raise ValueError("Multicast and 0.0.0.0 addresses cannot be scanned")
    addresses = [str(host) for host in hosts if str(host) != own_ip]
    return {"range": text, "hosts": addresses, "count": len(addresses),
            "first": addresses[0] if addresses else "", "last": addresses[-1] if addresses else ""}


def suggest_range(ip, prefix):
    """The adapter's own network, narrowed to a /24 at most."""
    network = ipaddress.IPv4Network(f"{ip}/{max(int(prefix), 24)}", strict=False)
    return str(network)
