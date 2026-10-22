import { CONVERTER_COPY } from '../../../../copy/converter.ts';
import { RANGES_COPY } from '../../../../copy/master.ts';

export type ConverterTab = keyof typeof CONVERTER_COPY.tabs | 'ranges';

/** The converter's views. A Modbus device cannot list its points, so it also gets "What to read" (the
 *  master's ranges, read by the same master service) right after the device points (change 0178). */
export function converterTabs(modbus: boolean): [ConverterTab, string][] {
  const tabs = (Object.keys(CONVERTER_COPY.tabs) as (keyof typeof CONVERTER_COPY.tabs)[])
    .map(key => [key, CONVERTER_COPY.tabs[key]] as [ConverterTab, string]);
  return modbus ? [tabs[0], ['ranges', RANGES_COPY.tab], ...tabs.slice(1)] : tabs;
}
