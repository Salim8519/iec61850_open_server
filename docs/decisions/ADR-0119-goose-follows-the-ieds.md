# ADR-0119: GOOSE follows the IEDs' network automatically; a real device asks once

Accepted: 2026-10-26. Owner: "I prefer GOOSE to use the same network as the IEDs automatically, just
protection for where the GOOSE goes as usual"; Npcap shown as one clear red line with a download
link. Amends [ADR-0073](ADR-0073-goose-engine-go.md) (inside by default, one confirmed switch to
share). Change [0191](../changes/2026/0191-goose-follows-the-ieds.md). Go engine and `studio_goose`.

## Problem

A PC (IED 130) and a VM (IED 1005) both ran 0.1.27 in lab mode and the VM never received the GOOSE.
GOOSE stayed inside ScadaBench by default, a second confirmed switch was needed to send it, the
Npcap note was easy to miss, and nothing said when the other machine was on a different network
(the owner's actual cause). A user who puts IEDs on a network expects their GOOSE there too.

## Decision

1. **Stored values stay Python-readable.** The choice is `auto` (default) or `local` (only
   inside). On disk `goose_reach` keeps Python's `inside`/`outside` unchanged; `local` is the Go-only
   key `goose_local: true`, stored only while chosen. The Python engine drops keys it does not know,
   so it opens every project (the fold-to-Python tests prove it), and default projects stay
   byte-identical. A project stored as `outside` (shared before) keeps sending on its picked adapter
   in every mode, and its `goose_share_guid` remains its confirmation.
2. **Auto.** GOOSE goes out on the IED network: the lab adapter in lab mode, or the adapter chosen
   under Advanced. With IEDs on this PC only (loopback) and no adapter chosen it stays inside and
   only listens, exactly as before.
3. **Protection unchanged and always on:** 3 s listen before the first frame, own reflections
   dropped, network GOOSE never resent, a real device sending our gocbRef holds that IED, rate cap.
4. **A real device asks once per adapter.** If the project has real IEDs, or `studio_goose` hears
   GOOSE from a globally administered (vendor) MAC on the adapter, sharing holds until the user
   confirms the existing high-risk warning; the confirmation is stored as `goose_share_guid`.
   `studio_goose` enforces the held state itself (`uplink.confirm_real`, `share: "real_hold"`) so no
   frame leaves before the answer; other ScadaBench instances (locally administered `02:` MACs) never
   trigger it. Capability `goose_engine_v2` lets the backend refuse an older engine.
5. **Nobody there.** While sharing, if a GOOSE the project expects is still waiting and nothing has
   been heard from the network for 10 s, the card says the other PC or VM may be on another network.
6. **Npcap.** Without it the card shows one red line, "Npcap isn't installed, so GOOSE can't leave
   this PC.", and a "Download Npcap (free)" link; GOOSE keeps working inside.
7. **Python twin.** Tests that compare with the frozen Python recordings switch these rules off
   (`netset.AsPythonTwin`, `goosedom.AsPythonTwin`); the application never does.

## Compatibility

Lab-mode projects now send their simulated GOOSE to the lab network at Run without a click; the guard
and the real-device question are the safety net. Loopback projects behave as before. The browser
sends `reach: "auto" | "local"`; old values are still accepted. Project files stay readable by the
Python engine, which keeps its own ADR-0073 behaviour. Python engine unchanged.
