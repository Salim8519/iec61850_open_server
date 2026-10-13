import pytest
from studio_app.domain.sample import make_sample
from studio_app.domain.signal import validate_value


def test_nested_measurement_uses_parent_quality_and_timestamp():
    ref = "IEDLD/GGIO1.AnIn1"
    measurement = {"ref": ref + ".mag.f", "kind": "FLOAT32", "fc": "MX"}
    quality = {"ref": ref + ".q", "kind": "Quality", "fc": "MX", "value": 0}
    timestamp = {"ref": ref + ".t", "kind": "Timestamp", "fc": "MX", "value": 0}
    index = {item["ref"]: item for item in (measurement, quality, timestamp)}
    updates, related = make_sample(measurement, index, 123.5, 1780000000000, 1)
    assert related == {"q": quality, "t": timestamp}
    assert updates == [{"ref": measurement["ref"], "value": 123.5},
                       {"ref": quality["ref"], "value": 1},
                       {"ref": timestamp["ref"], "value": 1780000000000}]


def test_negative_timestamp_is_rejected():
    with pytest.raises(ValueError):
        validate_value({"kind": "Timestamp"}, -1)
