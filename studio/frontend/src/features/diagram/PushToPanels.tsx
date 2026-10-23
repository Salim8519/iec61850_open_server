import { ListChecks } from 'lucide-react';

/** "Put push buttons into point panels" (change 0180): one point panel per IED instead of many buttons. */
export function PushToPanels({ count, convert }: { count: number; convert: () => void }) {
  if (count < 1) return null;
  return <section className="sheet-section"><h3>Tidier as a list</h3>
    <p className="small muted" style={{ margin: '0 0 8px' }}>
      Puts {count === 1 ? 'this push button' : `all ${count} push buttons`} into point panels, one panel per IED. Status
      buttons become ON/OFF rows and command buttons become command rows, with the same signals. Undo brings the
      buttons back.</p>
    <button type="button" onClick={convert}><ListChecks size={16} />Put push buttons into point panels</button>
  </section>;
}
