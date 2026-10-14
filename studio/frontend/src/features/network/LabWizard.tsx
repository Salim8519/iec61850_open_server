import { ArrowLeft, ArrowRight, Check, X } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { Adapter, ApplyResult, PlanPreview } from '../../api/network';
import { Hint } from '../../ui/Hint';
import { StepWizard } from '../../ui/StepWizard';
import { AdapterStep } from './AdapterStep';
import { AddressStep, type Manual } from './AddressStep';
import { ResultStep, ReviewStep } from './ReviewStep';

const STEPS = ['Choose adapter', 'Plan addresses', 'Review changes', 'Done'];

export function LabWizard({ close, finished }: { close: () => void; finished: () => void }) {
  const [step, setStep] = useState(0);
  const [adapters, setAdapters] = useState<Adapter[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [guid, setGuid] = useState('');
  const [strategy, setStrategy] = useState<'scd' | 'adapter'>('scd');
  const [manual, setManual] = useState<Manual>({});
  const [preview, setPreview] = useState<PlanPreview | null>(null);
  const [checking, setChecking] = useState(false);
  const [result, setResult] = useState<ApplyResult | null>(null);
  const [error, setError] = useState('');
  const [applying, setApplying] = useState(false);
  const load = useCallback((refresh: boolean, quiet = false) => {
    if (!quiet) { setLoading(true); setError(''); }
    api<{ adapters: Adapter[] }>(`/network/adapters${refresh ? '?refresh=1' : ''}`)
      .then(answer => setAdapters(answer.adapters)).catch(reason => { if (!quiet) setError(reason.message); })
      .finally(() => { if (!quiet) setLoading(false); });
  }, []);
  useEffect(() => load(false), [load]);
  // Live list: a plugged or unplugged cable shows up by itself while the adapter step is open.
  useEffect(() => {
    if (step !== 0) return;
    const timer = window.setInterval(() => load(true, true), 3000);
    return () => window.clearInterval(timer);
  }, [step, load]);
  const check = useCallback(() => {
    if (!guid) return;
    setChecking(true); setError('');
    api<PlanPreview>('/network/plan/preview', 'POST', { adapter_guid: guid, strategy, manual })
      .then(setPreview).catch(reason => setError(reason.message)).finally(() => setChecking(false));
  }, [guid, strategy, manual]);
  useEffect(() => { if (step === 1) { setPreview(null); check(); } }, [step, strategy]);
  const apply = async () => {
    if (!preview) return;
    setApplying(true); setError('');
    try {
      setResult(await api<ApplyResult>('/network/plan/apply', 'POST', { fingerprint: preview.fingerprint, confirm: true }));
      setStep(3);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally { setApplying(false); }
  };
  const footer = <>
    <button onClick={step === 0 || step === 3 ? close : () => setStep(step - 1)} disabled={applying}>
      {step === 0 || step === 3 ? <><X size={14} />Close</> : <><ArrowLeft size={14} />Back</>}</button>
    <div className="row">
      {step === 1 && <button onClick={check} disabled={checking}>Check again</button>}
      {step === 0 && <button className="primary" disabled={!guid} onClick={() => setStep(1)}>Next<ArrowRight size={14} /></button>}
      {step === 1 && <button className="primary" disabled={!preview?.ready || checking} onClick={() => setStep(2)}>Next<ArrowRight size={14} /></button>}
      {step === 2 && <button className="primary" disabled={applying || !preview} onClick={apply}>
        <Check size={14} />{applying ? 'Waiting for Windows…' : preview?.needs_admin ? 'Apply changes' : 'Use these addresses'}</button>}
      {step === 3 && <button className="primary" onClick={finished}>Finish</button>}
    </div>
  </>;
  return <div className="card"><div className="guide-section">
    <StepWizard steps={STEPS} current={step} footer={footer}>
      {error && <div className="error-banner" role="alert" style={{ borderRadius: 6 }}><span>{error}</span></div>}
      {step === 0 && <AdapterStep adapters={adapters} loading={loading} selected={guid} choose={setGuid} reload={() => load(true)} />}
      {step === 1 && <AddressStep strategy={strategy} setStrategy={value => { setStrategy(value); setManual({}); }}
        preview={preview} manual={manual} setManual={setManual} checking={checking} />}
      {step === 1 && preview && !preview.ready && <Hint id="network.address.fix" />}
      {step === 2 && preview && <ReviewStep preview={preview} />}
      {step === 3 && result && <ResultStep result={result} />}
    </StepWizard>
  </div></div>;
}
