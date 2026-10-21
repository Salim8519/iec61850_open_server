import { useEffect, useState } from 'react';
import { Copy } from 'lucide-react';
import { getActivationCode } from '../../api/vmHost';
import { VM_HOST_COPY as COPY } from '../../copy/vmHost';
import './license-gate.css';

/**
 * The offline activation code: this computer's Machine ID plus the key that lets its virtual machines use
 * the license later, as one code the activation page accepts (ADR-0099). Hidden when the backend cannot
 * make one (development mode, the frozen Python engine); the plain Machine ID then does the same job
 * without virtual machine sharing.
 */
export function ActivationCode() {
  const [code, setCode] = useState('');
  const [copied, setCopied] = useState(false);
  useEffect(() => { getActivationCode().then(answer => setCode(answer.code)).catch(() => setCode('')); }, []);
  if (!code) return null;
  const copy = async () => {
    await navigator.clipboard.writeText(code); setCopied(true); setTimeout(() => setCopied(false), 2000);
  };
  return <div className="activation-code">
    <label>{COPY.activationTitle}
      <textarea readOnly rows={3} value={code} className="mono" spellCheck={false} /></label>
    <p className="small muted license-hint">{COPY.activationHint}</p>
    <div className="row"><button type="button" onClick={copy}><Copy size={15} aria-hidden />
      {copied ? COPY.copiedActivation : COPY.copyActivation}</button></div>
  </div>;
}
