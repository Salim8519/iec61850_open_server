import { useMemo } from 'react';
import { ACTIVITY } from '../api/polls';
import type { Activity } from '../api/activity';
import { SNMP_STATUS, type SnmpStatus } from '../api/snmp';
import { useLiveFeed } from '../api/useLiveFeed';

const EVERY_MS = 2000;

/** Polls what runs in the tools so every place can show its own running dot. Through the shared
 * live feed: an unchanged answer re-renders nothing, and a hidden window does not poll. */
export function useActivity(enabled: boolean) {
  const activity = useLiveFeed<Activity>(enabled ? ACTIVITY : null, EVERY_MS).value;
  // The SNMP tool's state comes from its own status (Go engine; the Python engine answers 404).
  const snmp = useLiveFeed<SnmpStatus>(enabled ? SNMP_STATUS : null, EVERY_MS).value;
  return useMemo(() => activity && snmp ? { ...activity, snmp: { running: snmp.running, reader: snmp.reader.state,
    target: snmp.reader.target, listening: snmp.traps.listening } } : activity, [activity, snmp]);
}
