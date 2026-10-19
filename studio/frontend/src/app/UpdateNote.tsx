import { Download, X } from 'lucide-react';
import type { UpdateStatus } from '../api/updates';

/** One quiet line above Settings when a newer ScadaBench exists (ADR-0071). No popup, no sound,
 *  no badge on other places; × hides it until the next version. An important update keeps the line. */
export function UpdateNote({ status, dismiss }: { status: UpdateStatus | null; dismiss: (version: string) => void }) {
  if (!status?.available || status.dismissed || !status.latest) return null;
  const { version, page } = status.latest;
  // Short enough for one sidebar line; the full sentence is in the hover text and Settings › About.
  const words = status.important ? `Update ${version} · important` : `Update ${version}`;
  return <div className={`update-note${status.important ? ' important' : ''}`}>
    <a className="nav-item" href={page} target="_blank" rel="noreferrer"
      title={`ScadaBench ${version} is ready${status.important ? ' (important)' : ''}. You have ${status.current}. Opens the download page; nothing is installed by itself.`}>
      <Download size={18} aria-hidden /><span>{words}</span></a>
    {!status.important && <button className="ghost icon update-note-close" aria-label={`Hide the note about version ${version}`}
      title="Hide until the next version" onClick={() => dismiss(version)}><X size={14} /></button>}
  </div>;
}
