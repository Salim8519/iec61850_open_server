// The search guides in the order the guides page lists them.
import type { HowTo } from './types';
import { SIMULATE_IED } from './simulate-ied';
import { GOOSE_TEST } from './goose-test';
import { SCADA_TEST } from './scada-test';

export const HOWTOS: HowTo[] = [SIMULATE_IED, GOOSE_TEST, SCADA_TEST];
export { guidePath, guideSchema } from './types';
export type { HowTo } from './types';
