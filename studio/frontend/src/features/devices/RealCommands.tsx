import { Send, TriangleAlert } from 'lucide-react';
import { useState } from 'react';
import { api } from '../../api/client';
import type { Ied } from '../../api/types';
import { GATEWAY_COPY } from '../../copy/gateway';
import { hint } from '../../copy/hints';
import { PopupDialog } from '../../ui/PopupDialog';
import '../../ui/kind.css';
import { commandsState, commandWords } from './allowCommands';

type Pending = { ref: string; label: string; value: boolean };
type Answer = { ok: boolean; summary: string; observed: string | null; changed: boolean };
type Result = Pending & { answer?: Answer; error?: string };

/** Close/Open (On/Off) of a real IED's commands in Devices, only while its own Allow commands is on
 *  (ADR-0138). One line above Data points; the commands open in a pop-up so the values stay in view.
 *  Buttons only open the warning; the engine checks again and sends. */
export function RealCommands({ ied, connected }: { ied: Ied; connected: boolean }) {
  const [open, setOpen] = useState(false);
  const state = commandsState(ied.live, connected);
  if (!ied.controls.length) return null;
  return <div className="row panel-note" style={{ gap: 10, flexWrap: 'wrap' }} aria-label="Commands">
    <span className="section-label">Commands</span>
    <span className="small muted" style={{ flex: '1 1 320px' }}>{hint(`live.send.${state}`)}</span>
    <button disabled={state !== 'ready'} onClick={() => setOpen(true)}><Send size={14} />Send commands…</button>
    {open && <CommandsPopup ied={ied} ready={state === 'ready'} close={() => setOpen(false)} />}
  </div>;
}

function CommandsPopup({ ied, ready, close }: { ied: Ied; ready: boolean; close: () => void }) {
  const [pending, setPending] = useState<Pending | null>(null);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<Result | null>(null);
  const send = () => {
    if (!pending) return;
    setBusy(true);
    api<Answer>(`/live/${ied.id}/operate`, 'POST', { ref: pending.ref, value: pending.value, confirm: true })
      .then(answer => setResult({ ...pending, answer })).catch(reason => setResult({ ...pending, error: reason.message }))
      .finally(() => { setBusy(false); setPending(null); });
  };
  const failed = result && (result.error || !result.answer?.ok);
  return <PopupDialog title={`Commands · ${ied.name}`} label={`Commands of ${ied.name}`} busy={busy} close={close} wide>
    {pending ? <div className="confirm-box" style={{ borderWidth: 2, borderColor: 'var(--error)' }} role="alertdialog" aria-label="Confirm command">
      <p className="row" style={{ gap: 8 }}><TriangleAlert size={18} style={{ color: 'var(--error)' }} aria-hidden />
        <strong>{GATEWAY_COPY.confirmTitle(pending.label, `${ied.name} · ${pending.ref}`)}</strong></p>
      <p className="small">{GATEWAY_COPY.realWarning}</p>
      <div className="row" style={{ gap: 8 }}>
        <button onClick={() => setPending(null)} disabled={busy} autoFocus>{GATEWAY_COPY.cancel}</button>
        <button className="danger" disabled={busy} onClick={send}><Send size={14} />{GATEWAY_COPY.send(pending.label)}</button>
      </div></div>
      : <>
        <p className="small muted" style={{ margin: 0 }}>{hint(`live.send.${ready ? 'ready' : 'not_live'}`)}</p>
        {result && <p className={`small popup-result${failed ? '' : ' ok'}`} role="status">
          <strong>{result.label} {result.ref}:</strong> {result.error ?? result.answer?.summary}
          {result.answer?.observed ? ` Position: ${result.answer.observed}.` : ''}</p>}
        <div className="popup-list">{ied.controls.map(control =>
          <span key={control.ref} className="popup-list-row"><code>{control.ref}</code>
            {commandWords(control.cdc).map((word, index) => <button key={word} disabled={busy || !ready}
              onClick={() => { setResult(null); setPending({ ref: control.ref, label: word, value: index === 0 }); }}>
              <Send size={14} />{word}…</button>)}
          </span>)}</div>
        <div className="popup-actions"><button onClick={close}>Close</button></div>
      </>}
  </PopupDialog>;
}
