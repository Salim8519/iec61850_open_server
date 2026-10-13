import { useState } from 'react';
import { X } from 'lucide-react';
import type { Project } from '../api/types';

export function RecoveryNotice({ recovery }: { recovery: Project['startup_recovery'] }) {
  const [dismissed, setDismissed] = useState(false);
  if (!recovery || dismissed) return null;
  return <div className={recovery.status === 'unreadable' ? 'error-banner' : 'notice'}
    role={recovery.status === 'unreadable' ? 'alert' : 'status'}>
    <div><span>{recovery.message}</span>
      {recovery.backup && <details><summary>Recovery file and diagnostic</summary>
        <p className="small">{recovery.details}</p><code>{recovery.backup}</code>
      </details>}</div>
    <button className="ghost" aria-label="Dismiss recovery notice" onClick={() => setDismissed(true)}><X size={16} /></button>
  </div>;
}
