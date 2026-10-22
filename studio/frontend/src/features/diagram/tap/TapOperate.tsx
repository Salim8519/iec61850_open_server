import { useContext } from 'react';
import { Bot, ChevronDown, ChevronUp, Hand } from 'lucide-react';
import type { EquipmentData } from '../../../api/types';
import { TapBusy, useTapCommand, useTapValues } from '../../../equipment/tapContext';
import { avrLine, isAuto, lvText, modeWords, newTapChanger, stepBlock, tapText } from '../../../equipment/tapChanger';
import { TAP_COPY } from '../../../copy/tapChanger';

/** Operate section of a running tap-changer transformer (ADR-0109): tap, mode, LV voltage and regulator in
 *  words, then Raise / Lower and Auto / Manual. Disabled buttons say why in their hint. */
export function TapOperate({ nodeId, data, message }: { nodeId: string; data: EquipmentData; message?: string }) {
  const tap = data.tap_changer ?? newTapChanger();
  const send = useTapCommand();
  const busy = useContext(TapBusy) === nodeId;
  const { tapValue, modeValue, voltageValue, avr } = useTapValues(nodeId, tap);
  const auto = isAuto(tap, modeValue, avr);
  const facts = [tapText(tap, tapValue), modeWords(auto), lvText(voltageValue, avr), avrLine(avr)].filter(Boolean);
  const raise = stepBlock(tap, tapValue, auto, 'raise');
  const lower = stepBlock(tap, tapValue, auto, 'lower');
  const canAuto = !!tap.mode_control || (!tap.mode_source && tap.avr.enabled);
  const canManual = !!tap.mode_control || !tap.mode_source;
  return <section className="sheet-section"><h3>{TAP_COPY.operateTitle}</h3>
    <p className="energy-sentence" role="status">{facts.join(' · ')}</p>
    <div className="tap-actions">
      <button disabled={!!raise || busy} title={raise ?? TAP_COPY.raiseHint} onClick={() => send(nodeId, 'raise')}>
        <ChevronUp size={16} aria-hidden="true" />{TAP_COPY.raise}</button>
      <button disabled={!!lower || busy} title={lower ?? TAP_COPY.lowerHint} onClick={() => send(nodeId, 'lower')}>
        <ChevronDown size={16} aria-hidden="true" />{TAP_COPY.lower}</button>
      <button className={auto ? 'primary' : ''} aria-pressed={!!auto} disabled={!canAuto || busy || !!auto}
        title={canAuto ? 'The voltage regulator moves the tap' : TAP_COPY.noAuto} onClick={() => send(nodeId, 'auto')}>
        <Bot size={16} aria-hidden="true" />{TAP_COPY.auto}</button>
      <button className={auto === false ? 'primary' : ''} aria-pressed={auto === false} disabled={!canManual || busy || auto === false}
        title="Only an operator moves the tap" onClick={() => send(nodeId, 'manual')}>
        <Hand size={16} aria-hidden="true" />{TAP_COPY.manual}</button>
    </div>
    {(raise ?? lower) && <p className="small muted" style={{ margin: '6px 0 0' }}>{raise ?? lower}</p>}
    {!canAuto && <p className="small muted" style={{ margin: '6px 0 0' }}>{TAP_COPY.noAuto}</p>}
    {message && <p className="small" role="status" style={{ margin: '6px 0 0' }}>{message}</p>}
  </section>;
}
