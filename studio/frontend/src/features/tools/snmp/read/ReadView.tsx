import { useState } from 'react';
import type { SnmpConfig, SnmpStatus } from '../../../../api/snmp';
import { TabBar } from '../../../../ui/TabBar';
import { SNMP_FULL_PLACE } from '../../../../copy/fullView';
import { FullView, FullViewButton } from '../../../../ui/FullView';
import { useFullView } from '../../../../state/useFullView';
import { ConnectForm } from './ConnectForm';
import { MibLibrary } from './MibLibrary';
import { TrapLog } from './TrapLog';
import { ValuesTable } from './ValuesTable';

type Tab = 'values' | 'traps' | 'mibs';
const TABS: [Tab, string][] = [['values', 'Values'], ['traps', 'Traps'], ['mibs', 'MIB library']];

/** Read devices: the device to read on the left; its values, received traps and the MIB library on the right. */
export function ReadView({ status, config, refresh, fail }: {
  status: SnmpStatus; config: SnmpConfig; refresh: () => void; fail: (message: string) => void;
}) {
  const [tab, setTab] = useState<Tab>('values');
  const [full, setFull] = useFullView('snmpRead');
  const connected = status.reader.state === 'connected' || status.reader.state === 'walking';
  return <div className="fill-body fill-split scada-split">
    <aside className="card side-pane snmp-side" aria-label="Device to read">
      <ConnectForm status={status} config={config} refresh={refresh} fail={fail} />
    </aside>
    <FullView full={full} change={setFull} place={SNMP_FULL_PLACE.read}>
    <div className={`card fill-panel gateway-panel${connected ? ' running' : ''}`}>
      <TabBar label="Read views" value={tab} change={setTab} tabs={TABS} end={<FullViewButton full={full} change={setFull} />} />
      <div className="panel-body">
        {tab === 'values' && <ValuesTable status={status} config={config} refresh={refresh} fail={fail} />}
        {tab === 'traps' && <TrapLog status={status} config={config} refresh={refresh} fail={fail} />}
        {tab === 'mibs' && <MibLibrary fail={fail} />}
      </div>
    </div>
    </FullView>
  </div>;
}
