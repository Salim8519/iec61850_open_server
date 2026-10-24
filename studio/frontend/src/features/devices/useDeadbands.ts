import { useEffect, useState } from 'react';
import { api } from '../../api/client';

/** One measurement's SCD deadband as the engine describes it (ADR-0118). */
export type DeadbandPoint = {
  ref: string; object: string; db: number; zero_db?: number; range_min?: number; range_max?: number;
  band?: number; zero_band?: number; db_writable: boolean; zero_db_writable: boolean; applies: boolean; note: string;
};
export type Deadbands = { enabled: boolean; points: DeadbandPoint[]; applied: number; without_range: number };

/** The IED's deadbands, read again when the project changes; undefined while loading or on an older engine. */
export function useDeadbands(iedId: string, revision: number) {
  const [result, setResult] = useState<Deadbands | undefined>();
  useEffect(() => {
    let cancelled = false;
    api<Deadbands>(`/ieds/${iedId}/deadbands`).then(answer => { if (!cancelled) setResult(answer); })
      .catch(() => { if (!cancelled) setResult(undefined); });
    return () => { cancelled = true; };
  }, [iedId, revision]);
  return result;
}
