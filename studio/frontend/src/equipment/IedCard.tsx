import { useContext } from 'react';
import type { EquipmentData } from '../api/types';
import { IED_COPY } from '../copy/iedComponent';
import { EquipmentSymbol } from './symbols/EquipmentSymbol';
import { IedRecords, useIedLive } from './iedContext';
import { StoppedIeds } from './liveContext';
import { CanvasLocked } from './pointPanelContext';
import './ied-card.css';

/** Canvas projection of an IED component (ADR-0112): one whole IED. The full IED name (wrapped,
 *  never cut), simulated or real, running or not, its GOOSE in words ("Sends 2 messages · gets 1"
 *  or "No GOOSE"), and, while the layers are on, its GOOSE problems and readers. No terminals;
 *  GOOSE lines start, pass and end here, never on equipment. */
export function IedCard({ data }: { data: EquipmentData }) {
  const records = useContext(IedRecords);
  const locked = useContext(CanvasLocked);
  const stoppedIeds = useContext(StoppedIeds);
  const live = useIedLive(data.ied_id);
  const record = data.ied_id ? records[data.ied_id] : undefined;
  if (!record) return <div className="ied-card empty">
    <div className="ied-card-head"><span className="ied-card-icon"><EquipmentSymbol kind="IED" size={22} /></span>
      <strong className="ied-card-name">{data.label}</strong></div>
    <p className="ied-card-line muted">{IED_COPY.chooseIed}</p></div>;
  const running = locked && record.role !== 'off' && !stoppedIeds.has(data.ied_id!);
  const state = IED_COPY.state(record.role, locked, running);
  const goose = IED_COPY.goose(record.sends, record.gets);
  return <div className={`ied-card role-${record.role}${running ? ' running' : ''}`} title={IED_COPY.cardHint}>
    <div className="ied-card-head"><span className="ied-card-icon"><EquipmentSymbol kind="IED" size={22} /></span>
      <strong className="ied-card-name">{record.name}</strong></div>
    <p className={`ied-card-state ${state.tone}`}><span className="ied-card-dot" aria-hidden />{state.text}</p>
    <p className={`ied-card-line${record.sends || record.gets ? '' : ' muted'}`}>{goose}</p>
    {live?.problems.map(problem => <p key={problem.text} className={`ied-card-problem ${problem.tone}`} role="status">
      {problem.text}</p>)}
    {live?.readers && <p className="ied-card-readers" title={live.readersTitle}>{live.readers}</p>}
  </div>;
}
