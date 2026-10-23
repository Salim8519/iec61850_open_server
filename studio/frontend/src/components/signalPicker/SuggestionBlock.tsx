import { Lightbulb } from 'lucide-react';
import type { ReactNode } from 'react';
import { PICKER_COPY } from '../../copy/signalPicker';
import type { PickedSignal } from './pickerAnswer';

/** Points the engine found that mention the component's name in the IED's own words (ADR-0113),
 *  listed before the search results. Each row says where the name was found. */
export function SuggestionBlock({ name, items, row }: {
  name: string; items: PickedSignal[]; row: (signal: PickedSignal, note: string) => ReactNode;
}) {
  if (!items.length) return null;
  return <section className="signal-suggested" aria-label={PICKER_COPY.suggested(name)}>
    <h5 title={PICKER_COPY.suggestedHint}><Lightbulb size={14} aria-hidden />{PICKER_COPY.suggested(name)}</h5>
    {items.map(signal => row(signal, PICKER_COPY.mentions(name, signal.suggested_by ?? 'reference')))}
  </section>;
}
