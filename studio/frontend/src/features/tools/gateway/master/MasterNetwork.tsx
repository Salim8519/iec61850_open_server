import { useEffect, useState } from 'react';
import { masterApi, type MasterNetwork as Network } from '../../../../api/master';
import { NETWORK_COPY } from '../../../../copy/master';
import { HelpTip } from '../../../../ui/HelpTip';

/** "Network" (ADR-0123): All networks by default (Windows chooses), or one address of this PC to connect
 *  from. The list, the check and the address the device sees all come from the Go engine. */
export function MasterNetwork({ host, locked }: { host: string; locked: boolean }) {
  const [network, setNetwork] = useState<Network | null>(null);
  const [error, setError] = useState('');
  useEffect(() => { masterApi.network().then(setNetwork).catch(() => setNetwork(null)); }, [host, locked]);
  if (!network) return null;
  const choose = (local: string) => {
    setError('');
    masterApi.saveNetwork(local).then(setNetwork).catch((problem: Error) => setError(problem.message));
  };
  const missing = network.local && !network.choices.some(item => item.address === network.local);
  // The closed field shows the address only (never clipped in the narrow column); the list groups addresses by card.
  const cards = [...new Set(network.choices.map(item => item.adapter))];
  return <>
    <label><span className="label-row">{NETWORK_COPY.label}<HelpTip text={NETWORK_COPY.hint} label="Which network?" /></span>
      <select value={network.local} disabled={locked} onChange={event => choose(event.target.value)}>
        <option value="">{NETWORK_COPY.all}</option>
        {cards.map(card => <optgroup key={card} label={card}>
          {network.choices.filter(item => item.adapter === card).map(item =>
            <option key={item.address} value={item.address}>{item.address}</option>)}</optgroup>)}
        {missing && <option value={network.local}>{network.local}</option>}
      </select></label>
    {network.source && <p className="side-line">{NETWORK_COPY.sees(network.source, network.source_adapter)}</p>}
    {network.note && <p className="side-error" role="status">{network.note}</p>}
    {error && <p className="side-error" role="alert">{error}</p>}
  </>;
}
