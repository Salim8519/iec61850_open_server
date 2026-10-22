import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react';
import { snmpApi, type SnmpDevice } from '../../../../api/snmp';
import { NOT_SAVED } from '../../../../state/autoSave';
import { useAutoSave } from '../../../../state/useAutoSave';
import { deviceProblem, pickSettings, type DeviceSettings } from './deviceRules';

/** The settings and trap receivers of one device, saved as they change ("save always").
 *
 *  The draft lives with the device panel, so a tab switch keeps what was typed. A change is sent at once
 *  when it is valid; an invalid one stays in its field with a red line and is not sent. While the devices
 *  run every save restarts them, so typed text is sent when the field is left (or Enter) instead of on
 *  every key; choices and switches still go at once, and leaving the page sends what was typed. */
export function useDeviceDraft(device: SnmpDevice, running: boolean, refresh: () => void) {
  const [draft, setDraft] = useState<DeviceSettings>(() => pickSettings(device));
  const current = useRef(draft);
  const latest = useRef(device);
  latest.current = device;
  const typed = useRef(false);
  // Only the fields of this page go out; the values and everything else stay as the engine has them.
  const save = useAutoSave<DeviceSettings>(settings => snmpApi.update({ ...latest.current, ...settings }).then(() => refresh()));
  const send = (settings: DeviceSettings) => { if (!deviceProblem(settings)) save.push(settings); };
  const set = (part: Partial<DeviceSettings>, text = false) => {
    current.current = { ...current.current, ...part };
    setDraft(current.current);
    typed.current = text && running;
    if (!typed.current) send(current.current);
  };
  const commit = () => { if (typed.current) { typed.current = false; send(current.current); } };
  const commitNow = useRef(commit);
  commitNow.current = commit;
  useEffect(() => { if (!running) commit(); }, [running]);
  useEffect(() => () => commitNow.current(), []);
  const problem = useMemo(() => deviceProblem(draft), [draft]);
  const edge = { onBlur: commit, onKeyDown: (event: KeyboardEvent) => { if (event.key === 'Enter') commit(); } };
  return { draft, set, problem, state: problem ? NOT_SAVED : save.state, edge,
    /** The engine's own answer when it refused a value these rules do not know. */
    refused: problem || save.state.phase !== 'failed' ? '' : save.state.message };
}

export type DeviceDraft = ReturnType<typeof useDeviceDraft>;
