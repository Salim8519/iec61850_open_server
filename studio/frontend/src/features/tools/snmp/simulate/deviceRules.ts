// The device rules the engine checks on every save (service/check.go), mirrored in plain words so a wrong
// value shows its red line at once and is never sent. The engine still has the last word: anything not
// listed here (for example another device on the same address) comes back as its own message.
import type { SnmpDevice } from '../../../../api/snmp';

/** The device fields the Settings and Traps tabs edit; everything else (values) is never sent from here. */
export const SETTING_KEYS = ['name', 'ip', 'port', 'v1', 'v2c', 'v3', 'read_community', 'write_community', 'users',
  'engine_id', 'traps'] as const;
export type DeviceSettings = Pick<SnmpDevice, typeof SETTING_KEYS[number]>;

/** `field` tells the page which field or section shows the line. */
export type DeviceProblem = { field: 'name' | 'ip' | 'port' | 'versions' | 'community' | 'users' | 'engine' | 'traps'; text: string };

const OCTET = '(?:0|[1-9]\\d{0,2})';
const IPV4 = new RegExp(`^${OCTET}\\.${OCTET}\\.${OCTET}\\.${OCTET}$`);

/** A dotted IPv4 address the way the engine reads it (no leading zeros, each part 0 to 255). */
export const isIpv4 = (text: string) => IPV4.test(text) && text.split('.').every(part => Number(part) <= 255);
const inRange = (port: number) => Number.isInteger(port) && port >= 1 && port <= 65535;

export function pickSettings(device: SnmpDevice): DeviceSettings {
  return Object.fromEntries(SETTING_KEYS.map(key => [key, device[key]])) as DeviceSettings;
}

/** The first thing wrong with these settings, or null when they can be saved. */
export function deviceProblem(settings: DeviceSettings): DeviceProblem | null {
  const name = settings.name.trim();
  if (name === '' || name.length > 60) return { field: 'name', text: 'Give the device a name of 1 to 60 characters.' };
  if (!isIpv4(settings.ip)) return { field: 'ip', text: `"${settings.ip}" is not an IPv4 address such as 127.60.0.1.` };
  if (!inRange(settings.port)) return { field: 'port', text: 'The port must be a number from 1 to 65535 (SNMP uses 161).' };
  if (!settings.v1 && !settings.v2c && !settings.v3) return { field: 'versions', text: 'Turn on at least one SNMP version.' };
  if ((settings.v1 || settings.v2c) && settings.read_community === '') {
    return { field: 'community', text: 'v1 and v2c need a read community, for example public.' };
  }
  return usersProblem(settings) ?? engineProblem(settings) ?? trapsProblem(settings);
}

function usersProblem(settings: DeviceSettings): DeviceProblem | null {
  const fail = (text: string): DeviceProblem => ({ field: 'users', text });
  if (settings.v3 && settings.users.length === 0) return fail('Add a v3 user, or turn v3 off.');
  const seen = new Set<string>();
  for (const user of settings.users) {
    if (user.name === '' || user.name.length > 32) return fail('A v3 user needs a name of 1 to 32 characters.');
    if (seen.has(user.name)) return fail(`The user name ${user.name} is used twice.`);
    seen.add(user.name);
    if (user.level !== 'noAuthNoPriv' && (user.auth_password ?? '').length < 8) {
      return fail(`User ${user.name}: the authentication password needs at least 8 characters.`);
    }
    if (user.level === 'authPriv' && (user.priv_password ?? '').length < 8) {
      return fail(`User ${user.name}: the encryption password needs at least 8 characters.`);
    }
  }
  return null;
}

function engineProblem(settings: DeviceSettings): DeviceProblem | null {
  const hex = /^(?:[0-9a-fA-F]{2}){5,32}$/.test(settings.engine_id);
  return hex ? null : { field: 'engine', text: 'The engine ID must be 5 to 32 bytes written in hex, for example 800000090300001122334455.' };
}

/** The trap receivers of the Traps tab. */
export function trapsProblem(settings: DeviceSettings): DeviceProblem | null {
  const fail = (text: string): DeviceProblem => ({ field: 'traps', text });
  const names = new Set(settings.users.map(user => user.name));
  for (const target of settings.traps) {
    if (!isIpv4(target.host)) return fail(`The receiver address "${target.host}" is not an IPv4 address such as 127.0.0.1.`);
    if (!inRange(target.port)) return fail('The receiver port must be a number from 1 to 65535 (usually 162).');
    if (target.version === 'v3' && !names.has(target.user ?? '')) return fail(`The receiver ${target.host} needs one of the device's v3 users.`);
    if (target.version === 'v1' && target.inform) return fail('SNMPv1 has no informs. Choose v2c or v3 for this receiver.');
    if (target.version !== 'v3' && !target.community) return fail(`The receiver ${target.host} needs a community, for example public.`);
  }
  return null;
}
