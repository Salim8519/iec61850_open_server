import { ENERGY_LOOK, ENERGY_ORDER, wireStyle, type Energization } from '../../equipment/energization';

/** Words for every line style on the canvas while running (never colour alone). */
export function EnergizationLegend({ energy }: { energy: Energization }) {
  const shown = new Set(Object.values(energy.edges));
  if (!shown.size) return null;
  return <div className="energy-legend" role="note" aria-label="What the wire styles mean">
    {ENERGY_ORDER.filter(state => shown.has(state)).map(state => <div key={state} className="energy-row"
      title={ENERGY_LOOK[state].meaning}>
      <svg width="34" height="10" aria-hidden="true"><line x1="3" y1="5" x2="31" y2="5" style={wireStyle(state)} /></svg>
      <span>{ENERGY_LOOK[state].word}</span></div>)}
  </div>;
}
