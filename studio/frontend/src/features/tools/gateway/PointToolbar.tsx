import { ChevronDown, Download, ListChecks, Lock, Plus, Save, Sparkles, Trash2, Undo2, Upload } from 'lucide-react';
import { useRef } from 'react';
import { gatewayApi, type Protocol } from '../../../api/gateway';
import { GATEWAY_COPY, OWN_COPY } from '../../../copy/gateway';
import { HelpTip } from '../../../ui/HelpTip';
import { Menu } from '../../../ui/Menu';
import { saveFile } from './saveFile';

/** The point list's buttons. Propose, Add point and Select stay in sight; the rarer file and clean-up
 *  actions sit in "More". Nothing here changes data by itself: every action is a callback of the table. */
export function PointToolbar({ protocol, locked, dirty, adding, hasRows, picking, propose, add, togglePicking, removeAll,
  importFile, discard, save }: {
  protocol: Protocol; locked: boolean; dirty: boolean; adding: boolean; hasRows: boolean; picking: boolean;
  propose: () => void; add: () => void; togglePicking: () => void; removeAll: () => void;
  importFile: (file: File) => void; discard: () => void; save: () => void;
}) {
  const file = useRef<HTMLInputElement>(null);
  return <div className="panel-toolbar">
    <button className={dirty ? '' : 'primary'} onClick={propose} disabled={locked}><Sparkles size={16} />{GATEWAY_COPY.propose}</button>
    <button onClick={add} disabled={locked || adding} title={OWN_COPY.addHint}><Plus size={16} />{OWN_COPY.add}</button>
    {!locked && hasRows && <button className={picking ? 'active-toggle' : ''} aria-pressed={picking}
      title={GATEWAY_COPY.selectHint} onClick={togglePicking}><ListChecks size={16} />{GATEWAY_COPY.select}</button>}
    <Menu label={GATEWAY_COPY.moreLabel} align="left" trigger={<>{GATEWAY_COPY.more}<ChevronDown size={15} aria-hidden /></>} items={[
      { label: GATEWAY_COPY.importCsv, icon: <Upload size={16} />, onSelect: () => file.current?.click(), disabled: locked },
      { label: GATEWAY_COPY.exportCsv, icon: <Download size={16} />, onSelect: () => saveFile(gatewayApi.csvUrl(protocol)) },
      { label: GATEWAY_COPY.removeAll, icon: <Trash2 size={16} />, onSelect: removeAll, disabled: locked || !hasRows }]} />
    <input ref={file} type="file" accept=".csv,text/csv" hidden aria-label="CSV point list"
      onChange={event => { const picked = event.target.files?.[0]; if (picked) importFile(picked); event.target.value = ''; }} />
    <HelpTip text={GATEWAY_COPY.pointsHint} label="What is the point list?" />
    <span className="gateway-spacer" />
    {dirty && <><span className="small gateway-warning">{GATEWAY_COPY.unsaved}</span>
      <button className="ghost" onClick={discard}><Undo2 size={16} />{GATEWAY_COPY.discard}</button>
      <button className="primary" onClick={save} disabled={locked}><Save size={16} />{GATEWAY_COPY.save}</button></>}
    {locked && <span className="small muted row"><Lock size={14} />{GATEWAY_COPY.lockedPoints}</span>}
  </div>;
}
