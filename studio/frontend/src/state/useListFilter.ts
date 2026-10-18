import { useCallback, useEffect, useRef, useState } from 'react';
import { getPreferences, saveListFilter, type ListFilters } from '../api/preferences';
import { initialValues, rememberable, restore, type FilterSpec, type FilterValues } from '../ui/filterState';

let remembered: Promise<ListFilters> | null = null;
const load = () => remembered ??= getPreferences().then(prefs => prefs.list_filters ?? {}).catch((): ListFilters => ({}));

/** One list's filter values, restored from the user's last choice and saved again after a change. */
export function useListFilter(list: string, spec: FilterSpec) {
  const specRef = useRef(spec);
  specRef.current = spec;
  const [values, setValues] = useState<FilterValues>(() => initialValues(spec));
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const touched = useRef(false);
  useEffect(() => {
    let alive = true;
    load().then(all => {
      if (alive && !touched.current) setValues(current => ({ ...restore(specRef.current, all[list]), text: current.text }));
    });
    return () => { alive = false; };
  }, [list]);
  const pending = useRef<FilterValues | null>(null);
  const flush = useCallback(() => {
    const next = pending.current;
    pending.current = null;
    if (!next) return;
    const stored = rememberable(specRef.current, next);
    load().then(all => {
      if (JSON.stringify(all[list]) === JSON.stringify(stored)) return;
      all[list] = stored;
      saveListFilter(list, stored).catch(() => {});  // display-only: a failed save changes nothing else
    });
  }, [list]);
  const persist = useCallback((next: FilterValues) => {
    pending.current = next;
    clearTimeout(timer.current);
    timer.current = setTimeout(flush, 400);
  }, [flush]);
  const set = useCallback((key: string, value: string | boolean) => {
    touched.current = true;
    setValues(current => {
      const next = { ...current, [key]: value };
      if (key !== 'text') persist(next);
      return next;
    });
  }, [persist]);
  const clear = useCallback(() => {
    touched.current = true;
    const next = initialValues(specRef.current);
    setValues(next);
    persist(next);
  }, [persist]);
  useEffect(() => () => { clearTimeout(timer.current); flush(); }, [flush]);
  return { values, set, clear };
}
