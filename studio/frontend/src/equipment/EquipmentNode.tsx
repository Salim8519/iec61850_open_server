import { memo, useCallback, useContext, useEffect, useRef, useState,
  useSyncExternalStore } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import type { EquipmentNode as NodeType } from '../api/types';
import { EquipmentSymbol } from './symbols/EquipmentSymbol';
import { switchPosition, tripRelayState } from './position';
import { RelayPanel } from './RelayPanel';
import { JunctionBody } from './JunctionBody';
import { kindInfo, stateTone } from './kindInfo';
import { colorScheme, paintFor, stateFill } from './stateColors';
import './equipment-node.css';
import { IedNames, LiveValues, StoppedIeds, useModeBadge } from './liveContext';
import { stateBindings } from './stateBindings';
import { RUN_SHAPE_COPY } from '../copy/runShape';
import { modeStatus } from '../copy/testMode';
import { confirmedChange } from './readability';
import { markPaint } from '../app/perf';

/** One terminal = a target and a source handle on the same spot, so a wire may start and end on any
 *  side (interaction contract). With target-only north/west, a north-to-north wire was saved but React
 *  Flow could not draw it: an edge's source end must be a source handle. Existing wires keep their ids. */
function Terminal({ id, position }: { id: string; position: Position }) {
  return <><Handle id={id} type="target" position={position} /><Handle id={id} type="source" position={position} /></>;
}

function EquipmentNodeView({ id, data, selected }: NodeProps<NodeType>) {
  const store = useContext(LiveValues);
  const iedNames = useContext(IedNames);
  const badge = useModeBadge(id);
  const stoppedIeds = useContext(StoppedIeds);
  const subscribe = useCallback((listener: () => void) => store.subscribe(id, listener), [id, store]);
  const snapshot = useCallback(() => store.get(id), [id, store]);
  const value = useSyncExternalStore(subscribe, snapshot, snapshot);
  const linked = !!data.state_mapping;
  const switching = ['CB', 'DS', 'ES'].includes(data.kind);
  const position = switchPosition(data, value);
  const operatedValue = data.trip_state_polarity !== 'operated_false';
  const relayState = tripRelayState(linked, value, operatedValue);
  const pushStatus = data.kind === 'PUSH_BUTTON' && data.push_mode !== 'spc_momentary';
  const pushLatching = data.kind === 'PUSH_BUTTON' && data.push_mode === 'latching_status';
  const previous = useRef(position);
  const animate = ['Open', 'Closed', 'Intermediate'].includes(previous.current);
  // Count confirmed position changes; alternating the attribute restarts the CSS pulse.
  const [changes, setChanges] = useState(0);
  useEffect(() => {
    if (switching && confirmedChange(previous.current, position)) setChanges(count => count + 1);
    if (previous.current !== position) markPaint(id);
    previous.current = position;
  }, [id, position, switching]);
  const junction = data.kind === 'JUNCTION';
  const relay = data.kind === 'RELAY';
  const terminalLess = data.kind === 'PUSH_BUTTON';
  const category = kindInfo(data.kind).category;
  const infeed = data.kind === 'INFEED';
  const tone = infeed ? (data.infeed_state === 'dead' ? 'unmapped' : 'live') :
    data.kind === 'PUSH_BUTTON' && !pushStatus ? (data.control_binding ? 'live' : 'unmapped') :
    stateTone(linked, value);
  // Its value comes from an IED that is stopped while others run: say so instead of "Awaiting value".
  const sources = [...new Set(stateBindings(data.state_mapping).map(item => item.ied_id))];
  const idle = sources.length > 0 && sources.every(iedId => stoppedIeds.has(iedId));
  if (relay) return <div className={`equipment-node relay ${selected ? 'selected' : ''}`}>
    <RelayPanel nodeId={id} data={data} iedNames={iedNames} /></div>;
  const scheme = colorScheme(data);
  const fill = stateFill(data, position, value, linked);
  const fillColor = scheme && (fill === 'active' ? scheme.active : fill === 'inactive' ? scheme.inactive : null);
  const paint = fillColor ? paintFor(fillColor) : undefined;
  return <div className={`equipment-node cat-${category} ${junction ? 'draw' : ''} ${selected ? 'selected' : ''} ${fill === 'intermediate' || fill === 'invalid' ? `fill-${fill}` : ''}${idle ? ' ied-stopped' : ''}`}
    title={idle ? RUN_SHAPE_COPY.equipmentStoppedHint(sources.map(iedId => iedNames[iedId] || iedId).join(', ')) : undefined}
    data-pulse={changes ? changes % 2 ? 'a' : 'b' : undefined}>
    {!terminalLess && <Terminal id="north" position={Position.Top} />}
    {!terminalLess && <Terminal id="west" position={Position.Left} />}
    {junction ? <div className="equipment-symbol" title="Drag to move · drag a terminal to wire"><JunctionBody nodeId={id} /></div> :
    <div className="equipment-symbol" style={{ transform: `rotate(${data.rotation || 0}deg)` }}>
      <EquipmentSymbol kind={data.kind} position={position} animate={animate} paint={paint} infeed={data.infeed_type}
        operated={data.kind === 'TRIP_RELAY' ?
          value === undefined || value === null ? undefined : value === operatedValue :
          pushStatus && typeof value === 'boolean' ? value : undefined} />
    </div>}
    {!junction && <div className="equipment-label">{data.label}</div>}
    {!junction && badge && <div className="equipment-mode" title={modeStatus(badge).hint}>{modeStatus(badge).label}</div>}
    {!junction && idle && <div className="equipment-state tone-stopped">{RUN_SHAPE_COPY.equipmentStopped}</div>}
    {!junction && !idle && <div className={`equipment-state tone-${tone}`}>
      {paint && <span className="state-dot" aria-hidden="true"
        style={{ background: paint.fill, borderColor: paint.outline }} />}
      {infeed ? data.infeed_state === 'dead' ? 'Set dead' : 'Set live' : data.kind === 'TRIP_RELAY' ? relayState : data.kind === 'PUSH_BUTTON' ?
        pushStatus ? !linked ? 'Not mapped' : typeof value === 'boolean' ?
          value ? pushLatching ? 'Latched' : 'Pressed' : 'Released' : 'Awaiting state' : data.control_binding ? 'Ready' : 'Not mapped' :
        linked ? (switching ? position : (value === undefined ? 'Awaiting value' : String(value))) : 'Not mapped'}
    </div>}
    {!terminalLess && <Terminal id="east" position={Position.Right} />}
    {!terminalLess && <Terminal id="south" position={Position.Bottom} />}
  </div>;
}

export const EquipmentNode = memo(EquipmentNodeView);
