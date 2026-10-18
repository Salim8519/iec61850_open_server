import { Eye, Radio } from 'lucide-react';
import { LAYER_COPY } from '../../../copy/signalLayers';
import type { Layers } from '../../../equipment/signalLayerStore';

/** Two remembered on/off switches in the SLD toolbar. They change the view only, never the project. */
export function SignalLayerSwitches({ layers, toggle }: { layers: Layers; toggle: (layer: keyof Layers) => void }) {
  const item = (layer: keyof Layers, Icon: typeof Radio) => <button key={layer} className={`ghost layer-switch${layers[layer] ? ' on' : ''}`}
    role="switch" aria-checked={layers[layer]} title={LAYER_COPY[layer].hint} onClick={() => toggle(layer)}>
    <Icon size={17} aria-hidden />{LAYER_COPY[layer].label}<span className="layer-state">{layers[layer] ? 'On' : 'Off'}</span>
  </button>;
  return <>{item('goose', Radio)}{item('readers', Eye)}</>;
}
