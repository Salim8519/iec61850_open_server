import { useState } from 'react';
import { X } from 'lucide-react';
import type { Project } from '../api/types';

export function MigrationNotice({ notice }: { notice: Project['migration_notice'] }) {
  const [dismissed, setDismissed] = useState(false);
  if (!notice || dismissed) return null;
  return <div className="recovery-notice" role="status"><div><span>{notice.message}</span>
    <details><summary>Migration details and recovery file</summary>
      <ul>{notice.details.map(item => <li key={item}>{item}</li>)}</ul>
      <code>{notice.backup}</code>
    </details></div>
    <button className="ghost" aria-label="Dismiss migration notice" onClick={() => setDismissed(true)}><X size={16} /></button>
  </div>;
}
