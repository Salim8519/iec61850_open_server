import { useState } from 'react';
import { Copy, KeyRound, Upload } from 'lucide-react';
import { activateLicense, activateOffline, LicenseError, offlineUrl, updateLicenseServerUrl,
  type LicenseStatus } from '../../api/license';
import { LICENSE_COPY as COPY } from '../../copy/license';
import { Disclosure } from '../../ui/Disclosure';
import './license-gate.css';

type Changed = (status: LicenseStatus) => void;

/** Runs one license request at a time and keeps its error and error code for display. */
export function useLicenseTask() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<{ message: string; code: string } | null>(null);
  const run = async (task: () => Promise<LicenseStatus>, done: Changed) => {
    setBusy(true); setError(null);
    try { done(await task()); } catch (reason) {
      const code = reason instanceof LicenseError ? reason.code : 'error';
      setError({ message: reason instanceof Error ? reason.message : String(reason), code });
    } finally { setBusy(false); }
  };
  return { busy, error, run, clear: () => setError(null) };
}

/** License key box with the "move it here" follow-up when the key is bound elsewhere. */
export function KeyActivation({ changed, compact }: { changed: Changed; compact?: boolean }) {
  const [key, setKey] = useState('');
  const task = useLicenseTask();
  const activate = (move: boolean) => task.run(() => activateLicense(key, move), changed);
  return <form className="license-key" onSubmit={event => { event.preventDefault(); activate(false); }}>
    <label>{COPY.keyLabel}
      <input value={key} onChange={event => setKey(event.target.value)} placeholder={COPY.keyPlaceholder}
        autoComplete="off" spellCheck={false} className="mono" aria-describedby="license-key-hint" />
    </label>
    {!compact && <p id="license-key-hint" className="small muted license-hint">{COPY.keyHint}</p>}
    <div className="row">
      <button className={compact ? '' : 'primary large'} type="submit" disabled={task.busy || key.trim().length < 10}>
        <KeyRound size={16} aria-hidden />{task.busy ? 'Activating…' : COPY.activateButton}</button>
    </div>
    {task.error?.code === 'bound_elsewhere' ? <div className="license-message warning" role="alert">
      <strong>{COPY.moveQuestion}</strong><p className="small">{COPY.moveHint}</p>
      <button type="button" onClick={() => activate(true)} disabled={task.busy}>{COPY.moveButton}</button>
    </div> : task.error && <div className="license-message problem" role="alert">{task.error.message}</div>}
  </form>;
}

/** Machine ID, server address and offline activation; closed by default. */
export function ServerOptions({ status, changed }: { status: LicenseStatus; changed: Changed }) {
  const [server, setServer] = useState(status.server_url);
  const [certificate, setCertificate] = useState('');
  const [copied, setCopied] = useState(false);
  const task = useLicenseTask();
  const page = offlineUrl(status);
  const copy = async () => {
    await navigator.clipboard.writeText(status.hwid || ''); setCopied(true); setTimeout(() => setCopied(false), 2000);
  };
  const readFile = async (file: File | undefined) => { if (file) setCertificate(await file.text()); };
  return <Disclosure title={COPY.optionsTitle}>
    <div className="license-options">
      <label>{COPY.machineLabel}
        <div className="row"><input readOnly value={status.hwid || ''} className="mono" />
          <button type="button" onClick={copy}><Copy size={15} aria-hidden />{copied ? 'Copied' : COPY.copyMachine}</button></div>
      </label>
      <p className="small muted license-hint">{COPY.machineHint}</p>
      <label>{COPY.serverLabel}
        <div className="row"><input value={server} onChange={event => setServer(event.target.value)}
          placeholder="https://license.example.com" />
          <button type="button" disabled={task.busy || server === status.server_url}
            onClick={() => task.run(() => updateLicenseServerUrl(server), changed)}>Save</button></div>
      </label>
      <p className="small muted license-hint">{COPY.serverHint}</p>
      <h3>{COPY.offlineTitle}</h3>
      <ol className="small license-steps">{COPY.offlineSteps.map(step => <li key={step}>{step}</li>)}</ol>
      {page && <p className="small">Offline activation page: <code>{page}</code></p>}
      <div className="row">
        <label className="button license-file"><Upload size={15} aria-hidden />{COPY.loadFile}
          <input type="file" accept=".cert,.json,.txt" onChange={event => readFile(event.target.files?.[0])} /></label>
      </div>
      <label>{COPY.pasteLabel}
        <textarea rows={4} value={certificate} onChange={event => setCertificate(event.target.value)}
          className="mono" spellCheck={false} placeholder='{"payload": "...", "signature": "..."}' /></label>
      <div className="row"><button type="button" disabled={task.busy || !certificate.trim()}
        onClick={() => task.run(() => activateOffline(certificate), changed)}>{COPY.offlineButton}</button></div>
      {task.error && <div className="license-message problem" role="alert">{task.error.message}</div>}
    </div>
  </Disclosure>;
}
