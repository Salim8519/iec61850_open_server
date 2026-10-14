import { TriangleAlert } from 'lucide-react';
import type { SwitchHints as Hints } from '../../equipment/energization';
import { Hint } from '../../ui/Hint';

const ACTION = { open: 'Before you open', close: 'Before you close', both: 'Before you open or close' } as const;

/** Friendly switching advice from the drawing; it never disables Open or Close. */
export function SwitchHints({ hints }: { hints?: Hints }) {
  if (!hints) return null;
  const same = JSON.stringify(hints.open) === JSON.stringify(hints.close);
  const groups = (same ? [['both', hints.open]] : [['open', hints.open], ['close', hints.close]]) as
    [keyof typeof ACTION, Hints['open']][];
  const shown = groups.filter(([, items]) => items.length);
  if (!shown.length) return null;
  return <div className="switch-hints" role="note">
    {shown.map(([action, items]) => <div key={action}><strong><TriangleAlert size={15} aria-hidden="true" /> {ACTION[action]}</strong>
      <ul>{items.map(item => <li key={item.code + item.text}>{item.text}</li>)}</ul></div>)}
    <Hint id="switch.hints" compact />
  </div>;
}
