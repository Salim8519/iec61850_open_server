import { useEffect, useState } from 'react';
import { masterApi } from '../../api/master';

type Choice = { address: string; adapter: string };

/** The network card ScadaBench reads this relay through (change 0233): "Windows chooses" by default, or
 *  one address of this PC. Monitoring never depends on how the simulated IEDs are set up; the list is the
 *  PC's own card addresses, the same the SCADA master offers (ADR-0123). */
export function MonitorCard({ value, locked, change }: { value: string; locked: boolean; change: (local: string) => void }) {
  const [choices, setChoices] = useState<Choice[]>([]);
  useEffect(() => { masterApi.network().then(network => setChoices(network.choices)).catch(() => setChoices([])); }, []);
  const cards = [...new Set(choices.map(item => item.adapter))];
  const missing = value && !choices.some(item => item.address === value);
  return <label>Network card
    <select value={value} disabled={locked} onChange={event => change(event.target.value)}>
      <option value="">Windows chooses</option>
      {cards.map(card => <optgroup key={card} label={card}>
        {choices.filter(item => item.adapter === card).map(item =>
          <option key={item.address} value={item.address}>{item.address}</option>)}</optgroup>)}
      {missing && <option value={value}>{value} (not on this PC now)</option>}
    </select>
    <span className="small muted" style={{ fontWeight: 400 }}>{value ? `Connects from ${value}` : 'The card Windows routes the device address through'}</span>
  </label>;
}
