import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { liveFeed } from '../../../api/polls';
import { SNMP_STATUS, snmpApi, type SnmpConfig, type SnmpRole, type SnmpStatus } from '../../../api/snmp';
import { SNMP_COPY, SNMP_ROLES } from '../../../copy/snmp';
import { HelpTip } from '../../../ui/HelpTip';
import { SegmentedControl } from '../../../ui/SegmentedControl';
import { ReadView } from './read/ReadView';
import { SimulateView } from './simulate/SimulateView';
import '../explorer/explorer.css';
import '../gateway/gateway.css';
import './snmp.css';

/** Tools › SNMP devices (ADR-0092): "Act as" Simulate devices or Read devices in the header row (the long
 *  sentence about the role sits behind its "?"), the chosen view fills the window. Both may run at once;
 *  switching never stops anything. */
export function SnmpTool({ header }: { header: (extra?: ReactNode) => ReactNode }) {
  const [status, setStatus] = useState<SnmpStatus | null>(null);
  const [config, setConfig] = useState<SnmpConfig | null>(null);
  const [error, setError] = useState('');
  const reload = useCallback(() => snmpApi.config().then(setConfig).catch(reason => setError(reason.message)), []);
  useEffect(() => {
    reload();
    return liveFeed.subscribe<SnmpStatus>(SNMP_STATUS, 1500, value => { setStatus(value); setError(''); },
      reason => setError(reason.message));
  }, [reload]);
  const role: SnmpRole = status?.role ?? config?.role ?? 'simulate';
  const change = (next: SnmpRole) => snmpApi.setRole(next).then(setStatus).catch(reason => setError(reason.message));
  const switcher = <div className="role-switch">
    <strong>{SNMP_COPY.actAs}</strong>
    <SegmentedControl label={SNMP_COPY.actAs} value={role} onChange={change}
      options={(['simulate', 'read'] as SnmpRole[]).map(value => ({ value, label: SNMP_ROLES[value].label,
        title: SNMP_ROLES[value].sentence }))} />
    <HelpTip text={`${SNMP_ROLES[role].sentence} ${SNMP_COPY.roleHint}`} label="What does each role do?" />
  </div>;
  if (!status || !config) {
    const missing = error.includes('does not provide');
    return <>{header()}<div className="card fill-body"><div className="guide-empty">
      <p>{missing ? SNMP_COPY.notAvailable : SNMP_COPY.loading}</p>
      {error && !missing && <p className="explorer-error">{error}</p>}</div></div></>;
  }
  const refresh = () => { reload(); liveFeed.refresh(SNMP_STATUS); };
  return <>
    {header(switcher)}
    {error && <p className="explorer-error" role="alert">{error}</p>}
    {role === 'simulate'
      ? <SimulateView status={status} config={config} refresh={refresh} fail={setError} />
      : <ReadView status={status} config={config} refresh={refresh} fail={setError} />}
  </>;
}
