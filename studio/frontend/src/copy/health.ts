// Station health wording: every verdict and area has text, an icon (via tone) and a fix hint.
import type { Status } from './status';

export type Verdict = 'ok' | 'warning' | 'problem' | 'not_checked';

export const VERDICTS: Record<Verdict, Status> = {
  ok: { label: 'Matches the SCD', tone: 'good', hint: 'Everything Studio checked here agrees with the SCD file.' },
  warning: { label: 'Check this', tone: 'warning', hint: 'Not wrong for sure, but worth a look: for example test mode, a switched-off GOOSE block or traffic not seen yet.' },
  problem: { label: 'Differs from the SCD', tone: 'problem', hint: 'The real station does not match the SCD here. Load the matching configuration into the IED or get the current SCD.' },
  not_checked: { label: 'Not checked', tone: 'off', hint: 'Studio could not check this yet; the reason is shown next to it. It is never counted as OK.' },
};

export const AREAS: Record<string, { label: string; hint: string }> = {
  role: { label: 'Role', hint: 'IEDs set to Off are not checked.' },
  connection: { label: 'Connection', hint: 'Whether Studio can talk to the device on port 102.' },
  model: { label: 'Data model', hint: 'Every data point in the SCD should exist on the device with the same type.' },
  datasets: { label: 'Datasets', hint: 'Datasets group the values that reports and GOOSE send.' },
  reports: { label: 'Report blocks', hint: 'Report blocks send values to clients such as SCADA.' },
  report_settings: { label: 'Report settings', hint: 'The dataset and configuration revision of each report block.' },
  goose_device: { label: 'GOOSE block on device', hint: 'GOOSE ID, dataset, revision and on/off of each GOOSE block, read from the device.' },
  goose_wire: { label: 'GOOSE on the network', hint: 'What the listener actually received on the lab adapter.' },
  mode: { label: 'Mode', hint: 'Whether each logical device is on, blocked, in test or off, and whether the IED is in simulation.' },
};

export const verdictStatus = (verdict: string): Status => VERDICTS[verdict as Verdict] ?? VERDICTS.not_checked;
export const areaInfo = (area: string) => AREAS[area] ?? { label: area, hint: 'A station health check.' };
