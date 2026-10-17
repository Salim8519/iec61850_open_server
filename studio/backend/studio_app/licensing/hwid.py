"""Deterministic machine ID shared with the native workers (license_guard.cpp).

HWID = SHA256_HEX("substation-studio-hwid-v2|" + MachineGuid + "|" + volume serial
                  + "|" + SMBIOS system UUID). The SMBIOS UUID is a firmware-level
identifier, far harder to change than the registry MachineGuid or the volume serial,
so adding it makes a license much harder to move by spoofing one value. Every input
is read in well under a millisecond, without WMI. Keep this byte-identical to the
native compose_hwid(); a parity test compares both on the same machine.

Legacy v1 (no SMBIOS UUID) is still accepted for certificates issued before
LEGACY_ISSUED_BEFORE, so licenses activated by older builds survive an update without
spending a computer move (ADR-0051). New activations always bind v2.
"""
import hashlib
import os

PREFIX = "substation-studio-hwid-v2|"
LEGACY_PREFIX = "substation-studio-hwid-v1|"
LEGACY_ISSUED_BEFORE = 1814400000  # 2027-07-01T00:00:00Z; keep equal to license_guard.cpp


def compose(machine_guid, volume_serial, firmware_uuid):
    """Combine the three raw identifiers exactly as the native code does."""
    text = (f"{PREFIX}{machine_guid.strip().lower()}|{volume_serial & 0xFFFFFFFF:08X}"
            f"|{firmware_uuid}")
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def compose_legacy(machine_guid, volume_serial):
    """The v1 machine ID that builds before HWID v2 bound licenses to."""
    text = f"{LEGACY_PREFIX}{machine_guid.strip().lower()}|{volume_serial & 0xFFFFFFFF:08X}"
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def bound_hwid(certificate_hwid, issued_at, current, legacy):
    """Machine ID a certificate is checked against: legacy only for old certificates."""
    if legacy and certificate_hwid == legacy and certificate_hwid != current \
            and int(issued_at) < LEGACY_ISSUED_BEFORE:
        return legacy
    return current


def _machine_guid():
    import winreg
    access = winreg.KEY_READ | winreg.KEY_WOW64_64KEY
    with winreg.OpenKey(winreg.HKEY_LOCAL_MACHINE, r"SOFTWARE\Microsoft\Cryptography", 0, access) as key:
        value, _ = winreg.QueryValueEx(key, "MachineGuid")
    return str(value)


def _volume_serial():
    import ctypes
    from ctypes import wintypes
    root = os.environ.get("SystemDrive", "C:").rstrip("\\") + "\\"
    serial = wintypes.DWORD(0)
    ok = ctypes.windll.kernel32.GetVolumeInformationW(
        ctypes.c_wchar_p(root), None, 0, ctypes.byref(serial), None, None, None, 0)
    if not ok:
        raise OSError(f"Cannot read the volume serial number of {root}")
    return serial.value


def _smbios_system_uuid(raw):
    """Lowercase hex of the 16 raw UUID bytes from the SMBIOS Type 1 table, or ''.

    Raw bytes are used verbatim (no field reordering) so Python and C++ always agree.
    """
    data = raw[8:]  # skip the 8-byte RawSMBIOSData header (4 bytes + DWORD length)
    pos = 0
    while pos + 4 <= len(data):
        kind, length = data[pos], data[pos + 1]
        if length < 4:
            break
        if kind == 1 and pos + 24 <= len(data):
            return data[pos + 8:pos + 24].hex()
        pos += length
        while pos + 1 < len(data) and not (data[pos] == 0 and data[pos + 1] == 0):
            pos += 1
        pos += 2
    return ""


def _firmware_uuid():
    import ctypes
    rsmb = 0x52534D42  # 'RSMB' raw SMBIOS firmware table provider
    kernel32 = ctypes.windll.kernel32
    size = kernel32.GetSystemFirmwareTable(rsmb, 0, None, 0)
    if not size:
        return ""
    buffer = ctypes.create_string_buffer(size)
    if not kernel32.GetSystemFirmwareTable(rsmb, 0, buffer, size):
        return ""
    return _smbios_system_uuid(buffer.raw)


def machine_id():
    """Return this computer's 64-hex machine ID (Windows only)."""
    if os.name != "nt":
        raise OSError("Machine licensing is available on Windows only")
    return compose(_machine_guid(), _volume_serial(), _firmware_uuid())


def legacy_machine_id():
    """Return this computer's v1 machine ID (Windows only)."""
    if os.name != "nt":
        raise OSError("Machine licensing is available on Windows only")
    return compose_legacy(_machine_guid(), _volume_serial())
