import { SlidersHorizontal } from 'lucide-react';
import type { CommandOptions } from '../../api/types';
import { changedCount, MAX_ORIDENT } from '../../equipment/commandOptions';
import { SBO_COPY } from '../../copy/selectOperate';
import './sbo/sbo.css';
import { setSboSteps, useSboSteps, useSboStepsError } from './sbo/useSboSteps';

/** The folded "Command options" of the equipment sheet (ADR-0108): test, interlock check, synchro
 *  check and the originator. Closed by default; the title says how many differ from the defaults. */
export function CommandOptionsPanel({ options, change, disabled }: {
  options: CommandOptions; change: (next: CommandOptions) => void; disabled: boolean;
}) {
  const changed = changedCount(options);
  const steps = useSboSteps();
  const stepsError = useSboStepsError();
  const flag = (key: 'test' | 'interlock_check' | 'synchro_check', label: string, hint: string) =>
    <label title={hint}><input type="checkbox" checked={options[key]} disabled={disabled}
      onChange={event => change({ ...options, [key]: event.target.checked })} />{label}</label>;
  return <details className="command-options">
    <summary title={SBO_COPY.optionsHint}><SlidersHorizontal size={14} aria-hidden="true" />{SBO_COPY.optionsTitle}
      {changed > 0 && <span className="badge">{changed} on</span>}</summary>
    <div className="command-options-grid">
      <p className="small muted" style={{ margin: 0 }}>{SBO_COPY.optionsHint}</p>
      {flag('test', SBO_COPY.test, SBO_COPY.testHint)}
      {flag('interlock_check', SBO_COPY.interlock, SBO_COPY.interlockHint)}
      {flag('synchro_check', SBO_COPY.synchro, SBO_COPY.synchroHint)}
      <label title={SBO_COPY.stepsHint}><input type="checkbox" checked={steps} disabled={disabled}
        onChange={event => setSboSteps(event.target.checked)} />{SBO_COPY.stepsLabel}</label>
      {stepsError && <p className="small sbo-note" role="alert" style={{ margin: 0 }}>{SBO_COPY.stepsNotSaved(stepsError)}</p>}
      <label className="field">{SBO_COPY.orcat}
        <select value={options.orcat} disabled={disabled} onChange={event => change({ ...options, orcat: Number(event.target.value) })}>
          {SBO_COPY.orcats.map((name, value) => <option key={value} value={value}>{name}</option>)}
        </select><small>{SBO_COPY.orcatHint}</small></label>
      <label className="field">{SBO_COPY.orident}
        <input type="text" value={options.orident} maxLength={MAX_ORIDENT} disabled={disabled}
          onChange={event => change({ ...options, orident: event.target.value })} /><small>{SBO_COPY.oridentHint}</small></label>
      {changed > 0 && <p className="small" role="status" style={{ margin: 0 }}>{SBO_COPY.optionsChanged}</p>}
    </div>
  </details>;
}
