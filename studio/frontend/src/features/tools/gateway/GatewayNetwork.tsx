import { useEffect, useState } from 'react';
import { gatewayApi, type GatewayNetwork as Network } from '../../../api/gateway';
import { GATEWAY_NETWORK_COPY as COPY } from '../../../copy/gateway';
import { HelpTip } from '../../../ui/HelpTip';
import { TYPE_ADDRESS, TypeAddress } from './TypeAddress';

/** "Network" of the station gateway (ADR-0124): This PC only, All networks, one card of this PC or its own address
 *  on the ScadaBench Lab card (ADR-0125), and the address SCADA connects to. The list, checks and address come from
 *  the Go engine; this only shows them. */
export function GatewayNetwork({ locked, protocol, port, saved }: {
  locked: boolean; protocol: string; port: unknown; saved?: () => void;
}) {
  const [network, setNetwork] = useState<Network | null>(null);
  const [error, setError] = useState('');
  const [typing, setTyping] = useState(false);
  useEffect(() => { gatewayApi.network().then(setNetwork).catch(() => setNetwork(null)); }, [locked, protocol, port]);
  if (!network) return null;
  const send = (choice: string, allowed: string) => {
    setError('');
    gatewayApi.saveNetwork(choice, allowed).then(next => { setNetwork(next); saved?.(); })
      .catch((problem: Error) => setError(problem.message));
  };
  const add = (address: string) => gatewayApi.addOwnAddress(address)
    .then(next => { setNetwork(next); setTyping(false); saved?.(); });
  const pick = (choice: string) => {
    setTyping(choice === TYPE_ADDRESS);
    if (choice !== TYPE_ADDRESS) send(choice, network.allowed);
  };
  const cards = [...new Set(network.choices.map(item => item.adapter))];
  const missing = !['this_pc', 'network'].includes(network.choice) && !network.own_addresses.includes(network.choice)
    && !network.choices.some(item => item.address === network.choice);
  return <>
    <label><span className="label-row">{COPY.label}<HelpTip text={COPY.hint} label="Which network?" /></span>
      <select value={typing ? TYPE_ADDRESS : network.choice} disabled={locked} onChange={event => pick(event.target.value)}>
        <option value="this_pc">{COPY.thisPc}</option><option value="network">{COPY.all}</option>
        {cards.map(card => <optgroup key={card} label={card}>
          {network.choices.filter(item => item.adapter === card).map(item =>
            <option key={item.address} value={item.address}>{item.address}</option>)}</optgroup>)}
        {network.own_addresses.length > 0 && <optgroup label={COPY.own}>
          {network.own_addresses.map(address => <option key={address} value={address}>{address}</option>)}</optgroup>}
        {missing && <option value={network.choice}>{network.choice}</option>}
        <option value={TYPE_ADDRESS}>{COPY.type}</option>
      </select></label>
    {typing && <TypeAddress labReady={network.lab_ready} placeholder={COPY.ownPlaceholder} add={add} />}
    {!typing && network.own && <p className="side-line">{COPY.ownHint}</p>}
    <p className="side-line">{COPY.connectTo} <strong style={{ whiteSpace: 'nowrap' }}>{network.connect_to}:{String(network.port)}</strong>
      {network.connect_adapter && ` (${network.connect_adapter})`}{network.others > 0 && `, ${COPY.more(network.others)}`}.</p>
    {network.note && <p className="side-error" role="status">{network.note}</p>}
    {error && <p className="side-error" role="alert">{error}</p>}
  </>;
}

/** The optional allowed SCADA address, inside Advanced (IEC 104 and Modbus TCP; DNP3 checks the master address). */
export function AllowedScada({ locked }: { locked: boolean }) {
  const [network, setNetwork] = useState<Network | null>(null);
  const [error, setError] = useState('');
  useEffect(() => { gatewayApi.network().then(setNetwork).catch(() => setNetwork(null)); }, [locked]);
  if (!network) return null;
  const save = (allowed: string) => {
    setError('');
    if (allowed !== network.allowed) gatewayApi.saveNetwork(network.choice, allowed).then(setNetwork)
      .catch((problem: Error) => setError(problem.message));
  };
  return <>
    <label><span className="label-row">{COPY.allowed}<HelpTip text={COPY.allowedHint} label="Who may connect?" /></span>
      <input defaultValue={network.allowed} key={`allowed-${network.allowed}`} placeholder={COPY.allowedPlaceholder}
        disabled={locked || !network.allowed_works} onBlur={event => save(event.target.value.trim())} /></label>
    {!network.allowed_works && <p className="side-line">{COPY.allowedDnp3}</p>}
    {error && <p className="side-error" role="alert">{error}</p>}
  </>;
}
