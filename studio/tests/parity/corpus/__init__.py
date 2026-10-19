"""Golden SCL corpus (docs/architecture/go-engine/tests/05-scl-corpus.md).

Public and synthetic cases only: variants of the built-in sample IED (make_template) and of
the public files in scd/ and marketing/videos/demo/. Shared with stream B (scl-project):
`python -m parity.corpus.scl_corpus --write` regenerates fixtures/ (inputs) and fixtures/manifest.json, and
`python -m parity.corpus.run` runs every case through one or two engines and diffs outputs.
"""
