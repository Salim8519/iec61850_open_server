import { Laptop, Network } from 'lucide-react';
import type { Ied, NetworkSettings } from '../../api/types';
import { endpointOf, onLabNetwork } from '../network/endpoints';

/** One plain line under the device header: who publishes, and who can reach this IED (change 0098).
 *  "Be this IED" answers on this PC only until the IED is put on a network adapter. */
export function RoleLine({ ied, network, openNetwork }: {
  ied: Ied; network: NetworkSettings; openNetwork?: () => void;
}) {
  if (ied.role === 'off') return <p className="role-line small muted">Left out: not part of Run.</p>;
  if (ied.role === 'real') return <p className="role-line small">
    The real IED publishes. ScadaBench reads it at <code>{ied.ip || 'no address'}:{ied.port}</code>.
    Nothing to set up on this PC.</p>;
  const { host, port } = endpointOf(ied, network);
  const onNetwork = onLabNetwork(ied, network);
  return <p className="role-line small">
    {onNetwork ? <Network size={14} aria-hidden /> : <Laptop size={14} aria-hidden />}
    <span>ScadaBench publishes as this IED at <code>{host}:{port}</code>{' · '}
      {onNetwork ? <>reachable from the network on “{network.adapter_label}”</> : <>reachable from this PC only</>}</span>
    {!onNetwork && openNetwork && <button className="link-button" onClick={openNetwork}>Put on a network</button>}
  </p>;
}
