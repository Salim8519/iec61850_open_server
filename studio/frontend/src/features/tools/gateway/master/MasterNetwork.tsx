import { useEffect, useState } from 'react';
import { masterApi, type MasterNetwork as Network } from '../../../../api/master';
import { GATEWAY_NETWORK_COPY } from '../../../../copy/gateway';
import { NETWORK_COPY } from '../../../../copy/master';
import { HelpTip } from '../../../../ui/HelpTip';
import { TYPE_ADDRESS, TypeAddress } from '../TypeAddress';

/** "Network" (ADR-0123): All networks by default (Windows chooses), one address of this PC to connect from, or
 *  the master's own address on the ScadaBench Lab card (ADR-0125). The list, the check and the address the device
 *  sees all come from the Go engine. */
export function MasterNetwork({ host, locked }: { host: string; locked: boolean }) {
  const [network, setNetwork] = useState<Network | null>(null);
  const [error, setError] = useState('');
  const [typing, setTyping] = useState(false);
  useEffect(() => { masterApi.network().then(setNetwork).catch(() => setNetwork(null)); }, [host, locked]);
  if (!network) return null;
  const choose = (local: string) => {
    setError('');
    setTyping(local === TYPE_ADDRESS);
    if (local !== TYPE_ADDRESS) masterApi.saveNetwork(local).then(setNetwork).catch((problem: Error) => setError(problem.message));
  };
  const add = (address: string) => masterApi.addOwnAddress(address).then(next => { setNetwork(next); setTyping(false); });
  const missing = network.local && !network.own_addresses.includes(network.local)
    && !network.choices.some(item => item.address === network.local);
  // The closed field shows the address only (never clipped in the narrow column); the list groups addresses by card.
  const cards = [...new Set(network.choices.map(item => item.adapter))];
  return <>
    <label><span className="label-row">{NETWORK_COPY.label}<HelpTip text={NETWORK_COPY.hint} label="Which network?" /></span>
      <select value={typing ? TYPE_ADDRESS : network.local} disabled={locked} onChange={event => choose(event.target.value)}>
        <option value="">{NETWORK_COPY.all}</option>
        {cards.map(card => <optgroup key={card} label={card}>
          {network.choices.filter(item => item.adapter === card).map(item =>
            <option key={item.address} value={item.address}>{item.address}</option>)}</optgroup>)}
        {network.own_addresses.length > 0 && <optgroup label={GATEWAY_NETWORK_COPY.own}>
          {network.own_addresses.map(address => <option key={address} value={address}>{address}</option>)}</optgroup>}
        {missing && <option value={network.local}>{network.local}</option>}
        <option value={TYPE_ADDRESS}>{GATEWAY_NETWORK_COPY.type}</option>
      </select></label>
    {typing && <TypeAddress labReady={network.lab_ready} placeholder={NETWORK_COPY.ownPlaceholder} add={add} />}
    {!typing && network.own && <p className="side-line">{NETWORK_COPY.ownHint}</p>}
    {network.source && <p className="side-line">{NETWORK_COPY.sees(network.source, network.source_adapter)}</p>}
    {network.note && <p className="side-error" role="status">{network.note}</p>}
    {error && <p className="side-error" role="alert">{error}</p>}
  </>;
}
