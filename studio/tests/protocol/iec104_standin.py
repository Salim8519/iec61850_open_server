"""A minimal IEC 60870-5-104 controlled station written from the standard, independent of
lib60870: STARTDT, station interrogation, counter interrogation and command confirmations.
Used to check what the ScadaBench master asks for and how it words the answers."""
import socket
import struct
import threading

C_IC, C_CI, C_DC, M_SP, M_IT = 100, 101, 46, 1, 15
ACT, ACT_CON, ACT_TERM, INROGEN, REQCOGEN = 6, 7, 10, 20, 37
NEGATIVE = 0x40


class Station104:
    """counters: answer C_CI with M_IT IOA 800 (True) or refuse it negatively (False).
    Selects are refused with a negative confirmation; direct executes are confirmed."""

    def __init__(self, port, counters=True, ca=1):
        self.port, self.counters, self.ca = port, counters, ca
        self.received, self.sent_seq, self.received_seq = [], 0, 0
        self.listener = socket.create_server(("127.0.0.1", port))
        self.listener.settimeout(10)
        self.thread = threading.Thread(target=self._serve, daemon=True)
        self.thread.start()

    def close(self):
        self.listener.close()

    def _serve(self):
        try:
            connection, _ = self.listener.accept()
        except OSError:
            return
        with connection:
            connection.settimeout(15)
            try:
                while True:
                    header = self._exact(connection, 2)
                    if header[0] != 0x68:
                        return
                    self._frame(connection, self._exact(connection, header[1]))
            except (OSError, ConnectionError):
                return

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
        control = apdu[:4]
        if control[0] & 0x03 == 0x03:  # U format
            if control[0] == 0x07:
                connection.sendall(bytes([0x68, 4, 0x0B, 0, 0, 0]))  # STARTDT con
            elif control[0] == 0x43:
                connection.sendall(bytes([0x68, 4, 0x83, 0, 0, 0]))  # TESTFR con
            return
        if control[0] & 0x01:  # S format
            return
        self.received_seq += 1
        asdu = apdu[4:]
        kind, cot, ioa = asdu[0], asdu[2] & 0x3F, int.from_bytes(asdu[6:9], "little")
        self.received.append((kind, cot, ioa))
        if kind == C_IC:
            self._send(connection, C_IC, ACT_CON, 0, b"\x14")
            self._send(connection, M_SP, INROGEN, 100, b"\x01")
            self._send(connection, C_IC, ACT_TERM, 0, b"\x14")
        elif kind == C_CI:
            if not self.counters:
                self._send(connection, C_CI, ACT_CON | NEGATIVE, 0, asdu[9:10])
                return
            self._send(connection, C_CI, ACT_CON, 0, asdu[9:10])
            self._send(connection, M_IT, REQCOGEN, 800, struct.pack("<iB", 4242, 1))
            self._send(connection, C_CI, ACT_TERM, 0, asdu[9:10])
        elif kind == C_DC:
            select = asdu[9] & 0x80
            self._send(connection, C_DC, ACT_CON | (NEGATIVE if select else 0), ioa, asdu[9:10])
            if not select:
                self._send(connection, C_DC, ACT_TERM, ioa, asdu[9:10])

    def _send(self, connection, kind, cot, ioa, element):
        asdu = bytes([kind, 1, cot, 0]) + self.ca.to_bytes(2, "little") + ioa.to_bytes(3, "little") + element
        control = struct.pack("<HH", self.sent_seq << 1, self.received_seq << 1)
        self.sent_seq += 1
        connection.sendall(bytes([0x68, 4 + len(asdu)]) + control + asdu)
