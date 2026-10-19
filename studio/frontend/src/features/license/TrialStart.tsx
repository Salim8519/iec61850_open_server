import { useState } from 'react';
import { Mail, Sparkles } from 'lucide-react';
import { LicenseError, requestTrialCode, startTrial, type LicenseStatus } from '../../api/license';
import { LICENSE_COPY as COPY } from '../../copy/license';
import { useLicenseTask } from './LicenseActions';

type Changed = (status: LicenseStatus) => void;
const EMAIL_STEP = new Set(['email_required', 'email_used', 'email_disposable', 'bad_email', 'bad_code']);

/**
 * Start-trial button. When the server asks for a confirmed email (this PC or network
 * started trials before), the same place shows email, then code, then "Start free trial".
 */
export function TrialStart({ disabled, changed }: { disabled: boolean; changed: Changed }) {
  const trial = useLicenseTask();
  const [email, setEmail] = useState('');
  const [code, setCode] = useState('');
  const [codeSent, setCodeSent] = useState(false);
  const [sending, setSending] = useState(false);
  const [sendError, setSendError] = useState<string | null>(null);
  const emailStep = codeSent || sendError !== null || EMAIL_STEP.has(trial.error?.code ?? '');
  const busy = trial.busy || sending;

  const sendCode = async () => {
    setSending(true); setSendError(null); trial.clear();
    try {
      const answer = await requestTrialCode(email);
      if (answer.status === 'no_code_needed') await trial.run(() => startTrial(email), changed);  // mail down: never block
      else setCodeSent(true);
    } catch (reason) {
      setSendError(reason instanceof LicenseError || reason instanceof Error ? reason.message : String(reason));
    } finally { setSending(false); }
  };

  if (!emailStep) {
    return <>
      <button className="large" onClick={() => trial.run(() => startTrial(), changed)} disabled={busy || disabled}>
        <Sparkles size={16} aria-hidden />{trial.busy ? 'Starting trial…' : COPY.trialButton}</button>
      <p className="small muted license-hint">{COPY.trialHint}</p>
      {trial.error && <div className="license-message problem" role="alert">{trial.error.message}</div>}
    </>;
  }
  const problem = sendError ?? trial.error?.message;
  return <form className="license-key" onSubmit={event => {
    event.preventDefault();
    if (codeSent) trial.run(() => startTrial(email, code), changed); else sendCode();
  }}>
    {problem && <div className="license-message warning" role="alert">{problem}</div>}
    <label>{COPY.emailLabel}
      <input type="email" value={email} onChange={event => { setEmail(event.target.value); setCodeSent(false); }}
        autoComplete="email" placeholder="name@company.com" aria-describedby="trial-email-hint" />
    </label>
    <p id="trial-email-hint" className="small muted license-hint">{COPY.emailHint}</p>
    {codeSent && <>
      <label>{COPY.codeLabel}
        <input value={code} onChange={event => setCode(event.target.value)} inputMode="numeric"
          autoComplete="one-time-code" maxLength={6} className="mono" aria-describedby="trial-code-hint" />
      </label>
      <p id="trial-code-hint" className="small muted license-hint">{COPY.codeHint}</p>
    </>}
    <div className="row">
      {codeSent
        ? <button className="primary large" type="submit" disabled={busy || code.trim().length !== 6}>
            <Sparkles size={16} aria-hidden />{trial.busy ? 'Starting trial…' : COPY.startWithEmail}</button>
        : <button className="primary large" type="submit" disabled={busy || !email.includes('@')}>
            <Mail size={16} aria-hidden />{sending ? 'Sending…' : COPY.sendCode}</button>}
      {codeSent && <button type="button" onClick={sendCode} disabled={busy}>{COPY.sendCode}</button>}
    </div>
  </form>;
}
