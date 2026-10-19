"""Idle external MMS clients: real associations that then stay silent (S5-style, ADR-0079).

The association bytes are recorded once from studio_probe (the independent libiec61850 client)
through a loopback relay: the first two TPKT frames it sends are COTP CR and the ISO session
CONNECT carrying ACSE AARQ + MMS initiate-Request. Replaying them opens an association exactly
like a SCADA client that connects and then waits. Loopback only (127.67.x.y, ports >= 20000).
"""
import socket
import subprocess
import threading

RELAY = ("127.67.0.250", 20990)
SESSION_ACCEPT = 0x0E


def _frames(data, count):
    frames = []
    while len(frames) < count and len(data) >= 4:
        size = int.from_bytes(data[2:4], "big")
        frames.append(data[:size])
        data = data[size:]
    return frames


def _read_tpkt(sock):
    header = _exact(sock, 4)
    return header + _exact(sock, int.from_bytes(header[2:4], "big") - 4)


def _exact(sock, size):
    data = b""
    while len(data) < size:
        chunk = sock.recv(size - len(data))
        if not chunk:
            raise ConnectionError("server closed the connection during association")
        data += chunk
    return data


def capture_association(probe_exe, ip, port, reference):
    """Run one probe read through a recording relay; return the client's first two TPKT frames."""
    listener = socket.socket()
    listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    listener.bind(RELAY)
    listener.listen(1)
    recorded = bytearray()

    def relay():
        client, _ = listener.accept()
        server = socket.create_connection((ip, port), timeout=10)

        def pump(source, target, record):
            try:
                while True:
                    chunk = source.recv(65536)
                    if not chunk:
                        break
                    if record:
                        recorded.extend(chunk)
                    target.sendall(chunk)
            except OSError:
                pass
            finally:
                for sock in (source, target):
                    try:
                        sock.shutdown(socket.SHUT_RDWR)
                    except OSError:
                        pass
        back = threading.Thread(target=pump, args=(server, client, False), daemon=True)
        back.start()
        pump(client, server, True)
        back.join(5)
        client.close()
        server.close()
    worker = threading.Thread(target=relay, daemon=True)
    worker.start()
    try:
        completed = subprocess.run([str(probe_exe), RELAY[0], str(RELAY[1]), "read", reference, "ST"],
                                   capture_output=True, text=True, timeout=30)
        worker.join(10)
    finally:
        listener.close()
    frames = _frames(bytes(recorded), 2)
    if completed.returncode != 0 or len(frames) != 2:
        raise RuntimeError(f"association capture failed: {completed.stdout} {completed.stderr}")
    return frames


class IdleClients:
    """Open `count` associations with the recorded frames and keep them silent until close()."""

    def __init__(self, frames, ip, port, count):
        self.sockets, self.accepted, self.closed_by_server = [], 0, 0
        for _ in range(count):
            sock = socket.create_connection((ip, port), timeout=10)
            sock.sendall(frames[0])
            _read_tpkt(sock)  # COTP CC
            sock.sendall(frames[1])
            answer = _read_tpkt(sock)
            self.accepted += len(answer) > 7 and answer[7] == SESSION_ACCEPT  # TPKT 4 + COTP DT 3
            sock.settimeout(None)
            self.sockets.append(sock)
            threading.Thread(target=self._drain, args=(sock,), daemon=True).start()

    def _drain(self, sock):
        try:
            while sock.recv(65536):
                pass
            self.closed_by_server += 1
        except OSError:
            pass

    def close(self):
        for sock in self.sockets:
            try:
                sock.close()
            except OSError:
                pass
        self.sockets = []
