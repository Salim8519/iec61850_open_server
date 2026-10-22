import { useEffect, useMemo, useSyncExternalStore } from 'react';
import { ViewportPortal, useNodes, useStore } from '@xyflow/react';
import type { EquipmentNode } from '../../../api/types';
import { clientLine, externalClients, GOT_IT, outsideNames, readerBadge } from '../../../copy/signalLayers';
import { changeGroups } from '../../../equipment/signalChanges';
import type { Pulse, SignalLayerStore } from '../../../equipment/signalLayerStore';
import { edgePoint, equipmentByIed, linkCurve, markScale, placeTags, tagCentre, tagScale, type TagSpec } from '../../../equipment/signalLayout';
import { originIndex } from '../../../equipment/signalOrigin';
import { animatedLines, planChanges } from '../../../equipment/signalPlan';
import { EDGE_PULSE_MS, problemWords } from '../../../equipment/signalPulses';
import { ChangeChip } from './ChangeChip';
import { ChangeLines } from './ChangeLines';
import { IedTag, type TagProblem } from './IedTag';
import { PulseDot } from './PulseDot';
import './signal-layers.css';

export type SignalSelection = { kind: 'link'; id: string } | { kind: 'ied'; name: string } | null;

/** GOOSE and MMS activity drawn above the SLD (spec signal-layers). A view only: it reads the
 * shared store and the drawing's positions, and never changes or saves the diagram.
 * Only small IED corner marks (or name tags for unbound IEDs) stay on the canvas; IEDs outside the
 * project have no tag (changes 0097, 0105). One real change is one chip at its symbol and one smart
 * line per receiving IED (ADR-0103); a sender outside the project keeps its one dot beside the receiver. */
