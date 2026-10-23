import { useEffect, useState } from 'react';
import { messagesPath, type IedMessages } from '../../../api/gooseMessages';
import { liveFeed } from '../../../api/polls';

/** One IED's GOOSE messages from Go (ADR-0112), asked again every 2 s while `poll` (the run). */
export function useIedMessages(iedId: string | null | undefined, poll: boolean) {
  const [answer, setAnswer] = useState<IedMessages | null>(null);
  const [error, setError] = useState('');
  useEffect(() => {
    if (!iedId) return undefined;
    return liveFeed.subscribe<IedMessages>(messagesPath(iedId), poll ? 2000 : null,
      value => { setAnswer(value); setError(''); }, reason => setError(reason.message));
  }, [iedId, poll]);
  // An answer for another IED (the choice just changed) is never shown.
  const current = answer && answer.ied_id === iedId ? answer : null;
  return { answer: current, error: current ? '' : error, refresh: () => { if (iedId) liveFeed.refresh(messagesPath(iedId)); } };
}
