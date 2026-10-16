import { ShieldCheck, ShoppingCart, Sparkles } from 'lucide-react';
import { buyUrl, startTrial, type LicenseStatus } from '../../api/license';
import { LICENSE_COPY as COPY } from '../../copy/license';
import { KeyActivation, ServerOptions, useLicenseTask } from './LicenseActions';
import './license-gate.css';

/** Full-screen activation shown only by packaged builds that have no valid license. */
export function LicenseGate({ status, changed }: { status: LicenseStatus; changed: (status: LicenseStatus) => void }) {
  const trial = useLicenseTask();
  const purchase = buyUrl(status);
  // An expired trial certificate for this machine: a second trial would be refused anyway.
  const trialEnded = status.plan === 'trial' && !status.transfer_available;
  return <div className="license-gate" role="main">
    <div className="license-card card">
      <div className="license-head">
        <ShieldCheck size={32} aria-hidden className="license-icon" />
        <div><h1>{COPY.title}</h1><p className="muted">{COPY.intro}</p></div>
      </div>
      {status.reason && <div className="license-message warning" role="status">{status.reason}</div>}
      <div className="license-grid">
        <section className="license-choice">
          <h2>New here?</h2>
          <button className="large" onClick={() => trial.run(startTrial, changed)}
            disabled={trial.busy || trialEnded}>
            <Sparkles size={16} aria-hidden />{trial.busy ? 'Starting trial…' : COPY.trialButton}</button>
          <p className="small muted license-hint">{COPY.trialHint}</p>
          {trial.error && <div className="license-message problem" role="alert">{trial.error.message}</div>}
          {purchase && <>
            <a className="button large" href={purchase} target="_blank" rel="noreferrer">
              <ShoppingCart size={16} aria-hidden />{COPY.buyButton}</a>
            <p className="small muted license-hint">{COPY.buyHint}</p>
          </>}
        </section>
        <section className="license-choice">
          <h2>Have a key?</h2>
          <KeyActivation changed={changed} />
        </section>
      </div>
      <ServerOptions status={status} changed={changed} />
      <p className="small muted">{COPY.safeData}</p>
    </div>
  </div>;
}
