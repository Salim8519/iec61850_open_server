import { Cable, MonitorSmartphone, RefreshCw, Wifi } from 'lucide-react';
import type { Adapter } from '../../api/network';
import { Hint } from '../../ui/Hint';
import { InfoTip } from '../../ui/InfoTip';
import { StatusPill } from '../../ui/StatusPill';
import { UI_STATUS } from '../../copy/status';

const KIND_ICON: Record<string, typeof Cable> = { ethernet: Cable, wifi: Wifi, virtual: MonitorSmartphone,
  virtual_lab: MonitorSmartphone };
const KIND_WORD: Record<string, string> = { ethernet: 'Ethernet port', wifi: 'Wi-Fi', virtual: 'Virtual adapter',
  virtual_lab: 'Windows loopback adapter', vpn: 'VPN adapter', bluetooth: 'Bluetooth' };

export function AdapterStep({ adapters, loading, selected, choose, reload }: {
  adapters: Adapter[] | null; loading: boolean; selected: string; choose: (guid: string) => void; reload: () => void;
}) {
  const usable = (adapters || []).filter(item => !['vpn', 'bluetooth'].includes(item.kind));
  return <div className="stack">
    <div className="row" style={{ justifyContent: 'space-between' }}>
      <h2 style={{ margin: 0 }}>Which <InfoTip term="adapter" label="network adapter" /> are the lab devices on?</h2>
      <button onClick={reload} disabled={loading}><RefreshCw size={14} />Refresh now</button>
    </div>
    <Hint id="network.adapter.pick" />
    <Hint id="network.adapter.virtual" compact />
    {loading && !adapters && <p className="muted small">Asking Windows for the adapter list…</p>}
    {!loading && adapters && !usable.length && <p className="muted">No suitable adapter found. Plug in a USB Ethernet
      adapter or enable a virtual adapter; this list updates by itself.</p>}
    <div className="choice-grid">{usable.map(adapter => {
      const Icon = KIND_ICON[adapter.kind] ?? Cable;
      const up = adapter.status === 'Up';
      return <button key={adapter.guid} type="button" className={`choice${selected === adapter.guid ? ' selected' : ''}`}
        onClick={() => choose(adapter.guid)} aria-pressed={selected === adapter.guid}>
        <span className="row"><Icon size={16} /><strong>{adapter.name}</strong></span>
        <span className="small muted">{KIND_WORD[adapter.kind] ?? adapter.kind} · {adapter.description}</span>
        <span className="row">
          <StatusPill status={up ? UI_STATUS.adapter_up : UI_STATUS.adapter_down} />
          {adapter.default_route && <StatusPill status={UI_STATUS.adapter_internet} />}
        </span>
        <span className="small"><code>{adapter.addresses.map(item => `${item.ip}/${item.prefix}`).join(', ') || 'No IPv4 address'}</code></span>
        {adapter.default_route && <Hint id="network.adapter.default_route" compact />}
      </button>;
    })}</div>
  </div>;
}
