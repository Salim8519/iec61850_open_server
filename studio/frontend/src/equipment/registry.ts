import type { EquipmentKind } from '../api/types';

export const equipment: { kind: EquipmentKind; label: string; subtitle: string }[] = [
  { kind: 'CB', label: 'Circuit breaker', subtitle: 'XCBR' },
  { kind: 'DS', label: 'Disconnector', subtitle: 'XSWI' },
  { kind: 'ES', label: 'Earth switch', subtitle: 'XSWI' },
  { kind: 'CT', label: 'Current transformer', subtitle: 'TCTR' },
  { kind: 'VT', label: 'Voltage transformer', subtitle: 'TVTR' },
  { kind: 'SOURCE', label: 'Analog source', subtitle: 'MX measurement' },
  { kind: 'MEASUREMENT_LIST', label: 'Measurement list', subtitle: 'Several values · own scaling and units' },
  { kind: 'RELAY', label: 'Protection relay', subtitle: 'Status panel · initiate signals' },
  { kind: 'TRIP_RELAY', label: 'Trip relay', subtitle: 'SPC · latching' },
  { kind: 'PUSH_BUTTON', label: 'Push button', subtitle: 'Click to press · no wires' },
  { kind: 'POINT_PANEL', label: 'Point panel', subtitle: 'Many points of one IED · set or operate in a popup' },
  { kind: 'JUNCTION', label: 'Junction', subtitle: 'Drawing connection point' },
  { kind: 'INFEED', label: 'Infeed', subtitle: 'Line, transformer or grid feeding the station' },
  { kind: 'TAP_TRANSFORMER', label: 'Tap-changer transformer', subtitle: 'Power transformer · Raise/Lower · Auto/Manual' },
];
