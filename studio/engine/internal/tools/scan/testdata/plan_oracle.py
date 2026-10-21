"""plan_range on random range texts (networks, masks, ranges, junk) and preview port checks."""
import os
import sys

sys.path.insert(0, os.environ["STUDIO_ORACLE_LIB"])
from oracle_lib import case, rng, seed  # noqa: E402
from studio_app.domain.scan_range import plan_range  # noqa: E402
from studio_app.tools.ied_scan import ScanService  # noqa: E402

random = rng(seed())
FIXED = ["", " ", "192.168.1.0/24", "192.168.1.7", "192.168.1.20-40", "10.0.0.0/23", "10.0.0.0/255.255.255.0",
         "10.0.0.0/0.0.0.255", "10.0.0.5/31", "10.0.0.5/32", "224.0.0.1", "0.0.0.0", "0.0.0.0/24", "1.2.3.4-1.2.3.1",
         "1.2.3.4-1.2.4.4", "1.2.3.250-260", "1.2.3.4- +7", "1.2.3.4-1_0", "1.2.3.4-٧", "01.2.3.4", "1.2.3",
         "1.2.3.4/33", "1.2.3.4/024", "1.2.3.4//24", "1.2.3.4/", "1.2.3.4/255.0.255.0", "239.255.255.250-255",
         "1.2.3.4-5-6", " 1.2.3.4　", "\x1c1.2.3.4", "1.2.3.4 - 1.2.3.9", "a.b.c.d", "1.2.3.0/24-30",
         "255.255.255.0/24", "1.2.3.4-", "-1.2.3.4", "1.2.3.4 /24", "223.255.255.255-224.0.0.0", None, 5, True, 0,
         [], ["x"]]


def text():
    roll = random.random()
    if roll < 0.3:
        return random.choice(FIXED)
    octets = [str(random.choice([0, 1, 10, 127, 192, 223, 224, 255, 256, random.randint(0, 255)])) for _ in range(4)]
    base = ".".join(octets)
    if roll < 0.55:
        return f"{base}/{random.choice([0, 8, 23, 24, 25, 28, 30, 31, 32, 33, '255.255.255.0', '0.0.0.15', 'x'])}"
    if roll < 0.8:
        # A far end address would make Python build millions of hosts; Go refuses those at once.
        return f"{base}-{random.choice([str(random.randint(0, 300)), base, ' 9'])}"
    return base


for number in range(int(sys.argv[2]) if len(sys.argv) > 2 else 3000):
    value = text()
    own = random.choice([None, None, "192.168.1.7", "10.0.0.5", "1.2.3.5"])
    case("plan", [value, own], lambda: plan_range(value, own))
    port = random.choice([102, True, False, 0, 65535, 65536, "102", 1.0, None, 2 ** 70])
    case("preview", [value, port], lambda: ScanService(None).preview(value, port))
