// Data classes (SPS, DPS, BCR, SPC …) of an IED's data objects, from the Go engine's
// GET /ieds/<id>/data-classes (ADR-0095). They refine the point kind: a counter or a double
// command can only be told apart by its class. The lookup is an exact model prefix match on the
// object reference, never a guess from a name. Older engines answer 404: classes stay unknown.
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import { dataClassOf } from './pointKind';

export type DataClasses = Record<string, string>;
type Lookup = (ref: string) => string | undefined;

const cache = new Map<string, Promise<DataClasses>>();

/** The classes of one IED, asked once per IED and shared by every list. */
export function loadDataClasses(iedId: string): Promise<DataClasses> {
  let pending = cache.get(iedId);
  if (!pending) {
    pending = api<{ classes: DataClasses }>(`/ieds/${encodeURIComponent(iedId)}/data-classes`)
      .then(answer => answer.classes ?? {}).catch(() => ({}));
    cache.set(iedId, pending);
  }
  return pending;
}

/** A lookup from attribute reference to data class for one IED (empty until loaded or when unknown). */
export function useDataClasses(iedId: string | null | undefined): Lookup {
  const [classes, setClasses] = useState<DataClasses>({});
  useEffect(() => {
    let live = true;
    setClasses({});
    if (iedId) void loadDataClasses(iedId).then(found => { if (live) setClasses(found); });
    return () => { live = false; };
  }, [iedId]);
  return (ref: string) => dataClassOf(classes, ref);
}

/** Forget cached classes, e.g. after an IED was imported again. */
export const forgetDataClasses = (iedId?: string) => (iedId ? cache.delete(iedId) : cache.clear());
