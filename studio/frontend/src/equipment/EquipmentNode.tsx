import { memo, useCallback, useContext, useEffect, useRef,
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
import { IedNames, LiveValues } from './liveContext';



function EquipmentNodeView({ id, data, selected }: NodeProps<NodeType>) {
  const store = useContext(LiveValues);
  const iedNames = useContext(IedNames);
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
  useEffect(() => { previous.current = position; }, [position]);
  const junction = data.kind === 'JUNCTION';
  const relay = data.kind === 'RELAY';
  const terminalLess = data.kind === 'PUSH_BUTTON';
  const category = kindInfo(data.kind).category;
  const tone = data.kind === 'PUSH_BUTTON' && !pushStatus ? (data.control_binding ? 'live' : 'unmapped') :
    stateTone(linked, value);
  if (relay) return <div className={`equipment-node relay ${selected ? 'selected' : ''}`}>
    <RelayPanel nodeId={id} data={data} iedNames={iedNames} /></div>;
  const scheme = colorScheme(data);
  const fill = stateFill(data, position, value, linked);
  const fillColor = scheme && (fill === 'active' ? scheme.active : fill === 'inactive' ? scheme.inactive : null);
  const paint = fillColor ? paintFor(fillColor) : undefined;
  return <div className={`equipment-node cat-${category} ${junction ? 'draw' : ''} ${selected ? 'selected' : ''} ${fill === 'intermediate' || fill === 'invalid' ? `fill-${fill}` : ''}`}>
    {!terminalLess && <Handle id="north" type="target" position={Position.Top} />}
    {!terminalLess && <Handle id="west" type="target" position={Position.Left} />}
    {junction ? <div className="equipment-symbol" title="Drag to move · drag a terminal to wire"><JunctionBody nodeId={id} /></div> :
    <div className="equipment-symbol" style={{ transform: `rotate(${data.rotation || 0}deg)` }}>
      <EquipmentSymbol kind={data.kind} position={position} animate={animate} paint={paint}
        operated={data.kind === 'TRIP_RELAY' ?
          value === undefined || value === null ? undefined : value === operatedValue :
          pushStatus && typeof value === 'boolean' ? value : undefined} />
    </div>}
    {!junction && <div className="equipment-label">{data.label}</div>}
    {!junction && <div className={`equipment-state tone-${tone}`}>
      {paint && <span className="state-dot" aria-hidden="true"
        style={{ background: paint.fill, borderColor: paint.outline }} />}
      {data.kind === 'TRIP_RELAY' ? relayState : data.kind === 'PUSH_BUTTON' ?
        pushStatus ? !linked ? 'Not mapped' : typeof value === 'boolean' ?
          value ? pushLatching ? 'Latched' : 'Pressed' : 'Released' : 'Awaiting state' : data.control_binding ? 'Ready' : 'Not mapped' :
        linked ? (switching ? position : (value === undefined ? 'Awaiting value' : String(value))) : 'Not mapped'}
    </div>}
    {!terminalLess && <Handle id="east" type="source" position={Position.Right} />}
    {!terminalLess && <Handle id="south" type="source" position={Position.Bottom} />}
  </div>;
}

export const EquipmentNode = memo(EquipmentNodeView);
