"""Print the public template IED model (cfg and controls) used by the Explorer wire test."""
import json
import os
import sys

sys.path.insert(0, os.environ["STUDIO_BACKEND"])
from studio_app.scl.importer import import_scl  # noqa: E402
from studio_app.scl.templates import make_template  # noqa: E402

model = import_scl(make_template(sys.argv[1] if len(sys.argv) > 1 else "ExploreTest"))[0]
print(json.dumps({"cfg": model["cfg"], "controls": model["controls"]}))
