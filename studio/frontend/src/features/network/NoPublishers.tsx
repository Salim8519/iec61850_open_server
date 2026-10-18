import { Radio } from 'lucide-react';
import { api } from '../../api/client';
import type { Ied } from '../../api/types';

const ROLE_WORD: Record<string, string> = { real: 'Read real IED', off: 'Leave out' };

/** Shown instead of an error when no IED is set to "Be this IED": names each IED and offers the fix. */
export function NoPublishers({ ieds, locked, perform }: {
  ieds: Ied[]; locked: boolean; perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const publishAll = () => perform(() => Promise.all(ieds.map(ied =>
    api(`/ieds/${ied.id}/settings`, 'PUT', { role: 'simulated' }))));
  return <div className="stack">
    <div className="warning-banner" role="status">
      <span>No IED is set to <strong>Be this IED</strong>, so there is nothing to put on the network.</span>
    </div>
    {!!ieds.length && <p className="small" style={{ margin: 0 }}>
      {ieds.map(ied => `${ied.name}: ${ROLE_WORD[ied.role] ?? ied.role}`).join(' · ')}</p>}
    <p className="small muted" style={{ margin: 0 }}>
      "Read real IED" needs no setup here: ScadaBench only connects out to the real device.</p>
    {!!ieds.length && <div className="row">
      <button className="primary" disabled={locked} onClick={publishAll}><Radio size={16} />
        Make {ieds.length === 1 ? ieds[0].name : `all ${ieds.length} IEDs`} publish</button>
      <span className="small muted">or choose per IED in Devices</span>
    </div>}
  </div>;
}
