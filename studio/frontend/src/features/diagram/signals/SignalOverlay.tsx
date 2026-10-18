import { useEffect, useMemo, useSyncExternalStore } from 'react';
import { ViewportPortal, useNodes, useStore } from '@xyflow/react';
import type { EquipmentNode } from '../../../api/types';
import { clientLine, externalClients, outsideNames, readerBadge } from '../../../copy/signalLayers';
import type { SignalLayerStore } from '../../../equipment/signalLayerStore';
import { edgePoint, equipmentByIed, linkCurve, placeTags, tagScale, TAG_HEIGHT, type TagSpec } from '../../../equipment/signalLayout';
import { EDGE_PULSE_MS, problemWords, PULSE_MS } from '../../../equipment/signalPulses';
import { IedTag, type TagProblem } from './IedTag';
import { PulseDot } from './PulseDot';
import './signal-layers.css';

export type SignalSelection = { kind: 'link'; id: string } | { kind: 'ied'; name: string } | null;

/** GOOSE and MMS activity drawn above the SLD (spec signal-layers). A view only: it reads the
 * shared store and the drawing's positions, and never changes or saves the diagram.
 * Only small IED name tags stay on the canvas; IEDs outside the project have no tag (change 0097). */
export function SignalOverlay({ store, iedNames, running, selection, select }: {
  store: SignalLayerStore; iedNames: Record<string, string>; running: Set<string> | null; selection: SignalSelection;
  select: (selection: SignalSelection) => void;
}) {
  const state = useSyncExternalStore(store.subscribe, store.get, store.get);
  const nodes = useNodes<EquipmentNode>();
  // Tags keep a readable size when the drawing is zoomed out (like symbol strokes, ADR-0045).
  const scale = tagScale(useStore(store => store.transform[2]));
  const { goose, readers } = state.layers;
  useEffect(() => {
    const wait = store.prune(performance.now());
    if (wait === null) return;
    const timer = setTimeout(() => store.prune(performance.now()), wait);
    return () => clearTimeout(timer);
  }, [state.pulses, state.controls, store]);
  const bound = useMemo(() => equipmentByIed(nodes), [nodes]);
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
  if (!goose && !readers) return null;
  const boxes = nodes.map(node => ({ id: node.id, x: node.position.x, y: node.position.y,
    width: node.measured?.width ?? 80, height: node.measured?.height ?? 60 }));
  const placed = placeTags(boxes, bound, specs);
  const at = new Map(placed.map(tag => [tag.name, { x: tag.x + 36 * scale, y: tag.y + (TAG_HEIGHT / 2) * scale }]));
  const links = new Map(state.map.links.map(link => [link.id, link]));
  const outsiders = outsideNames(state.map.ieds);
  const now = performance.now();
  const active = goose ? state.pulses.filter(pulse => links.has(pulse.linkId)) : [];
  const problems = (name: string): TagProblem[] => !goose || !state.map.running ? [] : state.map.links
    .filter(link => link.subscriber === name && link.status === 'declared' && link.gocb_ref)
    .map(link => problemWords(link.publisher, state.goose[link.gocb_ref!] ?? link.live ?? undefined))
    .filter((item): item is TagProblem => !!item && (item.tone === 'problem' || state.listening));
  return <ViewportPortal>
    <div className="signal-layer">
      {active.length > 0 && <svg className="signal-links" width="1" height="1" aria-hidden>
        {active.map(pulse => {
          const link = links.get(pulse.linkId)!;
          const to = at.get(link.subscriber);
          if (!to) return null;
          // A sender outside the project has no tag: its one dot enters from the canvas edge, then nothing remains.
          const outside = !at.has(link.publisher) && (link.status === 'remote' || outsiders.has(link.publisher));
          const from = outside ? edgePoint(boxes, to) : at.get(link.publisher);
          if (!from) return null;
          const curve = linkCurve(from, to);
          const middle = curve.at(0.5);
          const duration = outside ? EDGE_PULSE_MS : PULSE_MS;
          return <g key={pulse.key} className={`signal-link ${pulse.kind}${outside ? ' edge' : ''}`}
            onClick={() => select({ kind: 'link', id: link.id })}>
            {!outside && <path d={curve.path} className="signal-link-hit" />}
            {!outside && <path d={curve.path} className="signal-link-line" />}
            {now - pulse.start < duration + 100 && <PulseDot curve={curve} kind={pulse.kind} start={pulse.start} duration={duration} />}
            <text x={middle.x} y={middle.y - 8} className="signal-link-label">{pulse.label}</text>
          </g>;
        })}
      </svg>}
      {placed.map(tag => {
        const clients = readers && tag.iedId ? state.clients[tag.iedId] : undefined;
        const operated = state.controls.filter(item => item.iedId === tag.iedId).at(-1);
        return <IedTag key={tag.name} tag={tag} scale={scale} problems={problems(tag.name)}
          readers={readerBadge(clients)} readersTitle={externalClients(clients).map(clientLine).join('\n')}
          operated={operated && { key: operated.key, label: operated.label }}
          stopped={!!running && !!tag.iedId && !running.has(tag.iedId)}
          selected={selection?.kind === 'ied' && selection.name === tag.name}
          select={() => select({ kind: 'ied', name: tag.name })} />;
      })}
    </div>
  </ViewportPortal>;
}
