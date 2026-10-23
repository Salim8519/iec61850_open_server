import type { Ied, Signal, SignalBinding } from '../../api/types';
import { SignalSearchPicker } from '../../components/SignalSearchPicker';

type Props = {
  ied: Ied; existing: Set<string>; disabled?: boolean;
  /** The relay's name: points that mention it are suggested first (ADR-0113). */
  near?: string;
  onAdd: (binding: SignalBinding, signal: Signal) => void;
};

/** Relay-scoped one-click signal adding; search and choice live in the shared picker. */
export function RelaySignalAdder({ ied, existing, disabled, near, onAdd }: Props) {
  return <SignalSearchPicker ieds={[ied]} fixedIedId={ied.id} near={near}
    exclude={existing} disabled={disabled} chooseLabel="Add" memoryKey={`sld-relay-${ied.id}`} choose={onAdd} />;
}
