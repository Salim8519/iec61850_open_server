"""The SLD refusal for a real IED names where real equipment can be operated (change 0091)."""
from studio_app.domain.network_settings import command_refusal


def test_real_ied_refusal_points_to_the_tools_that_may_command_it():
    text = command_refusal({"name": "FEED1", "role": "real"})
    assert "nothing was sent" in text and "IED Explorer or the gateway" in text
    assert "not enabled in this version" not in text
    assert command_refusal({"name": "FEED1", "role": "simulated"}) is None