export function SignalOverlay({ store, iedNames, running, selection, select }: {
  store: SignalLayerStore; iedNames: Record<string, string>; running: Set<string> | null; selection: SignalSelection;
  select: (selection: SignalSelection) => void;
}) {
  const state = useSyncExternalStore(store.subscribe, store.get, store.get);
  const nodes = useNodes<EquipmentNode>();
  // Tags keep a readable size when the drawing is zoomed out (like symbol strokes, ADR-0045).
  const zoom = useStore(store => store.transform[2]);
  const scale = tagScale(zoom);
  // Corner marks grow less, so they stay inside their equipment (change 0105).
  const mark = markScale(zoom);
  const { goose, readers } = state.layers;
  useEffect(() => {
    const wait = store.prune(performance.now());
    if (wait === null) return;
    const timer = setTimeout(() => store.prune(performance.now()), wait);
    return () => clearTimeout(timer);
  }, [state.pulses, state.controls, store]);
  const bound = useMemo(() => equipmentByIed(nodes), [nodes]);
  const originOf = useMemo(() => originIndex(nodes), [nodes]);
  const boxes = useMemo(() => nodes.map(node => ({ id: node.id, x: node.position.x, y: node.position.y,
    width: node.measured?.width ?? 80, height: node.measured?.height ?? 60 })), [nodes]);
  const specs = useMemo(() => {
    const result = new Map<string, TagSpec>();
    if (goose) for (const ied of state.map.ieds) {
      result.set(ied.name, { name: ied.name, iedId: ied.ied_id, outside: !ied.in_project, remote: ied.remote });
    }
    if (readers) for (const [id, clients] of Object.entries(state.clients)) {
      const name = iedNames[id];
      if (name && externalClients(clients).length && !result.has(name)) result.set(name, { name, iedId: id, outside: false, remote: false });
    }
    return [...result.values()];
  }, [goose, readers, state.map.ieds, state.clients, iedNames]);
  const placed = useMemo(() => placeTags(boxes, bound, specs), [boxes, bound, specs]);
  const at = useMemo(() => new Map(placed.map(tag => [tag.name, tagCentre(tag, scale, mark)])), [placed, scale, mark]);
  const links = useMemo(() => new Map(state.map.links.map(link => [link.id, link])), [state.map.links]);
  const active = useMemo(() => goose ? state.pulses.filter(pulse => links.has(pulse.linkId)) : [], [goose, state.pulses, links]);
  // Senders with a place on the drawing get smart lines; one without keeps today's dot beside the receiver.
  const plan = useMemo(() => planChanges(changeGroups(active.filter(pulse => at.has(links.get(pulse.linkId)!.publisher)),
    state.signals, links, originOf), boxes, placed, at, scale), [active, state.signals, links, originOf, boxes, placed, at, scale]);
  const animated = useMemo(() => animatedLines(plan), [plan]);
  if (!goose && !readers) return null;
  const outsiders = outsideNames(state.map.ieds);
  const now = performance.now();
  const edges = active.filter(pulse => !at.has(links.get(pulse.linkId)!.publisher));
  const problems = (name: string): TagProblem[] => !goose || !state.map.running ? [] : state.map.links
    .filter(link => link.subscriber === name && link.status === 'declared' && link.gocb_ref)
    .map(link => problemWords(link.publisher, state.goose[link.gocb_ref!] ?? link.live ?? undefined))
    .filter((item): item is TagProblem => !!item && (item.tone === 'problem' || state.listening));
  return <ViewportPortal>
    <div className="signal-layer">
      {active.length > 0 && <svg className="signal-links" width="1" height="1" aria-hidden
        style={{ ['--signal-scale' as string]: scale }}>
        <ChangeLines plan={plan} animated={animated} equipment={state.equipment} scale={scale} select={select} />
        {edges.map(pulse => {
          const link = links.get(pulse.linkId)!;
          const to = at.get(link.subscriber);
          // A sender outside the project has no tag: its one dot enters near the receiver, then nothing remains.
          if (!to || !(link.status === 'remote' || outsiders.has(link.publisher))) return null;
          const curve = linkCurve(edgePoint(to, scale), to);
          // Dot and line keep their screen size when zoomed out, like the tags (change 0102).
          return <g key={pulse.key} className={`signal-link ${pulse.kind} edge`}
            onClick={() => select({ kind: 'link', id: link.id })}>
            <title>{pulse.label}</title>
            <path d={curve.path} className="signal-link-hit" />
            <path d={curve.path} className="signal-link-line" />
            {now - pulse.start < EDGE_PULSE_MS + 100 && <PulseDot curve={curve} kind={pulse.kind} start={pulse.start}
              duration={EDGE_PULSE_MS} radius={5 * scale} />}
          </g>;
        })}
      </svg>}
      {plan.map(item => <ChangeChip key={item.group.id} item={item} equipment={state.equipment} scale={scale}
        select={() => select({ kind: 'ied', name: item.group.publisher })} />)}
      {placed.map(tag => {
        const clients = readers && tag.iedId ? state.clients[tag.iedId] : undefined;
        const operated = state.controls.filter(item => item.iedId === tag.iedId).at(-1);
        const received = receivedLine(active.filter(pulse => links.get(pulse.linkId)!.subscriber === tag.name).at(-1),
          pulse => at.has(links.get(pulse.linkId)!.publisher));
        return <IedTag key={tag.name} tag={tag} scale={tag.corner ? mark : scale} problems={problems(tag.name)}
          readers={readerBadge(clients)} readersTitle={externalClients(clients).map(clientLine).join('\n')}
          operated={operated && { key: operated.key, label: operated.label }}
          received={received && { key: received.key, label: received.label }}
          stopped={!!running && !!tag.iedId && !running.has(tag.iedId)}
          selected={selection?.kind === 'ied' && selection.name === tag.name}
          select={() => select({ kind: 'ied', name: tag.name })} />;
      })}
    </div>
  </ViewportPortal>;
}

/** The receiver's own short line: "← sender · change" from a sender with no tag; for senders on the
 * drawing only "✓ got it" after a real stNum increase, nothing when expected (never "received"). */
function receivedLine(pulse: Pulse | undefined, onDrawing: (pulse: Pulse) => boolean) {
  if (!pulse) return undefined;
  if (!onDrawing(pulse)) return { key: pulse.key, label: pulse.short };
  return pulse.kind === 'observed' ? { key: pulse.key, label: GOT_IT } : undefined;
}
