# Go module licences

Licence files of the third-party Go modules linked into `studio_server.exe`, copied unchanged from
the Go module cache at the versions pinned in `studio/engine/go.mod`. The Go release copies each one
to `_internal/licenses/go-<folder>` (`studio/tools/go_release_steps.py`). Added with
[ADR-0105](../../../../docs/decisions/ADR-0105-gateway-add-points.md).

| Folder | Module | Version | Licence |
| --- | --- | --- | --- |
| excelize | github.com/xuri/excelize/v2 | v2.9.1 | BSD-3-Clause |
| efp | github.com/xuri/efp | v0.0.1 | BSD-3-Clause |
| nfp | github.com/xuri/nfp | v0.0.1 | BSD-3-Clause |
| mscfb | github.com/richardlehane/mscfb | v1.0.4 | Apache-2.0 |
| msoleps | github.com/richardlehane/msoleps | v1.0.4 | Apache-2.0 |
| go-deepcopy | github.com/tiendc/go-deepcopy | v1.6.0 | MIT |
| x-crypto | golang.org/x/crypto | v0.38.0 | BSD-3-Clause |
| x-net | golang.org/x/net | v0.40.0 | BSD-3-Clause |
| x-text | golang.org/x/text | v0.25.0 | BSD-3-Clause |
| x-sys | golang.org/x/sys | v0.41.0 | BSD-3-Clause (in use since ADR-0099) |
