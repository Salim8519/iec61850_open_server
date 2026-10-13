# Change 0009: offline SCL XSD checks

Date: 2026-09-09.

Bundled unmodified SCL2003, SCL2007B and SCL2007B4 schemas and retained their
copyright, history, manifests and EULA. Dependency/provenance is documented in
[ADR-0003](../../decisions/ADR-0003-scl-validation.md).

The importer records full-source schema status and up to 100 line-numbered
diagnostics. The IED workspace exposes these separately from semantic/runtime
support. Schema-invalid vendor documents can still be imported with warnings;
unknown schema editions are explicitly unsupported, not silently accepted.
A read-only /scl/validate endpoint supports independent document checks.

Validation found our laboratory template omitted the required Server Authentication
element. The template now declares the supported 2007B4 schema and includes that
element; generated templates pass XSD validation. All three bundled schema sets
compile offline. Entity/network loading is disabled and unsafe XML is rejected.

Verification: 51 Python tests pass, including native CSV playback and schema
checks. Frontend production build and architecture/length/documentation checks pass.
This does not complete edited SCL/substation export, full array support, or
interoperability certification. Those remain separate implementation/acceptance work.

Read-only customer audit: 440KV_Y210_BCU-1.CID and 210_ORGINALCID.CID both pass
the selected XSD with zero diagnostics and retain successful semantic import.
Original files were not modified or copied into maintained test fixtures.

The frozen launcher adds --self-test to exercise its bundled schema, generated
template, native engine and independent MMS client using temporary data. It
does not claim clean-machine verification; it provides a repeatable check for it.

Final verification for this milestone: 52 Python tests pass, including the frozen
self-test and temporary-data cleanup. The portable folder was rebuilt successfully.
