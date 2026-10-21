"""Reference of the machine ID v3 (ADR-0098) for tests, written from the contract in
docs/architecture/license-binding-contract.md and independent of the Go engine and the native
guard. The frozen Python backend still uses the older v2 ID and never sees this module."""
import ctypes
import hashlib

PREFIX = "scadabench-hwid-v3|"
DISPLAYED = ["03000200-0400-0500-0006-000700080009", "00020003-0004-0005-0006-000700080009",
             "12345678-1234-5678-90ab-cddeefaabbcc"]
SPACE = " \t\r\n\v\f"


def swap(hex32):
    def flip(text):
        return "".join(text[i:i + 2] for i in range(len(text) - 2, -2, -2))
    return flip(hex32[0:8]) + flip(hex32[8:12]) + flip(hex32[12:16]) + hex32[16:]


PLAIN = [text.replace("-", "") for text in DISPLAYED]
JUNK_UUIDS = set(PLAIN) | {swap(text) for text in PLAIN}


def clean_firmware(raw):
    value = raw.strip().lower()
    junk = (len(value) != 32 or any(c not in "0123456789abcdef" for c in value)
            or len({value[i:i + 2] for i in range(0, 32, 2)}) == 1 or value in JUNK_UUIDS)
    return "" if junk else value


def clean_board(raw):
    text = raw.strip(SPACE)
    if text.lower() in ("", "default string", "to be filled by o.e.m.", "none", "n/a", "0") or len(set(text)) <= 1:
        return ""
    return text


def compose(firmware, board, guid):
    firmware, board = clean_firmware(firmware), clean_board(board)
    extra = guid.strip().lower() if not firmware and not board else ""
    return hashlib.sha256((PREFIX + firmware + "|" + board + "|" + extra).encode()).hexdigest()


def structure(kind, length, *strings, serial_index=None, serial_at=7):
    """One SMBIOS structure: formatted part of `length` bytes, then its string set."""
    formatted = bytearray(length)
    formatted[0], formatted[1] = kind, length
    if serial_index is not None:
        formatted[serial_at] = serial_index
    text = b"".join(s.encode() + b"\0" for s in strings) + b"\0" if strings else b"\0\0"
    return bytes(formatted) + text


def blob(*structures):
    return bytes(8) + b"".join(structures) + structure(127, 4)


def raw_table():
    """The real RawSMBIOSData of this computer (Windows), or b""."""
    kernel32 = ctypes.windll.kernel32
    provider = 0x52534D42
    size = kernel32.GetSystemFirmwareTable(provider, 0, None, 0)
    if not size:
        return b""
    buffer = ctypes.create_string_buffer(size)
    kernel32.GetSystemFirmwareTable(provider, 0, buffer, size)
    return buffer.raw


def system_uuid(raw):
    data, pos = raw[8:], 0
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


def board_serial(raw):
    data, pos = raw[8:], 0
    while pos + 4 <= len(data):
        kind, length = data[pos], data[pos + 1]
        if length < 4 or pos + length > len(data):
            return ""
        end = pos + length
        while end + 1 < len(data) and not (data[end] == 0 and data[end + 1] == 0):
            end += 1
        if kind == 2 and length >= 8:
            texts = data[pos + length:end].split(b"\0") if end > pos + length else []
            index = data[pos + 7]
            return texts[index - 1].decode("latin-1") if 1 <= index <= len(texts) else ""
        if kind == 127:
            return ""
        pos = end + 2
    return ""


def machine_guid():
    import winreg
    key = winreg.OpenKey(winreg.HKEY_LOCAL_MACHINE, r"SOFTWARE\Microsoft\Cryptography", 0,
                         winreg.KEY_READ | winreg.KEY_WOW64_64KEY)
    return winreg.QueryValueEx(key, "MachineGuid")[0]


def machine_id():
    """This computer's v3 ID, read the way the Go engine and the native guard read it."""
    raw = raw_table()
    firmware, board = (system_uuid(raw), board_serial(raw)) if len(raw) > 8 else ("", "")
    guid = machine_guid() if not clean_firmware(firmware) and not clean_board(board) else ""
    return compose(firmware, board, guid)

