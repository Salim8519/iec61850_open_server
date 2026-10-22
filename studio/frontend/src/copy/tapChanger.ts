/** Plain-English words of the tap-changer transformer (ADR-0109). Beginner first: every setting says
 *  what it does and what to choose; the IEC names follow in brackets. */
export const TAP_COPY = {
  intro: 'A power transformer with an on-load tap changer. The tap changer moves the voltage of the LV side up or ' +
    'down in small steps while power flows. Raise means more turns on the LV side, so a higher LV voltage; ' +
    'Lower gives a lower LV voltage.',
  tapTitle: 'Tap position comes from',
  tapHint: 'Choose the whole number that tells which tap the transformer is on: valWTr.posVal of a BSC ' +
    '(tap changer control), or stVal of an INS, INC or ISC.',
  commandTitle: 'Raise and lower go to',
  commandHint: 'How the IED takes the raise and lower command. Most tap changer IEDs use one step control (BSC). ' +
    'Some use two separate single commands, one to raise and one to lower.',
  modes: { none: 'No command', step: 'One step control (BSC or ISC)', spc_pair: 'Two single commands (raise and lower)' },
  stepLabel: 'Step control (BSC or ISC)',
  stepMissing: 'This IED lists no BSC or ISC controls here yet. Use two single commands for now.',
  raiseLabel: 'Raise command (SPC)',
  lowerLabel: 'Lower command (SPC)',
  modeTitle: 'Auto / Manual (optional)',
  modeHint: 'In Auto the voltage regulator moves the tap by itself; in Manual only you do. Leave these empty to ' +
    'use the simulated regulator’s own Auto/Manual switch.',
  modeSourceLabel: 'Auto/Manual status (on = Auto, or a number)',
  modeControlLabel: 'Auto/Manual command (SPC or ENC)',
  autoValueLabel: 'Number that means Auto',
  manualValueLabel: 'Number that means Manual',
  voltageTitle: 'LV voltage (optional)',
  voltageHint: 'A measured voltage on the LV side, such as mag.f of an MMXU phase voltage. With the simulated ' +
    'regulator on, ScadaBench writes the calculated LV voltage here.',
  rangeTitle: 'Tap range',
  minLabel: 'Lowest tap', maxLabel: 'Highest tap', nominalLabel: 'Nominal tap',
  nominalHint: 'The nominal tap gives the rated ratio, for example 132 kV to 11 kV.',
  avrTitle: 'Simulated voltage regulator (90)',
  avrHint: 'Only for simulated IEDs: ScadaBench calculates the LV voltage from the tap and, in Auto, sends one ' +
    'raise or lower when the voltage stays outside the band longer than the time delay. It never commands a real IED.',
  avrEnable: 'Simulate the voltage regulator',
  avrNeeds: 'Choose the tap position and the raise/lower command first.',
  fields: {
    target_kv: ['Target LV voltage', 'kV', 'The voltage the regulator holds on the LV side.'],
    bandwidth_pct: ['Band', '± %', 'No action while the voltage is within this percent of the target.'],
    delay_s: ['Time delay', 's', 'How long the voltage must stay outside the band before one step.'],
    step_pct: ['Voltage step per tap', '%', 'How much one tap changes the LV voltage.'],
    hv_rated_kv: ['Rated HV voltage', 'kV', 'Nameplate voltage of the HV side.'],
    lv_rated_kv: ['Rated LV voltage', 'kV', 'Nameplate voltage of the LV side at the nominal tap.'],
    source_kv: ['Source (HV) voltage', 'kV', 'The voltage feeding the transformer in the simulation. Set it higher ' +
      'or lower than rated to watch the regulator react.'],
  } as Record<string, [string, string, string]>,
  operateTitle: 'Tap changer',
  raise: 'Raise', lower: 'Lower', auto: 'Auto', manual: 'Manual',
  raiseHint: 'One tap up: higher LV voltage', lowerHint: 'One tap down: lower LV voltage',
  inAuto: 'In Auto the regulator moves the tap. Switch to Manual to raise or lower yourself.',
  noCommand: 'Choose the raise and lower command in the properties to operate the tap changer.',
  noAuto: 'Auto needs the simulated voltage regulator or an Auto/Manual command. Set one up in the properties ' +
    'while the IEDs are stopped.',
  notMapped: 'Tap not mapped',
  awaiting: 'Awaiting tap',
};

/** What the simulated regulator is doing now, in words. */
export const AVR_STATE: Record<string, string> = {
  in_band: 'Voltage in band',
  timing: 'Out of band, next step in',
  sending: 'Sending a step',
  at_limit: 'At the tap limit',
  no_position: 'Waiting for the tap position',
  manual: 'Manual',
  blocked: 'Regulator stopped',
};

/** Why a regulator is stopped. */
export const AVR_BLOCKED: Record<string, string> = {
  real_ied: 'one of its IEDs is a real device; the regulator never commands real IEDs',
  not_running: 'one of its IEDs is not running',
};
