import type { CommandOptions } from '../../../api/tools';
import { COMMAND_COPY, ORIGINATORS } from '../../../copy/commands';
import { Disclosure } from '../../../ui/Disclosure';

export const DEFAULT_OPTIONS: CommandOptions = { test: false, interlock_check: false, synchro_check: false, orcat: 3, orident: '' };

/** The rarely needed command options, folded away (ADR-0104). */
export function CommandOptionsBox({ options, change }: { options: CommandOptions; change: (next: CommandOptions) => void }) {
  const flag = (key: 'test' | 'interlock_check' | 'synchro_check', text: string) =>
    <label className="command-option"><input type="checkbox" checked={options[key]}
      onChange={event => change({ ...options, [key]: event.target.checked })} />{text}</label>;
  return <Disclosure title={COMMAND_COPY.optionsTitle}>
    <div className="command-options">
      <p className="small muted">{COMMAND_COPY.optionsHint}</p>
      {flag('test', COMMAND_COPY.test)}
      {flag('interlock_check', COMMAND_COPY.interlock)}
      {flag('synchro_check', COMMAND_COPY.synchro)}
      <label className="command-field">{COMMAND_COPY.originator}
        <select value={options.orcat} onChange={event => change({ ...options, orcat: Number(event.target.value) })}>
          {ORIGINATORS.map(item => <option key={item.value} value={item.value}>{item.label}</option>)}
        </select></label>
      <label className="command-field">{COMMAND_COPY.originatorName}
        <input value={options.orident} maxLength={64} placeholder="ScadaBench" title={COMMAND_COPY.originatorNameHint}
          onChange={event => change({ ...options, orident: event.target.value })} /></label>
    </div>
  </Disclosure>;
}
