"""A headless Chrome (or Edge) driven over the DevTools protocol, for the fix-plan UI checks.

Standard library only: a minimal WebSocket client (text frames, client masking) is enough for
CDP. The browser gets its own temporary profile and a debugging port in 23900-23999.
"""
import base64
import http.client
import json
import os
from pathlib import Path
import shutil
import socket
import struct
import subprocess
import tempfile
import time
from urllib.parse import urlparse

BROWSERS = [r"C:\Program Files\Google\Chrome\Application\chrome.exe",
            r"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"]


class Socket:
    def __init__(self, url):
        parts = urlparse(url)
        self.sock = socket.create_connection((parts.hostname, parts.port), timeout=60)
        key = base64.b64encode(os.urandom(16)).decode()
        self.sock.sendall((f"GET {parts.path} HTTP/1.1\r\nHost: {parts.hostname}:{parts.port}\r\n"
                           f"Upgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: {key}\r\n"
                           "Sec-WebSocket-Version: 13\r\n\r\n").encode())
        reply = b""
        while b"\r\n\r\n" not in reply:
            reply += self.sock.recv(4096)
        head, self.rest = reply.split(b"\r\n\r\n", 1)
        if b" 101 " not in head.split(b"\r\n")[0]:
            raise RuntimeError(f"WebSocket refused: {head[:200]!r}")

    def send(self, text):
        data, mask = text.encode(), os.urandom(4)
        header = bytes([0x81])
        if len(data) < 126:
            header += bytes([0x80 | len(data)])
        elif len(data) < 65536:
            header += bytes([0x80 | 126]) + struct.pack(">H", len(data))
        else:
            header += bytes([0x80 | 127]) + struct.pack(">Q", len(data))
        self.sock.sendall(header + mask + bytes(byte ^ mask[index % 4] for index, byte in enumerate(data)))

    def _read(self, count):
        while len(self.rest) < count:
            chunk = self.sock.recv(65536)
            if not chunk:
                raise ConnectionError("browser closed the connection")
            self.rest += chunk
        data, self.rest = self.rest[:count], self.rest[count:]
        return data

    def receive(self):
        message = b""
        while True:
            first, second = self._read(2)
            length = second & 0x7F
            if length == 126:
                length = struct.unpack(">H", self._read(2))[0]
            elif length == 127:
                length = struct.unpack(">Q", self._read(8))[0]
            payload = self._read(length)
            if first & 0x0F == 0x8:
                raise ConnectionError("browser closed the WebSocket")
            if first & 0x0F in (0x0, 0x1, 0x2):
                message += payload
                if first & 0x80:
                    return message.decode("utf-8", "replace")


class Browser:
    def __init__(self, port=23901):
        executable = next((path for path in BROWSERS if Path(path).is_file()), None)
        if not executable:
            raise RuntimeError("no Chrome or Edge found")
        self.port, self.profile = port, tempfile.mkdtemp(prefix="fixplan-browser-")
        self.process = subprocess.Popen([executable, "--headless=new", f"--remote-debugging-port={port}",
                                         f"--user-data-dir={self.profile}", "--no-first-run",
                                         "--no-default-browser-check", "--window-size=1366,768", "about:blank"],
                                        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        self.next_id, self.events = 0, []
        deadline = time.time() + 30
        while True:
            try:
                pages = self._json("/json/list")
                page = next(item for item in pages if item.get("type") == "page")
                break
            except (OSError, StopIteration, ValueError):
                if time.time() > deadline:
                    raise
                time.sleep(0.2)
        self.ws = Socket(page["webSocketDebuggerUrl"])

    def _json(self, path):
        connection = http.client.HTTPConnection("127.0.0.1", self.port, timeout=5)
        connection.request("GET", path)
        return json.loads(connection.getresponse().read())

    def call(self, method, **params):
        self.next_id += 1
        ident = self.next_id
        self.ws.send(json.dumps({"id": ident, "method": method, "params": params}))
        while True:
            message = json.loads(self.ws.receive())
            if message.get("id") == ident:
                if "error" in message:
                    raise RuntimeError(f"{method}: {message['error']}")
                return message.get("result", {})
            self.events.append(message)

    def pump(self, seconds=0.0):
        """Read events for a while (requests paused by Fetch arrive here)."""
        self.ws.sock.settimeout(max(seconds, 0.01))
        try:
            while True:
                self.events.append(json.loads(self.ws.receive()))
        except (socket.timeout, TimeoutError):
            pass
        finally:
            self.ws.sock.settimeout(60)

    def evaluate(self, expression):
        result = self.call("Runtime.evaluate", expression=expression, awaitPromise=True, returnByValue=True)
        if "exceptionDetails" in result:
            raise RuntimeError(f"page error: {result['exceptionDetails']}")
        return result.get("result", {}).get("value")

    def wait(self, expression, timeout=20):
        deadline = time.time() + timeout
        while time.time() < deadline:
            value = self.evaluate(expression)
            if value:
                return value
            time.sleep(0.1)
        return None

    def mouse(self, kind, x, y, buttons=1):
        self.call("Input.dispatchMouseEvent", type=kind, x=x, y=y, button="left", buttons=buttons,
                  clickCount=1 if kind != "mouseMoved" else 0)

    def drag(self, start, end, steps=8):
        self.mouse("mousePressed", *start)
        for step in range(1, steps + 1):
            self.mouse("mouseMoved", start[0] + (end[0] - start[0]) * step / steps,
                       start[1] + (end[1] - start[1]) * step / steps)
        self.mouse("mouseReleased", *end, buttons=0)

    def ctrl_s(self):
        for kind in ("rawKeyDown", "keyUp"):
            self.call("Input.dispatchKeyEvent", type=kind, modifiers=2, key="s", code="KeyS",
                      windowsVirtualKeyCode=83)

    def close(self):
        try:
            self.call("Browser.close")
        except (OSError, RuntimeError, ConnectionError):
            pass
        try:
            self.process.wait(10)
        except subprocess.TimeoutExpired:
            self.process.kill()
        shutil.rmtree(self.profile, ignore_errors=True)
