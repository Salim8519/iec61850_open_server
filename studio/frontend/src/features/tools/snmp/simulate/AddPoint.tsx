import { X } from 'lucide-react';
import { useState } from 'react';
import type { PointType, SnmpPoint } from '../../../../api/snmp';
import { POINTS_COPY, TYPE_NAMES } from '../../../../copy/snmp';
import { oidOf } from './oidText';

/** A value of the user's own: an OID (or a MIB name), its label, type and first value. */
export function AddPoint({ points, add, close, fail }: {
  points: SnmpPoint[]; add: (point: SnmpPoint) => Promise<unknown>; close: () => void; fail: (message: string) => void;
}) {
  const [name, setName] = useState('');
  const [label, setLabel] = useState('');
  const [type, setType] = useState<PointType>('integer');
  const [value, setValue] = useState('0');
  const [writable, setWritable] = useState(false);
  const submit = async () => {
    fail('');
    try {
      const oid = await oidOf(name, points);
      await add({ oid, label: label.trim() || undefined, type, value, behaviour: 'fixed', writable });
    } catch (reason) {
      fail((reason as Error).message);
    }
  };
  return <form className="guide-section panel-messages snmp-add-form" aria-label={POINTS_COPY.addTitle}
    onSubmit={event => { event.preventDefault(); submit(); }}>
    <div className="row"><strong>{POINTS_COPY.addTitle}</strong><span className="gateway-spacer" />
      <button type="button" className="ghost icon" aria-label="Close" onClick={close}><X size={16} /></button></div>
    <p className="small muted" style={{ margin: 0 }}>{POINTS_COPY.addHint}</p>
    <div className="row snmp-add-fields">
      <label>{POINTS_COPY.oid}<input value={name} onChange={event => setName(event.target.value)} required
        placeholder="1.3.6.1.4.1.99999.1.0" spellCheck={false} /></label>
      <label>{POINTS_COPY.label}<input value={label} onChange={event => setLabel(event.target.value)} maxLength={120}
        placeholder={POINTS_COPY.labelPlaceholder} /></label>
      <label>{POINTS_COPY.type}<select value={type} onChange={event => {
        const next = event.target.value as PointType;
        setType(next);
        setValue(next === 'octets' ? '' : next === 'oid' ? '0.0' : next === 'ipaddress' ? '0.0.0.0' : '0');
      }}>{(Object.keys(TYPE_NAMES) as PointType[]).filter(key => key !== 'null')
          .map(key => <option key={key} value={key}>{TYPE_NAMES[key]}</option>)}</select></label>
      <label>{POINTS_COPY.value}<input value={value} onChange={event => setValue(event.target.value)} /></label>
      <label className="row gateway-check"><input type="checkbox" checked={writable}
        onChange={event => setWritable(event.target.checked)} />{POINTS_COPY.writable}</label>
      <button className="primary" type="submit" disabled={!name.trim()}>{POINTS_COPY.addButton}</button>
    </div>
  </form>;
}
