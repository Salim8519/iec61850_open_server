import { useState } from 'react';
import { snmpApi, type ValueRow } from '../../../../api/snmp';
import { READ_COPY, refusedWrite } from '../../../../copy/snmpRead';

/** The short confirmation before ScadaBench writes a value into the device it reads (SET). */
export function WriteValue({ row, close, fail }: { row: ValueRow; close: () => void; fail: (message: string) => void }) {
  const [value, setValue] = useState(row.value);
  const [busy, setBusy] = useState(false);
  const [answer, setAnswer] = useState('');
  const send = () => {
    setBusy(true);
    fail('');
    snmpApi.write({ oid: row.oid, type: row.type, value, hex: row.hex })
      .then(result => setAnswer(result.values.map(item => `${item.value}${item.meaning ? ` · ${item.meaning}` : ''}`).join(', ')))
      .catch(reason => fail(refusedWrite(reason.message))).finally(() => setBusy(false));
  };
  return <div className="guide-section panel-messages snmp-write" role="alertdialog" aria-label={READ_COPY.setTitle(row.name)}>
    <strong>{READ_COPY.setTitle(row.name)}</strong>
    <p className="small muted" style={{ margin: 0 }}>{READ_COPY.setHint}</p>
    <div className="row">
      <label>{READ_COPY.newValue}<input autoFocus value={value} onChange={event => setValue(event.target.value)} /></label>
      <button className="primary" disabled={busy || Boolean(answer)} onClick={send}>{READ_COPY.setSend}</button>
      <button className="ghost" onClick={close}>{answer ? 'Close' : READ_COPY.cancel}</button>
    </div>
    {answer && <p className="small snmp-live" role="status">✓ The device now reads: {answer}</p>}
  </div>;
}
