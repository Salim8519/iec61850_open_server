"""Salted hashes of the three machine parts, sent when a free trial is requested.

The server refuses a new trial only when at least two of the three parts match an
earlier trial (ADR-0046 addendum 2026-10-19). Raw identifiers never leave the PC:
part = SHA256_HEX("scadabench-trial-part-v1|" + kind + "|" + value), with the same
normalisation as the machine ID (hwid.py). Placeholder values that many unrelated
PCs share are sent as "" and never compared; keep the list equal to parts.go.
"""
import hashlib

PREFIX = "scadabench-trial-part-v1|"
# Displayed forms; the raw SMBIOS byte order is derived by swap_uuid_fields().
PLACEHOLDER_UUIDS = (
    "03000200-0400-0500-0006-000700080009",
    "00020003-0004-0005-0006-000700080009",
    "12345678-1234-5678-90ab-cddeefaabbcc",
)
JUNK_VOLUME_SERIALS = (0x00000000, 0xFFFFFFFF)


def part_hash(kind, value):
    return hashlib.sha256(f"{PREFIX}{kind}|{value}".encode("utf-8")).hexdigest()


def swap_uuid_fields(hex32):
    """Convert between displayed UUID order and raw SMBIOS bytes (first fields little-endian)."""
    if len(hex32) != 32:
        return hex32
    flip = lambda field: "".join(reversed([field[i:i + 2] for i in range(0, len(field), 2)]))  # noqa: E731
    return flip(hex32[0:8]) + flip(hex32[8:12]) + flip(hex32[12:16]) + hex32[16:]


def _placeholders():
    plain = [uuid.replace("-", "") for uuid in PLACEHOLDER_UUIDS]
    return set(plain) | {swap_uuid_fields(value) for value in plain}


def is_junk_firmware(raw_hex):
    """True for empty, malformed, one-repeated-byte (all 0 / all F) or placeholder UUIDs."""
    value = str(raw_hex or "").strip().lower()
    if len(value) != 32 or any(c not in "0123456789abcdef" for c in value):
        return True
    if len({value[i:i + 2] for i in range(0, 32, 2)}) == 1:
        return True
    return value in _placeholders()


def compose_parts(machine_guid, volume_serial, firmware_uuid):
    """The three part hashes; "" for a part that is unreadable or a known placeholder."""
    guid = str(machine_guid or "").strip().lower()
    parts = {"guid": part_hash("guid", guid) if guid else "", "volume": "", "firmware": ""}
    if volume_serial is not None and (volume_serial & 0xFFFFFFFF) not in JUNK_VOLUME_SERIALS:
        parts["volume"] = part_hash("volume", f"{volume_serial & 0xFFFFFFFF:08X}")
    if not is_junk_firmware(firmware_uuid):
        parts["firmware"] = part_hash("firmware", firmware_uuid.strip().lower())
    return parts


def _safely(read):
    try:
        return read()
    except (OSError, ValueError, AttributeError, ImportError):
        return None


def machine_parts():
    """This PC's part hashes; every unreadable part is simply left empty."""
    from . import hwid
    return compose_parts(_safely(hwid._machine_guid), _safely(hwid._volume_serial),
                         _safely(hwid._firmware_uuid) or "")
