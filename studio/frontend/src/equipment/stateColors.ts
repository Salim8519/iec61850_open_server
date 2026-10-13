import type { EquipmentData, EquipmentKind } from '../api/types';

/** Two-state objects and their user-adjustable colours. Display only; never simulation input. */
export type ColorScheme = { active: string; inactive: string; activeLabel: string; inactiveLabel: string };

const GREEN = '#16a34a';
const RED = '#dc2626';
const WHITE = '#ffffff';

const SCHEMES: Partial<Record<EquipmentKind, ColorScheme>> = {
  CB: { active: GREEN, inactive: WHITE, activeLabel: 'Closed', inactiveLabel: 'Open' },
  DS: { active: GREEN, inactive: WHITE, activeLabel: 'Closed', inactiveLabel: 'Open' },
  ES: { active: GREEN, inactive: WHITE, activeLabel: 'Closed', inactiveLabel: 'Open' },
  TRIP_RELAY: { active: RED, inactive: WHITE, activeLabel: 'Operated', inactiveLabel: 'Reset' },
  PUSH_BUTTON: { active: GREEN, inactive: WHITE, activeLabel: 'Pressed', inactiveLabel: 'Released' },
};

/** Default scheme for a kind, or null when the object has no two-state appearance. */
export function defaultScheme(data: EquipmentData): ColorScheme | null {
  if (data.kind === 'PUSH_BUTTON' && data.push_mode === 'spc_momentary') return null;
  return SCHEMES[data.kind] || null;
}

/** Scheme with the object's own colours applied over the defaults. */
export function colorScheme(data: EquipmentData): ColorScheme | null {
  const base = defaultScheme(data);
  if (!base) return null;
  return { ...base, active: validHex(data.active_color) || base.active,
    inactive: validHex(data.inactive_color) || base.inactive };
}

export function validHex(value?: string | null): string | null {
  return value && /^#[0-9a-fA-F]{6}$/.test(value) ? value : null;
}

export type Fill = 'active' | 'inactive' | 'intermediate' | 'invalid' | null;

/** Which appearance applies for the current confirmed state; null keeps the neutral card. */
export function stateFill(data: EquipmentData, position: string, value: unknown, linked: boolean): Fill {
  if (!linked || !defaultScheme(data)) return null;
  if (['CB', 'DS', 'ES'].includes(data.kind)) {
    if (position === 'Closed') return 'active';
    if (position === 'Open') return 'inactive';
    if (position === 'Intermediate') return 'intermediate';
    return position === 'Invalid' ? 'invalid' : null;
  }
  if (typeof value !== 'boolean') return value === undefined || value === null ? null : 'invalid';
  if (data.kind === 'TRIP_RELAY') {
    const operated = data.trip_state_polarity !== 'operated_false';
    return value === operated ? 'active' : 'inactive';
  }
  return value ? 'active' : 'inactive';
}

function luminance(hex: string): number {
  const channel = (index: number) => {
    const c = parseInt(hex.slice(index, index + 2), 16) / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * channel(1) + 0.7152 * channel(3) + 0.0722 * channel(5);
}

/** Text colour (navy or white) with the higher WCAG contrast on the given fill. */
export function textOn(hex: string): string {
  const fill = luminance(hex);
  const navy = luminance('#112b46');
  const onNavy = (fill + 0.05) / (navy + 0.05);
  const onWhite = 1.05 / (fill + 0.05);
  return onNavy >= onWhite ? '#112b46' : '#ffffff';
}

export type Paint = { fill: string; outline: string; ink: string };

function darken(hex: string, amount: number): string {
  const part = (index: number) => Math.round(parseInt(hex.slice(index, index + 2), 16) * (1 - amount))
    .toString(16).padStart(2, '0');
  return `#${part(1)}${part(3)}${part(5)}`;
}

/** Colours for one painted symbol part: fill, a visible outline (also on white), contrasting ink. */
export function paintFor(hex: string): Paint {
  const outline = luminance(hex) > 0.6 ? '#475569' : darken(hex, 0.35);
  return { fill: hex, outline, ink: textOn(hex) };
}
