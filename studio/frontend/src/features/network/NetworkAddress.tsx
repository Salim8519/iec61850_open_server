import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import { EditableText } from '../../ui/EditableText';

/** The address an IED or SNMP device answers at on the project's network, changed in place
 *  (change 0215) instead of in Settings › Network. The engine decides how: on ScadaBench Lab it
 *  checks the address and uses it at once; on a card such as Ethernet Windows approves it once.
 *  Its answer is shown under the address; a refusal comes back as the page's error. */
export function NetworkAddress({ kind, id, address, port, perform, size }: {
  kind: 'ied' | 'snmp'; id: string; address: string; port?: number; size?: number;
  perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const [answer, setAnswer] = useState('');
  useEffect(() => setAnswer(''), [id]);
  const save = (text: string) => {
    if (!text.trim()) return;
    setAnswer('');
    perform(async () => {
      const result = await api<{ message: string }>(`/network/address/${kind}/${id}`, 'PUT', { ip: text.trim() });
      setAnswer(result.message);
    });
  };
  return <span style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
    <span className="row" style={{ gap: 2, fontSize: size }}>
      <EditableText value={address} onSave={save} label="Address on the network" placeholder="Type an address" mono={!size} />
      {port !== undefined && <code>:{port}</code>}
    </span>
    <span className="small muted">{answer || 'Click the address to change it. ScadaBench checks it first; on a card such as Ethernet, Windows asks once.'}</span>
  </span>;
}
