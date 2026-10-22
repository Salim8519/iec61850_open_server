import { createContext, useContext } from 'react';
import type { TapAction } from './tapCommands';
import type { TapChanger } from '../api/types';
import { relayValueKey } from './stateBindings';
import { useLiveValues } from './pointPanelContext';
import type { AvrState } from './tapChanger';

/** Sends one tap changer command of a node (ADR-0109); the diagram page provides it. */
export const TapCommand = createContext<(nodeId: string, action: TapAction) => void>(() => {});

/** Node id whose tap changer command is in flight, so its buttons wait. */
export const TapBusy = createContext<string | null>(null);

/** The live values of one transformer: tap position, Auto/Manual status, LV voltage and the regulator state. */
export function useTapValues(nodeId: string, tap: TapChanger | undefined) {
  const [tapValue, modeValue, voltageValue, avr] = useLiveValues(
    ['tap', 'mode', 'lv', 'avr'].map(row => relayValueKey(nodeId, row)));
  return { tapValue: tap?.tap_source ? tapValue : undefined, modeValue, voltageValue,
    avr: (avr ?? undefined) as AvrState | undefined };
}

export const useTapCommand = () => useContext(TapCommand);
