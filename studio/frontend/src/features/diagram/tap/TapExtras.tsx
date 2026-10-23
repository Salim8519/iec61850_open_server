import type { AvrSettings, ExtendedControl, Ied, Project, TapChanger } from '../../../api/types';
import { commandReady, MODE_KINDS, tapControls } from '../../../equipment/tapChanger';
import { NUMERIC_KINDS } from '../../../equipment/measurementDisplay';
import { TAP_COPY } from '../../../copy/tapChanger';
import { TapControlField, TapNumber, TapSignalField } from './TapFields';

type Props = { nodeId: string; name: string; tap: TapChanger; project: Project; locked: boolean;
  set: (patch: Partial<TapChanger>) => void; typedOf: (ied: Ied | undefined) => readonly ExtendedControl[] };

const AVR_FIELDS: (keyof AvrSettings)[] = ['target_kv', 'bandwidth_pct', 'delay_s', 'step_pct', 'hv_rated_kv',
  'lv_rated_kv', 'source_kv'];

/** Plain-words limits; the engine checks the same ones. */
function avrProblem(key: keyof AvrSettings, value: number): string | null {
  if (key === 'delay_s') return value < 0 ? 'The delay cannot be negative.' : value > 3600 ? 'At most 3600 s.' : null;
  if (key === 'source_kv') return value < 0 ? 'The source voltage cannot be negative.' : null;
  if (value <= 0) return 'Must be above 0.';
  if (key === 'bandwidth_pct' && value > 50) return 'At most 50 %.';
  if (key === 'step_pct' && value > 20) return 'At most 20 %.';
  return null;
}

/** The optional parts of a tap changer: Auto/Manual, LV voltage and the simulated voltage regulator. */
export function TapExtras({ nodeId, name, tap, project, locked, set, typedOf }: Props) {
  const avr = tap.avr;
  const canRegulate = !!tap.tap_source && commandReady(tap);
  const setAvr = (patch: Partial<AvrSettings>) => set({ avr: { ...avr, ...patch } });
  return <>
    <section className="sheet-section"><h3>{TAP_COPY.modeTitle}</h3>
      <p className="small muted" style={{ marginTop: 0 }}>{TAP_COPY.modeHint}</p>
      <TapSignalField title={TAP_COPY.modeSourceLabel} hint="Not chosen: the simulated regulator's own switch is used."
        ieds={project.ieds} value={tap.mode_source} kinds={MODE_KINDS} fc="ST" memoryKey={`sld-tap-mode-${nodeId}`} near={name}
        disabled={locked} change={binding => set({ mode_source: binding })} />
      <TapControlField label={TAP_COPY.modeControlLabel} ieds={project.ieds} value={tap.mode_control}
        controlsOf={ied => tapControls(ied, typedOf(ied)).mode} empty="This IED has no SPC or ENC control here." disabled={locked}
        change={binding => set({ mode_control: binding })} />
      {(tap.mode_source || tap.mode_control) && <div className="tap-field-grid" style={{ marginTop: 10 }}>
        <TapNumber label={TAP_COPY.autoValueLabel} value={tap.auto_value} whole disabled={locked}
          hint="Only used when the IED keeps the mode as a number (ENS/ENC)." change={value => set({ auto_value: value })} />
        <TapNumber label={TAP_COPY.manualValueLabel} value={tap.manual_value} whole disabled={locked}
          change={value => set({ manual_value: value })} /></div>}
    </section>
    <section className="sheet-section"><h3>{TAP_COPY.voltageTitle}</h3>
      <TapSignalField title="LV voltage" hint={TAP_COPY.voltageHint} ieds={project.ieds} value={tap.voltage_source}
        kinds={NUMERIC_KINDS} fc="MX" memoryKey={`sld-tap-lv-${nodeId}`} near={name} disabled={locked}
        change={binding => set({ voltage_source: binding })} /></section>
    <section className="sheet-section"><h3>{TAP_COPY.avrTitle}</h3>
      <p className="small muted" style={{ marginTop: 0 }}>{TAP_COPY.avrHint}</p>
      <label className="row" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <input type="checkbox" checked={avr.enabled} disabled={locked || (!canRegulate && !avr.enabled)}
          onChange={event => setAvr({ enabled: event.target.checked })} />{TAP_COPY.avrEnable}</label>
      {!canRegulate && <p className="small muted" style={{ margin: '4px 0 0' }}>{TAP_COPY.avrNeeds}</p>}
      {avr.enabled && <div className="tap-field-grid" style={{ marginTop: 10 }}>
        {AVR_FIELDS.map(key => { const [label, unit, hint] = TAP_COPY.fields[key];
          return <TapNumber key={key} label={label} unit={unit} hint={hint} value={avr[key] as number} disabled={locked}
            check={value => avrProblem(key, value)} change={value => setAvr({ [key]: value } as Partial<AvrSettings>)} />; })}
      </div>}
    </section>
  </>;
}
