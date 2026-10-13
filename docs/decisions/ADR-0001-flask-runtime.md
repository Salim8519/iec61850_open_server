# ADR-0001: Flask and native workers

Status: accepted by user, 2026-09-09.

Use Flask/Waitress instead of the earlier .NET proposal. Bundle Python and static
React assets. Native libiec61850 workers implement MMS; Flask does not implement
the wire protocol. SSE and HTTP commands simplify the browser transport.
Private subprocess pipes avoid an additional listening service for worker IPC.

Maintain native worker isolation and a pinned dependency. Packaging is one-folder,
with no runtime dependency installation. A user-owned lab adapter may be configured
only through the explicit network workflow. First release prioritizes SCADA tests.
