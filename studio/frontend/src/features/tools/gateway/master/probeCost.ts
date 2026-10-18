/** How many requests Find registers sends, mirroring the worker's split rule in
 * native/master/client_modbus.cpp (blocks of 125 registers or 2000 bits; a refused block is
 * halved until 16 or fewer, then read address by address). Only an estimate for the page. */
export const PROBE_BLOCK: Record<string, number> = { coils: 2000, discrete_inputs: 2000, holding_registers: 125, input_registers: 125 };
const SPLIT_BELOW = 16;

/** Requests for one block when every part of it is refused. */
const refusedBlock = (count: number): number => 1 + (count > SPLIT_BELOW
  ? refusedBlock(Math.floor(count / 2)) + refusedBlock(count - Math.floor(count / 2))
  : count > 1 ? count : 0);

export function probeCost(table: string, start: number, end: number) {
  const block = PROBE_BLOCK[table] ?? 125;
  let fewest = 0, most = 0;
  for (let first = start; first <= end; first += block) {
    fewest += 1;
    most += refusedBlock(Math.min(block, end - first + 1));
  }
  return { fewest, most };
}
