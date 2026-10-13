import pytest
from studio_app.scl.templates import make_template
from studio_app.scl.schema_validation import validate_schema, load_schema


def test_generated_template_passes_2007b4_schema():
    result = validate_schema(make_template())
    assert result == {"status": "valid", "schema": "SCL2007B4", "errors": []}


@pytest.mark.parametrize("version", ["SCL2003", "SCL2007B", "SCL2007B4"])
def test_bundled_schemas_compile_offline(version):
    assert load_schema(version) is not None


def test_invalid_schema_and_unavailable_edition_are_not_called_valid():
    source = make_template().replace(b'<Authentication none="true" />', b'')
    result = validate_schema(source)
    assert result["status"] == "invalid" and result["errors"][0]["line"] > 0
    unsupported = make_template().replace(b'revision="B"', b'revision="C"')
    assert validate_schema(unsupported)["status"] == "unsupported"
