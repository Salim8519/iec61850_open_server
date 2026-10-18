import type { Project } from '../api/types';
import { CompactNotice } from './CompactNotice';

/** Startup recovery in one line; an unreadable autosave is a problem and keeps its diagnostic and backup path. */
export function RecoveryNotice({ recovery }: { recovery: Project['startup_recovery'] }) {
  if (!recovery) return null;
  return <CompactNotice message={recovery.message} problem={recovery.status === 'unreadable'}
    detailsLabel="Recovery file and diagnostic" dismissLabel="Dismiss recovery notice">
    {recovery.backup ? <><p className="small" style={{ margin: 0 }}>{recovery.details}</p><code>{recovery.backup}</code></> : undefined}
  </CompactNotice>;
}
