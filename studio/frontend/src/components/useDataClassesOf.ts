// Data classes of several IEDs at once, for lists that mix IEDs (Test points). One IED at a time
// is points/dataClasses.ts (useDataClasses); this shares its loader for each listed IED.
// Older engines answer 404: classes stay unknown and the kind falls back to the control list.
import { useEffect, useState } from 'react';
import { loadDataClasses, type DataClasses } from '../points/dataClasses';
import { dataClassOf } from '../points/pointKind';

/** A lookup from (IED id, attribute reference) to data class; empty until the answers arrive. */
export function useDataClassesOf(iedIds: readonly string[]): (iedId: string, ref: string) => string | undefined {
  const key = [...new Set(iedIds)].sort().join('|');
  const [byIed, setByIed] = useState<Record<string, DataClasses>>({});
  useEffect(() => {
    let live = true;
    for (const id of key.split('|').filter(Boolean)) {
      void loadDataClasses(id).then(found => { if (live) setByIed(current => ({ ...current, [id]: found })); });
    }
    return () => { live = false; };
  }, [key]);
  return (iedId, ref) => (byIed[iedId] ? dataClassOf(byIed[iedId], ref) : undefined);
}
