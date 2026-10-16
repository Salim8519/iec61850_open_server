"""Deterministic machine ID shared with the native workers (license_guard.cpp).

HWID = SHA256_HEX("substation-studio-hwid-v1|" + MachineGuid + "|" + system volume serial)
Both inputs are read in well under a millisecond, without WMI.
"""
import hashlib
import os

PREFIX = "substation-studio-hwid-v1|"


def compose(machine_guid, volume_serial):
    """Combine the two raw identifiers exactly as the native code does."""
    text = f"{PREFIX}{machine_guid.strip().lower()}|{volume_serial & 0xFFFFFFFF:08X}"
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


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


def machine_id():
    """Return this computer's 64-hex machine ID (Windows only)."""
    if os.name != "nt":
        raise OSError("Machine licensing is available on Windows only")
    return compose(_machine_guid(), _volume_serial())
