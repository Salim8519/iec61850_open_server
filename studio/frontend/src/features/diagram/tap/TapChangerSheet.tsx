import type { EquipmentData, Project, TapChanger, TapCommandMode } from '../../../api/types';
import { commandReady, newTapChanger, rangeProblem, TAP_KINDS, tapControls } from '../../../equipment/tapChanger';
import { TAP_COPY } from '../../../copy/tapChanger';
import { TapControlField, TapNumber, TapSignalField } from './TapFields';
import { TapExtras } from './TapExtras';
import '../../../equipment/tap-transformer.css';

type Props = { nodeId: string; data: EquipmentData; project: Project; locked: boolean;
  update: (data: EquipmentData) => void };

/** Sheet sections of a tap-changer transformer (ADR-0109): where the tap position comes from, where raise
 *  and lower go, the tap range, then the optional Auto/Manual, LV voltage and simulated regulator. */
export function TapChangerSheet({ nodeId, data, project, locked, update }: Props) {
  const tap = data.tap_changer ?? newTapChanger();
  const set = (patch: Partial<TapChanger>) => {
    const next = { ...tap, ...patch };
    // The engine refuses a regulator without a tap position or a command; switch it off with them.
    if (next.avr.enabled && (!next.tap_source || !commandReady(next))) next.avr = { ...next.avr, enabled: false };
    update({ ...data, tap_changer: next });
  };
  const chooseMode = (mode: TapCommandMode) =>
    set({ command_mode: mode, step_control: null, raise_control: null, lower_control: null });
  const range = (key: 'min_tap' | 'max_tap' | 'nominal_tap') => (value: number) => rangeProblem({ ...tap, [key]: value });
  return <>
    <section className="sheet-section"><h3>{TAP_COPY.tapTitle}</h3>
      <p className="small muted" style={{ marginTop: 0 }}>{TAP_COPY.intro}</p>
      <TapSignalField title="Tap position" hint={TAP_COPY.tapHint} ieds={project.ieds} value={tap.tap_source}
        kinds={TAP_KINDS} fc="ST" memoryKey={`sld-tap-${nodeId}`} disabled={locked}
        change={binding => set({ tap_source: binding })} /></section>
    <section className="sheet-section"><h3>{TAP_COPY.commandTitle}</h3>
      <p className="small muted" style={{ marginTop: 0 }}>{TAP_COPY.commandHint}</p>
      <div className="stack">
        <label>How the IED is commanded<select disabled={locked} value={tap.command_mode}
          onChange={event => chooseMode(event.target.value as TapCommandMode)}>
          {(Object.keys(TAP_COPY.modes) as TapCommandMode[]).map(mode =>
            <option key={mode} value={mode}>{TAP_COPY.modes[mode]}</option>)}</select></label>
        {tap.command_mode === 'step' && <TapControlField label={TAP_COPY.stepLabel} ieds={project.ieds}
          value={tap.step_control} controlsOf={ied => tapControls(ied).step} empty={TAP_COPY.stepMissing}
          disabled={locked} change={binding => set({ step_control: binding })} />}
        {tap.command_mode === 'spc_pair' && <>
          <TapControlField label={TAP_COPY.raiseLabel} ieds={project.ieds} value={tap.raise_control}
            controlsOf={ied => tapControls(ied).spc} empty="This IED has no single commands (SPC)." disabled={locked}
            change={binding => set({ raise_control: binding })} />
          <TapControlField label={TAP_COPY.lowerLabel} ieds={project.ieds} value={tap.lower_control}
            controlsOf={ied => tapControls(ied).spc.filter(control => control.ref !== tap.raise_control?.ref)}
            empty="Choose a different single command (SPC) than the raise command." disabled={locked}
            change={binding => set({ lower_control: binding })} /></>}
      </div></section>
    <section className="sheet-section"><h3>{TAP_COPY.rangeTitle}</h3>
      <div className="tap-field-grid">
        <TapNumber label={TAP_COPY.minLabel} value={tap.min_tap} whole disabled={locked} check={range('min_tap')}
          change={value => set({ min_tap: value })} />
        <TapNumber label={TAP_COPY.maxLabel} value={tap.max_tap} whole disabled={locked} check={range('max_tap')}
          change={value => set({ max_tap: value })} />
        <TapNumber label={TAP_COPY.nominalLabel} value={tap.nominal_tap} whole disabled={locked}
          check={range('nominal_tap')} hint={TAP_COPY.nominalHint} change={value => set({ nominal_tap: value })} />
      </div></section>
    <TapExtras nodeId={nodeId} tap={tap} project={project} locked={locked} set={set} />
  </>;
}
