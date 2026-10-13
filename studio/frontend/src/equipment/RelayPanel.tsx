import type { EquipmentData, RelayStatus } from '../api/types';
import { EquipmentSymbol } from './symbols/EquipmentSymbol';
import { relayValueKey } from './stateBindings';
import { relayRowLabel, relayValue } from './relayDisplay';
import { useLiveValue } from './liveContext';
import './relay-panel.css';

/** Value chip: text plus filled, hollow or dashed marker, never colour alone. */
export function RelayValueChip({ value }: { value: unknown }) {
  const shown = relayValue(value);
  return <span className={`relay-chip ${shown.state}`} title={shown.state === 'none' ? 'No value yet' : shown.text}>
    <span className="relay-chip-mark" aria-hidden="true" /><span className="relay-chip-text">{shown.text}</span></span>;
}

function RelayPanelRow({ nodeId, entry, iedName }: {
  nodeId: string; entry: RelayStatus; iedName?: string;
}) {
  const value = useLiveValue(relayValueKey(nodeId, entry.id));
  return <li className="relay-panel-row" title={`${entry.label}\n${entry.source.ref}`}>
    <span className="relay-panel-name">{relayRowLabel(entry, iedName)}</span>
    <RelayValueChip value={value} />
  </li>;
}

/** Canvas projection of a relay: header plus one live row per named status. */
export function RelayPanel({ nodeId, data, iedNames }: {
  nodeId: string; data: EquipmentData; iedNames: Record<string, string>;
}) {
  const entries = data.relay_statuses || [];
  return <div className="relay-panel">
    <div className="relay-panel-head">
      <span className="relay-panel-icon"><EquipmentSymbol kind="RELAY" size={22} /></span>
      <span className="relay-panel-title"><strong>{data.label}</strong>
        <small>{entries.length} signal{entries.length === 1 ? '' : 's'}</small></span>
    </div>
    {entries.length ? <ul className="relay-panel-rows">
      {entries.map(entry => <RelayPanelRow key={entry.id} nodeId={nodeId} entry={entry}
        iedName={iedNames[entry.source.ied_id]} />)}
    </ul> : <p className="relay-panel-empty">No signals · select to add</p>}
  </div>;
}
