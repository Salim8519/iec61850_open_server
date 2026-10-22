// Every status is text + icon + line pattern; colour is only an extra cue.
export type Tone = 'good' | 'info' | 'waiting' | 'warning' | 'problem' | 'off';
export type ToneStyle = { icon: 'check' | 'info' | 'clock' | 'alert' | 'x' | 'minus'; pattern: 'solid' | 'dotted' | 'dashed' | 'double'; word: string };
export type Status = { label: string; tone: Tone; hint: string };

export const TONES: Record<Tone, ToneStyle> = {
  good: { icon: 'check', pattern: 'solid', word: 'OK' },
  info: { icon: 'info', pattern: 'solid', word: 'Note' },
  waiting: { icon: 'clock', pattern: 'dotted', word: 'Waiting' },
  warning: { icon: 'alert', pattern: 'dashed', word: 'Check' },
  problem: { icon: 'x', pattern: 'double', word: 'Problem' },
  off: { icon: 'minus', pattern: 'dashed', word: 'Off' },
};

export const GOOSE_STATES: Record<string, Status> = {
  stopped: { label: 'Not running', tone: 'off', hint: 'Press Run to start sending and listening for GOOSE.' },
  waiting: { label: 'Not received yet', tone: 'waiting', hint: 'ScadaBench is listening. If this stays, check that the publisher runs and is plugged into the same network.' },
  receiving: { label: 'Receiving', tone: 'good', hint: 'GOOSE from this block is arriving on the network.' },
  problem: { label: 'Received with a problem', tone: 'problem', hint: 'Messages arrive but do not match the SCD. See the reason.' },
  timeout: { label: 'Stopped arriving', tone: 'problem', hint: 'Messages arrived before but stopped. The publisher may be off, unplugged or restarted.' },
};

export const PROBLEMS: Record<string, Status> = {
  invalid_frame: { label: 'Unreadable message', tone: 'problem', hint: 'The message could not be decoded. Check the publisher configuration.' },
  conf_rev_mismatch: { label: 'Configuration revision differs', tone: 'problem', hint: 'The publisher runs a different configuration than the SCD. Update the IED or the SCD so both match.' },
  dataset_mismatch: { label: 'Different dataset', tone: 'problem', hint: 'The publisher sends a different dataset than the SCD says. Check the GOOSE block dataset.' },
  go_id_mismatch: { label: 'Different GOOSE ID', tone: 'problem', hint: 'The publisher sends another GOOSE ID than the SCD. Subscribers that check the GOOSE ID will ignore it; load the matching configuration.' },
  needs_commission: { label: 'Publisher needs commissioning', tone: 'warning', hint: 'The publisher says its configuration is not finished. Finish commissioning the publishing IED.' },
  timeout: { label: 'Messages stopped', tone: 'problem', hint: 'No message within the allowed time. Check cables, the publisher and that it is running.' },
  missing_scd_address: { label: 'No address in SCD', tone: 'warning', hint: 'This IED has no IP address in the SCD. Use the suggested free address or type one.' },
  adapter_has_no_network: { label: 'Adapter has no network', tone: 'warning', hint: 'This network card has no usable IPv4 network. Type an address or pick another network card.' },
  network_full: { label: 'No free address', tone: 'problem', hint: 'There is no free address left in this network. Type an address in another network.' },
  already_on_adapter: { label: 'Already on adapter', tone: 'info', hint: 'ScadaBench will use this existing address and will not remove it later.' },
  used_by_other_adapter: { label: 'Used by another adapter', tone: 'problem', hint: 'Another adapter of this PC already has this address. Pick another address.' },
  duplicate: { label: 'Used twice', tone: 'problem', hint: 'Two IEDs would get the same address. Give each IED its own address.' },
  in_use_on_network: { label: 'Taken on the network', tone: 'problem', hint: 'Another device already answers on this address. Pick a different one.' },
  invalid_manual: { label: 'Address not valid', tone: 'problem', hint: 'Type an address like 192.168.1.21 with a prefix from 8 to 30.' },
  ok: { label: 'Free', tone: 'good', hint: 'No conflict found; this address will be added.' },
};

export const LIVE_STATES: Record<string, Status> = {
  starting: { label: 'Starting', tone: 'waiting', hint: 'Preparing the connection.' },
  connecting: { label: 'Connecting', tone: 'waiting', hint: 'Opening a connection to the device on port 102.' },
  discovering: { label: 'Reading the model', tone: 'waiting', hint: 'Reading the device data model to compare it with the SCD. Large IEDs take a while.' },
  live: { label: 'Live', tone: 'good', hint: 'Values are arriving from the device.' },
  retrying: { label: 'Trying again', tone: 'warning', hint: 'The device did not answer. Check the cable, the address and that port 102 is open.' },
  no_address: { label: 'No address', tone: 'problem', hint: 'Type the device address in the connection card.' },
  stopped: { label: 'Not connected', tone: 'off', hint: 'Press Run to connect.' },
};

