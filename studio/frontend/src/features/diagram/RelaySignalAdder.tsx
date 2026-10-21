import type { Ied, Signal, SignalBinding } from '../../api/types';
import { SignalSearchPicker } from '../../components/SignalSearchPicker';

type Props = {
  ied: Ied; existing: Set<string>; disabled?: boolean;
  onAdd: (binding: SignalBinding, signal: Signal) => void;
};

/** Relay-scoped one-click signal adding; search and choice live in the shared picker. */
export function RelaySignalAdder({ ied, existing, disabled, onAdd }: Props) {
  return <SignalSearchPicker ieds={[ied]} fixedIedId={ied.id}
    exclude={existing} disabled={disabled} chooseLabel="Add" memoryKey={`sld-relay-${ied.id}`} choose={onAdd} />;
}
