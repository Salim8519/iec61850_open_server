import { useState } from 'react';
import { Trash2 } from 'lucide-react';
import { DeleteIedsDialog } from './DeleteIedsDialog';

/** Delete one IED: a labelled button (Files tab) or a compact one in the device header. */
export function DeleteIed({ iedId, iedName, disabled, perform, prepare, compact = false }: {
  iedId: string; iedName?: string; disabled: boolean; compact?: boolean;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const reason = disabled ? 'Stop the run to delete IEDs' : `Delete ${iedName ?? 'this IED'} from the project`;
  return <>
    <button className={compact ? 'ghost danger-text' : 'danger-outline'} disabled={disabled}
      title={reason} aria-label={compact ? reason : undefined} onClick={() => setOpen(true)}>
      <Trash2 size={compact ? 18 : 14} />{compact ? 'Delete' : 'Delete IED'}</button>
    {open && <DeleteIedsDialog ids={[iedId]} close={() => setOpen(false)} perform={perform} prepare={prepare} />}
  </>;
}
