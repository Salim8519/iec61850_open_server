import { Upload, X } from 'lucide-react';
import { useRef, useState } from 'react';
import { snmpApi } from '../../../../api/snmp';
import { SNMP_COPY } from '../../../../copy/snmp';
import { Spinner } from '../../../../ui/Spinner';

/** Makes a simulated device from an snmpwalk file or a ScadaBench CSV file; skipped lines are listed. */
export function ImportDevice({ kind, close, done, fail }: {
  kind: 'walk' | 'csv'; close: () => void; done: () => void; fail: (message: string) => void;
}) {
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [problems, setProblems] = useState<string[]>([]);
  const file = useRef<HTMLInputElement>(null);
  const read = (picked: File) => {
    setBusy(true);
    fail('');
    picked.text().then(text => snmpApi.importDevice(kind, name || picked.name.replace(/\.[^.]+$/, ''), text))
      .then(answer => {
        done();
        if (answer.problems.length) setProblems(answer.problems);
        else close();
      }).catch(reason => fail(reason.message)).finally(() => setBusy(false));
  };
  return <div className="snmp-import guide-section" role="group" aria-label={SNMP_COPY.importTitle}>
    <div className="row"><strong>{SNMP_COPY.importTitle}</strong><span className="gateway-spacer" />
      <button className="ghost icon" aria-label="Close" onClick={close}><X size={16} /></button></div>
    {kind === 'walk' && <p className="small muted">{SNMP_COPY.importWalkHint}</p>}
    <label className="field"><span>{SNMP_COPY.importName}</span>
      <input value={name} onChange={event => setName(event.target.value)} placeholder="Bay 7 switch" /></label>
    <button disabled={busy} onClick={() => file.current?.click()}><Upload size={16} />
      {kind === 'walk' ? SNMP_COPY.addMenu.walk : SNMP_COPY.addMenu.csv}</button>
    <input ref={file} type="file" hidden accept={kind === 'csv' ? '.csv,text/csv' : '.txt,.walk,.snmpwalk,text/plain'}
      aria-label={SNMP_COPY.importTitle}
      onChange={event => { const picked = event.target.files?.[0]; if (picked) read(picked); event.target.value = ''; }} />
    {busy && <Spinner label="Reading the file…" inline />}
    {problems.length > 0 && <div className="small">
      <p className="gateway-warning" style={{ margin: 0 }}>{SNMP_COPY.importProblems(problems.length)}</p>
      <ul className="snmp-problems">{problems.slice(0, 20).map(text => <li key={text}>{text}</li>)}</ul>
      {problems.length > 20 && <p className="muted">… {problems.length - 20} more</p>}
    </div>}
  </div>;
}
