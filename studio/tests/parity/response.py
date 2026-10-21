"""Response object with the attributes tests read from Flask's test response."""
import json as _json

from werkzeug.datastructures import Headers


class Response:
    def __init__(self, status, headers, data):
        self.status_code = status
        self.headers = Headers(headers)
        self.data = data

    @property
    def status(self):
        return str(self.status_code)

    @property
    def content_type(self):
        return self.headers.get("Content-Type", "")

    @property
    def mimetype(self):
        return self.content_type.split(";", 1)[0].strip().lower()

    @property
    def is_json(self):
        kind = self.mimetype
        return kind == "application/json" or (kind.startswith("application/") and kind.endswith("+json"))

    @property
    def text(self):
        return self.data.decode("utf-8")

    def get_data(self, as_text=False):
        return self.text if as_text else self.data

    def get_json(self, force=False, silent=False):
        if not (force or self.is_json):
            return None
        try:
            return _json.loads(self.data)
        except ValueError:
            if silent:
                return None
            raise

    @property
    def json(self):
        return self.get_json()

    def __repr__(self):
        return f"<parity Response {self.status_code} {self.mimetype}>"
