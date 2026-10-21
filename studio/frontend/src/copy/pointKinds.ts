// Words and symbols of the point kinds (ADR-0095). Letters carry the meaning; colour only repeats it.
import type { Kind } from '../points/pointKind';

export const KIND_TEXT: Record<Kind, { glyph: string; name: string; means: string }> = {
  SP: { glyph: '●', name: 'Single point', means: 'On/off status' },
  DP: { glyph: '◐', name: 'Double point', means: 'Position: intermediate, open, closed or bad' },
  IS: { glyph: '#', name: 'Integer status', means: 'Whole number or named state' },
  MV: { glyph: '∿', name: 'Measured value', means: 'Analog value such as current or voltage' },
  CT: { glyph: 'Σ', name: 'Counter', means: 'Total that counts up' },
  SC: { glyph: '▸', name: 'Single command', means: 'On/off command' },
  DC: { glyph: '▸▸', name: 'Double command', means: 'Open/close command' },
  SE: { glyph: '▸~', name: 'Set point', means: 'Command that sends a number' },
  CM: { glyph: '▸', name: 'Command', means: 'Command (single or double not known)' },
  CF: { glyph: '⚙', name: 'Setting', means: 'Setting or configuration value' },
  TX: { glyph: 'T', name: 'Text / other', means: 'Name, text, address or object id' },
  AT: { glyph: '·', name: 'Quality / time', means: 'Helper attribute of a point' },
  UN: { glyph: '?', name: 'Unknown type', means: 'Type not known; shown, never guessed' },
};

export const KIND_COPY = {
  all: 'All types', chips: 'Point type', edit: 'Click to change', save: 'Enter to save · Esc to cancel',
  auto: 'auto', autoHint: (how: string) => `Changes by itself (${how}). Pick a fixed value to type one.`,
  hideHelpers: 'Hide quality / time', hideHelpersHint: 'Hide the q and t helper attributes, show only the points',
};
