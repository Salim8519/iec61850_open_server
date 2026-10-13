# ADR-0003: offline SCL schema validation

Date: 2026-09-09. Implements the existing validated-SCL requirement.

Use pinned lxml 6.0.2 for offline XSD validation, retaining defusedxml checks before
validation. External entities, DTD loading and network resolution are disabled.
Schema selection matches the document's edition/version/revision/release; unsupported
versions are reported as unsupported, never silently validated against another edition.

Unmodified IEC code components are bundled under studio/schemas, obtained from
[CoMPAS SCL XSD](https://github.com/com-pas/compas-scl-xsd), commit
f41cc23ed058e4a89c752488c705ac5a8d8918af. The direct IEC package URL returned an empty
file during retrieval; provenance is the CoMPAS repository, not a claimed direct download.

Components: SCL2003 (IEC 61850-6 edition 1), SCL2007B (IEC 61850-6:2009),
SCL2007B4 (IEC 61850-6:2009/AMD1:2018). Original copyright notices, manifests,
history and Code Components EULA are retained. Do not edit these schema files or
imply IEC endorsement. The bundled license governs redistribution.

SCL2007B4/SCL.xsd SHA-256:
66d662f97aba46248f78a273af280c7764796abfefafb65f0ad15544d55c2b00.

Schema-invalid vendor documents may still be semantically imported with explicit
diagnostics; preserving engineering input is distinct from certifying its validity.
Generated/edited exports must pass both XSD and semantic re-import before being
called verified. XSD validity alone does not prove IEC behavior or interoperability.
