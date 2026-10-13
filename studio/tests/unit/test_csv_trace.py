import pytest
from studio_app.domain.source import validate_source, source_summary
from studio_app.simulation.generators import value_at

SIGNAL = {"kind": "FLOAT32", "fc": "MX"}


def spec(text="time,value\n0,10\n1,20\n2,30\n", **options):
    return {"mode": "csv", "csv_text": text, "time_column": "time", "value_column": "value", **options}


def test_hold_linear_repeat_and_terminal_value():
    hold = validate_source(SIGNAL, spec())
    assert value_at(hold, 0.5) == 10
    assert value_at(hold, 1) == 20
    assert value_at(hold, 30) == 30
    linear = validate_source(SIGNAL, spec(interpolation="linear"))
    assert value_at(linear, 0.5) == 15
    repeated = validate_source(SIGNAL, spec(repeat=True))
    assert value_at(repeated, 2.5) == 10
    assert "csv_text" not in source_summary(hold) and "points" not in source_summary(hold)


@pytest.mark.parametrize("text", ["", "time,time\n0,1", "time,value\n0,NaN", "time,value\n0,inf",
                                  "time,value\n0,10\n0,20", "time,value\n1,10", "time,value\n0,=SUM(1)",
                                  "time,value\n0,10,extra", "time,value\n0,10\n-1,5"])
def test_invalid_traces_are_rejected(text):
    with pytest.raises(ValueError):
        validate_source(SIGNAL, spec(text))


def test_integer_constraints_and_activation_clock():
    with pytest.raises(ValueError, match="floating-point"):
        validate_source({"kind": "INT32", "fc": "MX"}, spec(interpolation="linear"))
    with pytest.raises(ValueError, match="integer"):
        validate_source({"kind": "INT32", "fc": "MX"}, spec("time,value\n0,1.5"))
    source = validate_source(SIGNAL, spec())
    source["start_elapsed"] = 100
    assert value_at(source, 100.5) == 10


def test_csv_is_reparsed_instead_of_trusting_stored_points():
    source = validate_source(SIGNAL, spec(points=[[0, 999]]))
    assert value_at(source, 0) == 10
