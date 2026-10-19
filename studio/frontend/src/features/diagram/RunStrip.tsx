import { Play } from 'lucide-react';
import { RUN_MARK, RUN_SHAPE_COPY, nameList, type RunSummary } from '../../copy/runShape';

/** The canvas lock badge while only some IEDs run (change 0106): who is live, why the drawing is
 *  locked, and one way to start the rest. A full run keeps the short "Running · drawing is locked". */
export function RunStrip({ summary, runAll, busy }: { summary: RunSummary; runAll: () => void; busy: boolean }) {
  return <span className="run-strip" title={RUN_SHAPE_COPY.strip(summary)}>
    <span className="run-strip-line"><span className="run-mark" aria-hidden>{RUN_MARK.partial}</span>
      {summary.running.length} of {summary.total} IEDs live: {nameList(summary.running, 2)}</span>
    <span className="run-strip-line small">Grey equipment belongs to stopped IEDs · drawing locked
      <button className="link-button" disabled={busy} onClick={runAll}><Play size={12} aria-hidden />Run all</button></span>
  </span>;
}
