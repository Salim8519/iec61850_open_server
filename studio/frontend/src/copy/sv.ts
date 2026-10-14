// Sampled Values wording: every state and problem has text, a tone (icon) and a fix hint.
import type { Status } from './status';

export const SV_STATES: Record<string, Status> = {
  stopped: { label: 'Not running', tone: 'off', hint: 'Press Run to start listening for Sampled Values.' },
  waiting: { label: 'Not received yet', tone: 'waiting', hint: 'Studio is listening. If this stays, check that the merging unit runs and is on the same network.' },
  receiving: { label: 'Receiving', tone: 'good', hint: 'Samples from this stream are arriving as the SCD describes.' },
  problem: { label: 'Received with a problem', tone: 'problem', hint: 'Samples arrive but something differs from the SCD. See the reason.' },
  timeout: { label: 'Stopped arriving', tone: 'problem', hint: 'No sample for a second. The merging unit may be off, unplugged or restarting.' },
};

export const SV_PROBLEMS: Record<string, Status> = {
  conf_rev_mismatch: { label: 'Configuration revision differs', tone: 'problem', hint: 'The merging unit runs another configuration than the SCD. Load the matching configuration or get the current SCD.' },
  dataset_mismatch: { label: 'Different dataset size', tone: 'problem', hint: 'Each sample carries a different amount of data than the SCD dataset. Check the dataset of this block.' },
  rate_mismatch: { label: 'Wrong sample rate', tone: 'problem', hint: 'Samples per second differ from the SCD. Check smpRate, the merging unit setting and the nominal frequency (50 or 60 Hz).' },
  not_synchronised: { label: 'Not time synchronised', tone: 'warning', hint: 'The merging unit says its clock is not synchronised (smpSynch 0). Check the PTP or 1PPS time source.' },
  sample_loss: { label: 'Samples lost', tone: 'warning', hint: 'Some sample counters were skipped. Check the network load, cabling and switches.' },
  invalid_quality: { label: 'Invalid quality', tone: 'problem', hint: 'Some values are marked invalid by the merging unit. Check its inputs and status.' },
  simulation: { label: 'Simulation traffic', tone: 'warning', hint: 'This stream is marked as simulation (test equipment). Real protection ignores it unless set to simulation mode.' },
};

export const svStatus = (state: string, problems: string[] = []): Status =>
  problems.length && SV_PROBLEMS[problems[0]] ? SV_PROBLEMS[problems[0]] : SV_STATES[state] ?? SV_STATES.waiting;
