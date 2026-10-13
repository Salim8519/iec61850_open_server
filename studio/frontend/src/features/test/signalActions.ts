// The only module that changes simulated values (UI-06). Every screen uses these.
import { api } from '../../api/client';
import type { SourceSpec } from '../../api/types';

export type Point = { ied_id: string; ref: string };

export const setValue = (point: Point, value: unknown) =>
  api('/signals/update', 'POST', { ...point, value });

export const pulse = (point: Point, activeValue: boolean, durationMs: number) =>
  api('/signals/pulse', 'POST', { ...point, active_value: activeValue, duration_ms: durationMs });

export const resume = (point: Point) => api('/signals/release-override', 'POST', point);

export const configureSource = (point: Point, spec: Partial<SourceSpec> & Record<string, unknown>) =>
  api('/signals/source', 'POST', { ...point, spec });
