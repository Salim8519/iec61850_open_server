"""An IEC 60870-5-104 controlled station for trial R7, written from the standard (no lib60870).

It answers STARTDT and TESTFR, confirms a station interrogation, sends `count` single points
(M_SP_NA_1, 60 per ASDU) and the termination, and acknowledges a counter interrogation
(C_CI_NA_1) on the link only, never answering it, like a device without counters that ignores
it. It serves one master connection at a time, again and again, until close(). Based on
studio/tests/protocol/iec104_standin.py.
"""
import socket
import struct
import threading

C_IC, C_CI, M_SP = 100, 101, 1
ACT_CON, ACT_TERM, INROGEN = 7, 10, 20
PER_ASDU = 60


class SilentCounterStation:
    def __init__(self, host, port, count, ca=1):
        self.count, self.ca, self.counters_asked = count, ca, 0
        self.listener = socket.create_server((host, port))
        self.listener.settimeout(0.5)
        self.running = True
        threading.Thread(target=self._serve, daemon=True).start()

    def close(self):
        self.running = False
        self.listener.close()

    def _serve(self):
        while self.running:
            try:
                connection, _ = self.listener.accept()
            except OSError:
                continue
            self.sent, self.received = 0, 0
            with connection:
                connection.settimeout(120)
                try:
                    while self.running:
                        header = self._exact(connection, 2)
                        if header[0] != 0x68:
                            break
                        self._frame(connection, self._exact(connection, header[1]))
                except (OSError, ConnectionError):
                    pass

    @staticmethod
    def _exact(connection, size):
        data = b""
        while len(data) < size:
            chunk = connection.recv(size - len(data))
            if not chunk:
                raise ConnectionError("closed")
            data += chunk
        return data

    def _frame(self, connection, apdu):
        control = apdu[0]
        if control & 0x03 == 0x03:  # U format
            if control == 0x07:
                connection.sendall(bytes([0x68, 4, 0x0B, 0, 0, 0]))  # STARTDT con
            elif control == 0x43:
                connection.sendall(bytes([0x68, 4, 0x83, 0, 0, 0]))  # TESTFR con
            return
        if control & 0x01:  # S format
            return
        self.received += 1
        asdu = apdu[4:]
        kind = asdu[0]
        if kind == C_IC:
            self._send(connection, C_IC, ACT_CON, b"\x00\x00\x00\x14")
            for first in range(0, self.count, PER_ASDU):
                objects = b"".join((1000 + n).to_bytes(3, "little") + b"\x01"
                                   for n in range(first, min(first + PER_ASDU, self.count)))
                self._send(connection, M_SP, INROGEN, objects, len(objects) // 4)
            self._send(connection, C_IC, ACT_TERM, b"\x00\x00\x00\x14")
        elif kind == C_CI:  # acknowledged on the link (S frame), never answered
            self.counters_asked += 1
            connection.sendall(bytes([0x68, 4, 0x01, 0]) + struct.pack("<H", self.received << 1))

    def _send(self, connection, kind, cot, objects, number=1):
        asdu = bytes([kind, number, cot, 0]) + self.ca.to_bytes(2, "little") + objects
        control = struct.pack("<HH", self.sent << 1, self.received << 1)
        self.sent = (self.sent + 1) % 32768
        connection.sendall(bytes([0x68, 4 + len(asdu)]) + control + asdu)