export const gooseStatus = (state: string, problem = ''): Status =>
  problem && PROBLEMS[problem] ? PROBLEMS[problem] : GOOSE_STATES[state] ?? GOOSE_STATES.waiting;
export const liveStatus = (state: string): Status => LIVE_STATES[state] ?? LIVE_STATES.starting;
/** A typed address the page is still checking: shown at once, never as "Free" before it is checked. */
const CHECKING_ADDRESS: Status = { label: 'Checking…', tone: 'waiting', hint: 'ScadaBench is checking this address.' };
export const addressStatus = (status: string): Status =>
  status === 'checking' ? CHECKING_ADDRESS : PROBLEMS[status] ?? PROBLEMS.invalid_manual;

export const UI_STATUS: Record<string, Status> = {
  equipment_ready: { label: 'Ready', tone: 'good', hint: 'Connected to its IED signals.' },
  adapter_up: { label: 'Connected', tone: 'good', hint: 'The cable or virtual link is up.' },
  adapter_down: { label: 'No cable', tone: 'warning', hint: 'Plug the device or switch into this port. This list updates by itself.' },
  adapter_internet: { label: 'Internet connection', tone: 'info', hint: 'This adapter carries your internet connection.' },
  item_keep: { label: 'Already there', tone: 'info', hint: 'ScadaBench uses it as it is and will never remove it.' },
  item_add: { label: 'Will be added', tone: 'waiting', hint: 'Added when you press Apply; removed by Clean up.' },
  item_remove: { label: 'Will be removed', tone: 'waiting', hint: 'ScadaBench added this earlier and removes only this.' },
  item_restore: { label: 'Will be switched on again', tone: 'waiting', hint: 'ScadaBench switched this Windows rule off; Clean up puts it back as it was.' },
  result_done: { label: 'Done', tone: 'good', hint: '' },
  result_failed: { label: 'Failed', tone: 'problem', hint: 'Windows refused this change.' },
  mode_loopback: { label: 'This PC only', tone: 'good', hint: 'Private addresses; nothing in Windows is changed.' },
  mode_lab: { label: 'Lab network', tone: 'info', hint: 'Simulated IEDs are reachable on the selected adapter.' },
  npcap_ready: { label: 'Npcap installed', tone: 'good', hint: 'GOOSE can be shared with the network and real devices can be heard.' },
  npcap_missing: { label: 'Npcap missing', tone: 'warning', hint: 'GOOSE works inside ScadaBench. Install the free Npcap driver to share it with the network or hear real devices.' },
  goose_on: { label: 'Sends GOOSE', tone: 'good', hint: 'Publishes its GOOSE blocks while running, inside ScadaBench and, when shared, on the network.' },
  goose_off: { label: 'GOOSE off', tone: 'off', hint: 'This IED does not send GOOSE.' },
  sv_needs_lab: { label: 'Lab network only', tone: 'off', hint: 'Sampled Values are received only after you set up the lab network.' },
  goose_none: { label: 'No GOOSE in SCD', tone: 'off', hint: 'The SCD defines no publishable GOOSE block for this IED.' },
  role_real: { label: 'Monitors real device', tone: 'info', hint: 'ScadaBench reads the real IED on your network and shows its live values.' },
  role_off: { label: 'Skipped', tone: 'off', hint: 'Kept in the project, but not started when you press Run.' },
  stale: { label: 'Old value', tone: 'warning', hint: 'The connection was lost; this value may have changed since.' },
};

/** One short status for a device in the Devices list. */
export function roleStatus(role: string, state: string): Status {
  if (role === 'off') return UI_STATUS.role_off;
  if (role === 'real') {
    if (state === 'real_live') return { label: 'Live', tone: 'good', hint: 'Reading live values from the real device.' };
    if (state.startsWith('real_')) return { label: 'Connecting', tone: 'waiting', hint: 'ScadaBench is reaching the device.' };
    return UI_STATUS.role_real;
  }
  if (state === 'running') return { label: 'Publishing', tone: 'good', hint: 'ScadaBench is this IED now: it answers on its address and sends its reports and GOOSE.' };
  if (state === 'failed') return { label: 'Stopped by error', tone: 'problem', hint: 'See Activity for the reason.' };
  return { label: 'Simulate · stopped', tone: 'off', hint: 'ScadaBench publishes as this IED once it runs: press Run, or "Run this IED" on its page.' };
}

export const needsStatus = (missing: string[]): Status => ({ label: `Needs: ${missing.join(', ').toLowerCase()}`,
  tone: 'warning', hint: 'Connect it to an IED signal below.' });
