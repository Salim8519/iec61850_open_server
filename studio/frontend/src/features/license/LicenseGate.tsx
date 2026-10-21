import { RefreshCw, ShieldCheck, ShoppingCart } from 'lucide-react';
import { buyUrl, checkOnline, isOtherComputer, type LicenseStatus } from '../../api/license';
import { LICENSE_COPY as COPY } from '../../copy/license';
import { KeyActivation, ServerOptions, useLicenseTask } from './LicenseActions';
import { TrialStart } from './TrialStart';
import { VmHostNotice } from './VmHostNotice';
import './license-gate.css';

/** Full-screen activation shown only by packaged builds that have no valid license. */
export function LicenseGate({ status, changed }: { status: LicenseStatus; changed: (status: LicenseStatus) => void }) {
  const online = useLicenseTask();
  const purchase = buyUrl(status);
  // An ended or cancelled trial of this machine: a second trial would be refused anyway.
  // A trial file made for another computer is just a wrong file, so a trial can still start here.
  const trialEnded = status.plan === 'trial' && !isOtherComputer(status);
  return <div className="license-gate" role="main">
    <div className="license-card card">
      <div className="license-head">
        <ShieldCheck size={32} aria-hidden className="license-icon" />
        <div><h1>{COPY.title}</h1><p className="muted">{COPY.intro}</p></div>
      </div>
      {status.reason && <div className="license-message warning" role="status">{status.reason}</div>}
      {status.online_check_required && <div className="license-online">
        <button className="primary large" onClick={() => online.run(checkOnline, changed)} disabled={online.busy}>
          <RefreshCw size={16} aria-hidden />{online.busy ? 'Checking…' : COPY.checkOnlineButton}</button>
        <p className="small muted license-hint">{status.plan === 'trial' ? COPY.checkOnlineHint : COPY.checkOnlineLeaseHint}</p>
        {online.error && <div className="license-message problem" role="alert">{online.error.message}</div>}
      </div>}
      <div className="license-grid">
        <section className="license-choice">
          <h2>New here?</h2>
          <TrialStart disabled={trialEnded} changed={changed} />
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
      <VmHostNotice changed={changed} />
      <p className="small muted">{COPY.safeData}</p>
    </div>
  </div>;
}
