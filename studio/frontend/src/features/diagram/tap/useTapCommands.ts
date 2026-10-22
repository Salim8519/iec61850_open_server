import { useCallback, useState } from 'react';
import { sendTapCommand, tapAnswerText, type TapAction } from '../../../equipment/tapCommands';
import { markClick } from '../../../app/perf';

/** Sends tap changer commands for the diagram page (ADR-0109): one at a time per page, the answer in words
 *  for the sheet. Errors appear like every other operate error (studio.perform). */
export function useTapCommands(perform: (task: () => Promise<unknown>) => Promise<void>,
  report: (nodeId: string, message: string) => void) {
  const [busy, setBusy] = useState<string | null>(null);
  const send = useCallback((nodeId: string, action: TapAction) => {
    markClick(nodeId);
    setBusy(nodeId);
    void perform(async () => {
      try {
        report(nodeId, tapAnswerText(await sendTapCommand(nodeId, action)));
      } finally {
        setBusy(null);
      }
    });
  }, [perform, report]);
  return { send, busy };
}
