import { Search, Send } from 'lucide-react';
import { useState, type ReactNode } from 'react';
import type { Ied } from '../../api/types';
import { hint } from '../../copy/hints';
import { commandControls } from '../../equipment/pointRequests';
import { PopupDialog } from '../../ui/PopupDialog';
import '../../ui/kind.css';
import { CommandOptionsPanel } from '../diagram/CommandOptionsPanel';
import { useTypedControls } from '../diagram/points/useTypedControls';
import { commandsState } from './allowCommands';
import { RealCommandControl } from './RealCommandControl';
import { commandRefOf, matchesSearch } from './commandSearch';
import { useDeviceOptions } from './realCommandSteps';

/** How many command rows the pop-up draws at once; the search narrows the rest. */
const SHOWN = 300;

/** The commands of a real IED in Devices, only while its own Allow commands is on (ADR-0138, ADR-0149):
 *  every class the CID gives (on/off, open/close, numbers, lists, raise/lower) in a pop-up with a search.
 *  Buttons only ask; the engine checks again and sends. */
export function RealCommands({ ied, connected, testMode }: { ied: Ied; connected: boolean; testMode: boolean }) {
  const [open, setOpen] = useState(false);
  const state = commandsState(ied.live, connected);
  if (!ied.controls.length && !ied.extended_controls?.length && state === 'off') return null;
  return <div className="row panel-note" style={{ gap: 10, flexWrap: 'wrap' }} aria-label="Commands">
    <span className="section-label">Commands</span>
    <span className="small muted" style={{ flex: '1 1 320px' }}>{hint(`live.send.${state}`)}</span>
    <button disabled={state !== 'ready'} onClick={() => setOpen(true)}><Send size={14} />Send commands…</button>
    {open && <CommandsPopup ied={ied} ready={state === 'ready'} testMode={testMode} close={() => setOpen(false)} />}
  </div>;
}

function CommandsPopup({ ied, ready, testMode, close }: { ied: Ied; ready: boolean; testMode: boolean; close: () => void }) {
  const extended = useTypedControls(ied);
  const all = commandControls(ied, extended);
  const [query, setQuery] = useState('');
  const [options, setOptions] = useDeviceOptions(ied.id, testMode);
  const shown = all.filter(control => matchesSearch(control, query));
  const why = hint(`live.send.${ready ? 'ready' : 'not_live'}`);
  return <PopupDialog title={`Commands · ${ied.name}`} label={`Commands of ${ied.name}`} close={close} wide>
    <div className="real-commands-tools">
      <p className="small muted" style={{ margin: 0 }}>{why}</p>
      <label className="search-field real-commands-search"><Search size={15} aria-hidden="true" />
        <input type="search" value={query} placeholder="Search by reference or type, e.g. CSWI3 or DPC"
          aria-label="Search commands" onChange={event => setQuery(event.target.value)} /></label>
      <span className="small muted" role="status">{shown.length === all.length ? `${all.length} commands`
        : `${shown.length} of ${all.length} commands`}{shown.length > SHOWN ? ` · first ${SHOWN} shown, search to narrow` : ''}</span>
      <CommandOptionsPanel options={options} change={setOptions} disabled={false} />
    </div>
    <div className="popup-list">{shown.slice(0, SHOWN).map(control =>
      <div key={control.ref} className="popup-list-row real-command-row">
        <code>{control.ref}</code><span className="badge" title="Command class">{control.cdc}</span>
        <RealCommandControl iedId={ied.id} iedName={ied.name} control={control} options={options} ready={ready} why={why} />
      </div>)}
      {!shown.length && <p className="small muted" style={{ margin: 0 }}>No command matches "{query}".</p>}</div>
    <div className="popup-actions"><button onClick={close}>Close</button></div>
  </PopupDialog>;
}

/** Send from the Data points rows (ADR-0149): the `.Oper.ctlVal` row of a command gets the same buttons
 *  as the pop-up while the IED is live and its Allow commands is on; other rows get nothing. */
export function useRowCommands(ied: Ied, connected: boolean, testMode: boolean): (signalRef: string) => ReactNode {
  const extended = useTypedControls(ied);
  const [options] = useDeviceOptions(ied.id, testMode);
  const ready = commandsState(ied.live, connected) === 'ready';
  const byRef = new Map(commandControls(ied, extended).map(control => [control.ref, control]));
  return signalRef => {
    const ref = commandRefOf(signalRef);
    const control = ref && ready ? byRef.get(ref) : undefined;
    return control ? <RealCommandControl iedId={ied.id} iedName={ied.name} control={control} options={options}
      ready={ready} why={hint('live.send.ready')} /> : null;
  };
}
