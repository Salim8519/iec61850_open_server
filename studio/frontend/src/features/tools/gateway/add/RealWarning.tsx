import { AlertTriangle } from 'lucide-react';
import { ADD_COPY } from '../../../../copy/gatewayAdd';

/** The amber line before points of a real device are added, with the "I understand" tick that unlocks Add. */
export function RealWarning({ understood, change }: { understood: boolean; change: (value: boolean) => void }) {
  return <div className="confirm-box add-real" role="note">
    <p><AlertTriangle size={18} aria-hidden /><strong>{ADD_COPY.realLine}</strong></p>
    <label className="row add-check"><input type="checkbox" checked={understood} onChange={event => change(event.target.checked)} />
      {ADD_COPY.understand}</label>
  </div>;
}
