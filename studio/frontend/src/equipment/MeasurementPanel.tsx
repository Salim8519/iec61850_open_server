import { useContext } from 'react';
import { TriangleAlert } from 'lucide-react';
import type { EquipmentData, Measurement } from '../api/types';
import { EquipmentSymbol } from './symbols/EquipmentSymbol';
import { relayValueKey } from './stateBindings';
import { shortRef } from './relayDisplay';
import { formatMeasurement, MEASUREMENT_COPY, qualityBad } from './measurementDisplay';
import { StoppedIeds, useLiveValue } from './liveContext';
import { useSignalInfo } from './useSignalInfo';
import { useSignalValue } from '../state/useLive';
import './measurement-panel.css';

/** Scaled value chip: the text says the state; border pattern and icon repeat it, never colour alone. */
export function MeasurementValue({ raw, entry, stopped, bad }: {
  raw: unknown; entry: Measurement; stopped: boolean; bad: boolean;
}) {
  const shown = formatMeasurement(raw, entry, stopped);
  const title = shown.state === 'none' ? MEASUREMENT_COPY.noValue : shown.state === 'stopped' ? MEASUREMENT_COPY.stopped
    : bad ? `${shown.text} · ${MEASUREMENT_COPY.badQuality}` : `${shown.text} (raw ${String(raw)})`;
  return <span className={`measure-chip ${shown.state}${bad && shown.state === 'value' ? ' bad' : ''}`} title={title}>
    {bad && shown.state === 'value' && <TriangleAlert size={11} aria-hidden="true" />}
    <span className="measure-chip-text">{shown.text}</span></span>;
}

/** Live raw value of a row plus its quality flag; the equipment store is filled while running. */
export function useMeasurementLive(nodeId: string, entry: Measurement) {
  const raw = useLiveValue(relayValueKey(nodeId, entry.id));
  const signal = useSignalInfo(entry.source);
  const quality = useSignalValue(signal?.quality_ref ? `${entry.source.ied_id}:${signal.quality_ref}` : null);
  const stopped = useContext(StoppedIeds).has(entry.source.ied_id);
  return { raw, signal, stopped, bad: raw !== undefined && raw !== null && qualityBad(quality) };
}

/** Untouched names that equal the reference are shortened, as on relay panels. */
export function measurementName(entry: Measurement, iedName?: string): string {
  return entry.label === entry.source.ref ? shortRef(entry.label, iedName) : entry.label;
}

function PanelRow({ nodeId, entry, iedName }: { nodeId: string; entry: Measurement; iedName?: string }) {
  const live = useMeasurementLive(nodeId, entry);
  return <li className="measure-panel-row" title={`${entry.label}\n${entry.source.ref}`}>
    <span className="measure-panel-name">{measurementName(entry, iedName)}
      {live.bad && <small className="measure-panel-flag">Bad quality</small>}</span>
    <MeasurementValue raw={live.raw} entry={entry} stopped={live.stopped} bad={live.bad} />
  </li>;
}

/** Canvas projection of a measurement list: header plus one scaled live row per measurement. */
export function MeasurementPanel({ nodeId, data, iedNames }: {
  nodeId: string; data: EquipmentData; iedNames: Record<string, string>;
}) {
  const entries = data.measurements || [];
  return <div className="measure-panel">
    <div className="measure-panel-head">
      <span className="measure-panel-icon"><EquipmentSymbol kind="MEASUREMENT_LIST" size={22} /></span>
      <span className="measure-panel-title"><strong>{data.label}</strong>
        <small>{entries.length} measurement{entries.length === 1 ? '' : 's'}</small></span>
    </div>
    {entries.length ? <ul className="measure-panel-rows">
      {entries.map(entry => <PanelRow key={entry.id} nodeId={nodeId} entry={entry}
        iedName={iedNames[entry.source.ied_id]} />)}
    </ul> : <p className="measure-panel-empty">{MEASUREMENT_COPY.empty}</p>}
  </div>;
}
