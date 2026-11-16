import type { LiveSettings } from '../../api/types';

/** Allow commands of a real IED (ADR-0138). It is stored only while on, so turning it off again
 *  gives back exactly the settings the IED had and Save stays off. */
export function withCommands(live: LiveSettings, allow: boolean): LiveSettings {
  const { allow_commands: _dropped, ...rest } = live;
  return allow ? { ...rest, allow_commands: true } : rest;
}

/** Whether Devices may offer Send buttons for this real IED: the engine still checks every command. */
export function commandsState(live: LiveSettings, connected: boolean): 'off' | 'not_live' | 'ready' {
  if (!live.allow_commands) return 'off';
  return connected ? 'ready' : 'not_live';
}

/** The two buttons of a command: Close/Open for a position (DPC), On/Off for a single command (SPC). */
export function commandWords(cdc: string): [string, string] {
  return cdc === 'DPC' ? ['Close', 'Open'] : ['On', 'Off'];
}
