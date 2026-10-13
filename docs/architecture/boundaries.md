# Architecture boundaries

The new application lives in studio/. The original C/Flask/Docker demo remains intact.

| Folder | Responsibility |
| --- | --- |
| backend/studio_app/api | Request validation and service calls |
| backend/studio_app/contracts | Typed boundary validation |
| backend/studio_app/domain | Framework-independent engineering concepts |
| backend/studio_app/projects | Atomic archives, loading and autosaves |
| backend/studio_app/scl | XML import, resolution, diagnostics, export |
| backend/studio_app/simulation | Source ownership, behaviour and scheduling |
| backend/studio_app/runtime | Worker lifecycle and acknowledged updates |
| backend/studio_app/networking | Endpoint preflight, lab plan, journal and elevated helper |
| backend/studio_app/acquisition | Read-only real-IED sessions, live values and GOOSE observation |
| backend/studio_app/observability | Append-only structured run evidence and diagnostics |
| backend/studio_app/launcher | Process entrypoint, browser and shutdown |
| frontend/src/features | Substation, IED, scenario and connection workspaces |
| frontend/src/components | Shared reusable controls |
| frontend/src/equipment | SVG equipment symbols and terminal definitions |
| native/worker | Native MMS model, reports, controls, GOOSE publishing and IPC |
| native/live | Read-only MMS client and GOOSE receiver for real devices |
| native/common | Npcap loading, MMS model discovery and value conversion shared by targets |
| packaging | Frozen application entry point and one-folder build specification |
| tests | Unit, protocol, browser, packaging and performance evidence |
| tools | Architecture, file-size and documentation checks |

API -> application services -> domain/contracts. Domain imports neither API nor
infrastructure. Native protocol state is isolated per IED. Authoritative sources
live in the coordinator; UI animation cannot change the simulation.

The file-size check counts blank lines and comments. Each module must have one
reason to change. Boundaries apply even when a smaller implementation is faster.
