import { useContext, type MouseEvent } from 'react';
import { Bot, ChevronDown, ChevronUp, Hand } from 'lucide-react';
import type { EquipmentData } from '../api/types';
import { EquipmentSymbol } from './symbols/EquipmentSymbol';
import { CanvasLocked } from './pointPanelContext';
import { StoppedIeds } from './liveContext';
import { TapBusy, useTapCommand, useTapValues } from './tapContext';
import { avrLine, isAuto, lvText, modeWords, newTapChanger, stepBlock, tapText } from './tapChanger';
import { TAP_COPY } from '../copy/tapChanger';
import './tap-transformer.css';

/** Canvas body of a tap-changer transformer (ADR-0109): HV and LV sides, "Tap 9 / 1–17", an Auto/Manual
 *  badge (icon and word, dashed for Manual) and, while running, ▲ Raise / ▼ Lower. The engine owns the tap;
 *  this only shows what it publishes and sends the button presses. */
export function TapTransformerBody({ nodeId, data }: { nodeId: string; data: EquipmentData }) {
  const tap = data.tap_changer ?? newTapChanger();
  const locked = useContext(CanvasLocked);
  const stopped = useContext(StoppedIeds);
  const busy = useContext(TapBusy) === nodeId;
  const send = useTapCommand();
  const { tapValue, modeValue, voltageValue, avr } = useTapValues(nodeId, tap);
  const auto = isAuto(tap, modeValue, avr);
  const running = locked && !(tap.tap_source && stopped.has(tap.tap_source.ied_id));
  const regulator = avrLine(avr);
  const lv = lvText(voltageValue, avr);
  const tone = !tap.tap_source ? 'unmapped' : typeof tapValue === 'number' ? 'live' : 'awaiting';
  const press = (action: 'raise' | 'lower') => (event: MouseEvent) => { event.stopPropagation(); send(nodeId, action); };
  const button = (action: 'raise' | 'lower') => {
    const block = stepBlock(tap, tapValue, auto, action);
    const words = action === 'raise' ? TAP_COPY.raise : TAP_COPY.lower;
    return <button className="tap-step nodrag" disabled={!!block || busy} onClick={press(action)}
      title={block ?? (action === 'raise' ? TAP_COPY.raiseHint : TAP_COPY.lowerHint)} aria-label={`${words} ${data.label}`}>
      {action === 'raise' ? <ChevronUp size={14} aria-hidden="true" /> : <ChevronDown size={14} aria-hidden="true" />}
      {words}</button>;
  };
  return <>
    <span className="tap-side" aria-hidden="true">HV</span>
    <div className="equipment-symbol" style={{ transform: `rotate(${data.rotation || 0}deg)` }}>
      <EquipmentSymbol kind="TAP_TRANSFORMER" /></div>
    <span className="tap-side" aria-hidden="true">LV</span>
    <div className="equipment-label">{data.label}</div>
    <div className={`equipment-state tone-${tone}`}>{tapText(tap, tapValue)}</div>
    <div className={`tap-mode ${auto ? 'auto' : 'manual'}`} title={auto ? 'Auto: the voltage regulator moves the tap' :
      'Manual: only an operator moves the tap'}>
      {auto ? <Bot size={12} aria-hidden="true" /> : <Hand size={12} aria-hidden="true" />}{modeWords(auto)}</div>
    {running && lv && <div className="tap-line">{lv}</div>}
    {running && regulator && <div className="tap-line">{regulator}</div>}
    {running && <div className="tap-steps">{button('raise')}{button('lower')}</div>}
  </>;
}
