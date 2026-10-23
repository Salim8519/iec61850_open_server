import { useState, useSyncExternalStore } from 'react';
import { setPreferences } from '../../api/preferences';
import type { DisplaySetting } from '../../app/displayFit';
import { fitSnapshot, setDisplaySetting, subscribeFit } from '../../app/installFit';
import { DISPLAY_COPY as COPY, displayNow } from '../../copy/display';
import { SegmentedControl } from '../../ui/SegmentedControl';

const CHOICES = { fit: 0, full: 1 } as const;
type Choice = keyof typeof CHOICES;

/** Settings › Display: fit to window or always 100%, with the zoom in use and the window size. */
export function DisplaySection() {
  const state = useSyncExternalStore(subscribeFit, fitSnapshot);
  const [problem, setProblem] = useState('');
  const choose = async (choice: Choice) => {
    const before = state.setting;
    const next: DisplaySetting = CHOICES[choice];
    setProblem('');
    setDisplaySetting(next);
    // The frozen Python engine answers "Unknown preference": put the choice back and say so.
    try { await setPreferences({ display_size: next }); } catch { setDisplaySetting(before); setProblem(COPY.notSaved); }
  };
  return <div className="card">
    <div className="card-heading"><h2>{COPY.heading}</h2></div>
    <div className="card-body">
      <div className="row"><SegmentedControl label={COPY.label} value={state.setting === 1 ? 'full' : 'fit'} onChange={choose}
        options={[{ value: 'fit', label: COPY.fit, title: COPY.fitTitle }, { value: 'full', label: COPY.full, title: COPY.fullTitle }]} /></div>
      <p className="small" style={{ margin: 0 }}><strong>{displayNow(state.fit, state.width, state.height)}</strong></p>
      <p className="small muted settings-note" style={{ margin: 0 }}>{state.supported ? COPY.hint : COPY.oldBrowser}</p>
      {problem && <p className="small muted settings-note" role="status" style={{ margin: 0 }}>{problem}</p>}
    </div>
  </div>;
}
