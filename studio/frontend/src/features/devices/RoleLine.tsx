import { Laptop, Network } from 'lucide-react';
import type { Ied, NetworkSettings } from '../../api/types';
import { ROLE_COPY } from '../../copy/roles';
import { HelpTip } from '../../ui/HelpTip';
import { endpointOf, onLabNetwork, realEndpointOf } from '../network/endpoints';

/** One plain line under the device header: what the chosen role does, in one sentence, and
 *  where the IED is reached (changes 0098, 0142). Simulate answers on this PC only until the
 *  IED is put on a network adapter. The role sentence sits behind "?" so the line stays one line
 *  and the data points keep their room (ADR-0095). */
export function RoleLine({ ied, network, openNetwork }: {
  ied: Ied; network: NetworkSettings; openNetwork?: () => void;
}) {
  const role = ROLE_COPY[ied.role];
  if (ied.role === 'off') return <p className="role-line small muted"><strong>{role.label}:</strong> {role.sentence}</p>;
  if (ied.role === 'real') {
    const { host, port } = realEndpointOf(ied);
    return <p className="role-line small"><span><strong>{role.label}:</strong> {role.sentence}{' '}
      {host ? <>Device address <code>{host}:{port}</code>; nothing to set up on this PC.</>
        : <>Type the device address below.</>}</span></p>;
  }
  const { host, port } = endpointOf(ied, network);
  const onNetwork = onLabNetwork(ied, network);
  return <p className="role-line small">
    {onNetwork ? <Network size={14} aria-hidden /> : <Laptop size={14} aria-hidden />}
    <span><strong>{role.label}:</strong> Address <code>{host || 'chosen at Run'}:{port}</code>{' · '}
      {onNetwork ? <>reachable from the network on “{network.adapter_label}”</> : <>reachable from this PC only</>}</span>
    <HelpTip text={role.sentence} label={`What ${role.label} does`} />
    {!onNetwork && openNetwork && <button className="link-button" onClick={openNetwork}>Put on a network</button>}
  </p>;
}
