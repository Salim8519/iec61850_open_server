import { Eraser } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { PlanItem } from '../../api/network';
import { CompactNotice } from '../../components/CompactNotice';

type Leftovers = { items: PlanItem[]; count: number; fingerprint: string };

/** Addresses ScadaBench left on a network card (an older plan, or a run without the network
 *  helper), asked about when the app opens and after Stop (ADR-0136). The engine decides what is
 *  left over; this only shows the list and sends the confirmed one back. */
export function LeftoverNotice({ running, perform }: {
  running: boolean; perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const [leftovers, setLeftovers] = useState<Leftovers | null>(null);
  const [checked, setChecked] = useState(0);
  const [working, setWorking] = useState(false);  // waiting for the Windows (UAC) prompt
  useEffect(() => {
    if (running) return;
    let current = true;
    api<Leftovers>('/network/leftovers').then(found => { if (current) setLeftovers(found); })
      .catch(() => { if (current) setLeftovers(null); });
    return () => { current = false; };
  }, [running, checked]);
  if (running || !leftovers?.count) return null;
  const count = leftovers.count;
  const message = `ScadaBench left ${count} address${count === 1 ? '' : 'es'} on your network from an earlier run. ` +
    'Remove them? Windows asks once.';
  const remove = () => {
    setWorking(true);
    perform(() => api('/network/leftovers/remove', 'POST', { fingerprint: leftovers.fingerprint, confirm: true }))
      .finally(() => { setWorking(false); setChecked(value => value + 1); });
  };
  return <CompactNotice key={leftovers.fingerprint} message={message} problem detailsLabel="Show addresses"
    dismissLabel="Keep the addresses for now"
    action={<button disabled={working} onClick={remove}><Eraser size={14} />{working ? 'Waiting for Windows…' : 'Remove them'}</button>}>
    <ul className="small" style={{ margin: 0, paddingLeft: 18 }}>
      {leftovers.items.map((item, index) => <li key={index}>{item.description}</li>)}
    </ul>
  </CompactNotice>;
}
