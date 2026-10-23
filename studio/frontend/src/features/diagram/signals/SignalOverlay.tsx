import { useEffect, useMemo, useSyncExternalStore } from 'react';
import { ViewportPortal, useNodes, useStore } from '@xyflow/react';
import type { EquipmentNode } from '../../../api/types';
import { GOT_IT } from '../../../copy/signalLayers';
import { triggeredOrigins } from '../../../equipment/gooseTriggers';
import { changeGroups } from '../../../equipment/signalChanges';
import type { SignalLayerStore } from '../../../equipment/signalLayerStore';
import { componentBoxes, tagScale } from '../../../equipment/signalLayout';
import { animatedLines, planChanges, type PlannedChange } from '../../../equipment/signalPlan';
import { ChangeChip } from './ChangeChip';
import { ChangeLines } from './ChangeLines';
import { FlyLines } from './FlyLines';
import { IedFlash } from './IedFlash';
import { MissingChip } from './MissingChip';
import { useMeasuredBoxes } from './useMeasuredBoxes';
import './signal-layers.css';

export type SignalSelection = { kind: 'link'; id: string } | { kind: 'ied'; name: string } | null;
type Flash = { key: number; label: string; start: number };

/** GOOSE and MMS activity drawn above the SLD (spec signal-layers, ADR-0112). A view only: it reads
 * the shared store and the drawing's positions, and never changes or saves the diagram. IEDs are
 * the IED components the owner placed: one real change is one chip at the component that triggered
 * it and one line per receiving IED component, through the sender's component. No marks or tags. */
export function SignalOverlay({ store, iedNames, selection: _selection, select }: {
  store: SignalLayerStore; iedNames: Record<string, string>; running: Set<string> | null; selection: SignalSelection;
  select: (selection: SignalSelection) => void;
}) {
  const state = useSyncExternalStore(store.subscribe, store.get, store.get);
  const nodes = useNodes<EquipmentNode>();
  // Chips, dots and lines keep a readable size when the drawing is zoomed out (like symbol strokes, ADR-0045).
  const zoom = useStore(flow => flow.transform[2]);
  const scale = tagScale(zoom);
  const { goose, readers } = state.layers;
  useEffect(() => {
    const wait = store.prune(performance.now());
    if (wait === null) return;
    const timer = setTimeout(() => store.prune(performance.now()), wait);
    return () => clearTimeout(timer);
  }, [state.pulses, state.controls, store]);
  const boxes = useMeasuredBoxes(nodes);
  const components = useMemo(() => componentBoxes(nodes, boxes, iedNames), [nodes, boxes, iedNames]);
  const originOf = useMemo(() => triggeredOrigins(nodes, state.triggers), [nodes, state.triggers]);
  const links = useMemo(() => new Map(state.map.links.map(link => [link.id, link])), [state.map.links]);
  const active = useMemo(() => goose ? state.pulses.filter(pulse => links.has(pulse.linkId)) : [], [goose, state.pulses, links]);
  const plan = useMemo(() => planChanges(changeGroups(active, state.signals, links, originOf), boxes, components, scale),
    [active, state.signals, links, originOf, boxes, components, scale]);
  const animated = useMemo(() => animatedLines(plan), [plan]);
  if (!goose && !readers) return null;
  const now = performance.now();
  const flashes = receiverFlashes(plan);
  const publisherOf = (item: PlannedChange) => () => select({ kind: 'ied', name: item.group.publisher });
  return <ViewportPortal>
    <div className="signal-layer">
      {plan.length > 0 && <svg className="signal-links" width="1" height="1" aria-hidden
        style={{ ['--signal-scale' as string]: scale }}>
        <ChangeLines plan={plan} animated={animated} equipment={state.equipment} scale={scale} select={select} />
        <FlyLines plan={plan} now={now} scale={scale} select={select} />
      </svg>}
      {plan.map(item => item.chip && <ChangeChip key={item.group.id} item={item} equipment={state.equipment} scale={scale}
        select={publisherOf(item)} />)}
      {plan.map(item => <MissingChip key={`m${item.group.id}`} item={item} scale={scale} select={publisherOf(item)} />)}
      {[...components.byId].map(([iedId, box]) => {
        const name = iedNames[iedId] ?? iedId;
        const control = readers ? state.controls.filter(item => item.iedId === iedId).at(-1) : undefined;
        return <IedFlash key={iedId} box={box} scale={scale} received={flashes.get(name)}
          operated={control && { key: control.key, label: control.label }} select={() => select({ kind: 'ied', name })} />;
      })}
    </div>
  </ViewportPortal>;
}

/** The newest short line per receiving component: "✓ got it" only after a real stNum increase from
 * a sender on the drawing (never for an expected change); a flying dot names its sender. */
function receiverFlashes(plan: PlannedChange[]): Map<string, Flash> {
  const result = new Map<string, Flash>();
  const keep = (name: string, flash: Flash) => {
    const known = result.get(name);
    if (!known || known.start <= flash.start) result.set(name, flash);
  };
  for (const change of plan) {
    for (const line of change.lines) if (line.kind === 'observed') keep(line.subscriber, { key: line.key, label: GOT_IT, start: line.start });
    for (const line of change.fly) keep(line.subscriber, { key: line.key, label: line.label, start: line.start });
  }
  return result;
}
