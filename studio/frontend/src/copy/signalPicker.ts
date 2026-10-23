// Words of the smart point picker (ADR-0113): plain English, one hint per control.
import type { Kind } from '../points/pointKind';

/** What a list of each kind is called in a sentence ("showing single points"). */
const MANY: Partial<Record<Kind, string>> = {
  SP: 'single points (on/off)', DP: 'double points (open/closed)', IS: 'integer status points',
  MV: 'measured values', CT: 'counters', SC: 'single commands', DC: 'double commands', SE: 'set point commands',
};

export const PICKER_COPY = {
  search: 'Search name, number or description…',
  searchLabel: 'Search signals by name, number or description',
  types: 'Point type',
  typesHint: 'Each point has one type, shown by its badge. Pick a type to list only those points; the number says how many there are.',
  allTypes: 'All types',
  objects: 'Object (logical node)',
  objectsHint: 'The part of the IED a point belongs to, such as XCBR1 (a breaker) or GGIO1 (general inputs). Pick one to list only its points.',
  allObjects: 'All objects',
  helpers: 'Show quality and time',
  helpersHint: 'Every point also has a quality (q) and a time (t) attribute. They are hidden because you rarely connect them.',
  suggested: (name: string) => `Suggested for ${name}`,
  suggestedHint: 'These points mention this name in the IED\'s own description or name. Nothing is guessed.',
  mentions: (name: string, where: string) =>
    where === 'description' ? `Mentions ${name} in its description` : `Mentions ${name} in its name`,
  /** Why a type is chosen for the user: the slot's own words, then what to do to see more. */
  why: (kind: Kind, slot?: string) => {
    const what = MANY[kind] ?? 'points of this type';
    const start = slot ? `${slot}: showing ${what}.` : `Showing ${what}, the type this connection takes.`;
    return `${start} Pick ${PICKER_COPY.allTypes} to see every point.`;
  },
  none: 'No signals match.',
  noneOfKind: (kind: Kind) => `No ${MANY[kind] ?? 'points of this type'} match. Pick another type above or ${PICKER_COPY.allTypes}.`,
};
