import { useEffect, useRef, useState } from 'react';
import { RotateCcw } from 'lucide-react';
import type { EquipmentData } from '../../api/types';
import type { ReactNode } from 'react';
import { colorScheme, defaultScheme, paintFor, validHex, type Paint } from '../../equipment/stateColors';
import { EquipmentSymbol } from '../../equipment/symbols/EquipmentSymbol';
import { kindInfo } from '../../equipment/kindInfo';
import './appearance.css';

type Props = {
  data: EquipmentData; locked: boolean; update: (data: EquipmentData) => void;
  applyToKind: (active: string | null, inactive: string | null) => void;
};

/** Dragging in the picker previews locally; the native change event commits one edit. */
function ColorRow({ label, value, fallback, disabled, preview, change }: {
  label: string; value: string; fallback: string; disabled: boolean;
  preview: (paint: Paint) => ReactNode; change: (value: string | null) => void;
}) {
  const [draft, setDraft] = useState<string | null>(null);
  const input = useRef<HTMLInputElement>(null);
  const commit = useRef(change);
  commit.current = change;
  useEffect(() => {
    const element = input.current;
    if (!element) return;
    const done = () => {
      setDraft(null);
      commit.current(element.value.toLowerCase() === fallback.toLowerCase() ? null : element.value);
    };
    element.addEventListener('change', done);
    return () => element.removeEventListener('change', done);
  }, [fallback]);
  useEffect(() => { setDraft(null); }, [value]);
  const shown = draft || value;
  return <div className="appearance-row">
    <span className="appearance-preview">{preview(paintFor(shown))}<strong>{label}</strong></span>
    <input ref={input} type="color" aria-label={`${label} colour`} value={shown.toLowerCase()} disabled={disabled}
      onChange={event => setDraft(event.target.value)} />
    <code>{shown.toUpperCase()}</code>
  </div>;
}

/** Per-object state colours with defaults, reset and copy to every object of the same kind. */
export function AppearanceEditor({ data, locked, update, applyToKind }: Props) {
  const scheme = colorScheme(data);
  const defaults = defaultScheme(data);
  if (!scheme || !defaults) return null;
  const custom = !!validHex(data.active_color) || !!validHex(data.inactive_color);
  const plural = `${kindInfo(data.kind).name.toLowerCase()}s`;
  const symbol = (active: boolean) => (paint: Paint) => <EquipmentSymbol kind={data.kind} size={36} paint={paint}
    position={active ? 'Closed' : 'Open'} operated={active} />;
  return <div className="appearance">
    <ColorRow label={scheme.activeLabel} value={scheme.active} fallback={defaults.active} disabled={locked}
      preview={symbol(true)} change={value => update({ ...data, active_color: value })} />
    <ColorRow label={scheme.inactiveLabel} value={scheme.inactive} fallback={defaults.inactive} disabled={locked}
      preview={symbol(false)} change={value => update({ ...data, inactive_color: value })} />
    {!locked && <div className="appearance-actions">
      <button className="ghost" disabled={!custom}
        onClick={() => update({ ...data, active_color: null, inactive_color: null })}>
        <RotateCcw size={14} />Reset to defaults</button>
      <button onClick={() => applyToKind(validHex(data.active_color), validHex(data.inactive_color))}>
        Apply to all {plural}</button>
    </div>}
  </div>;
}
