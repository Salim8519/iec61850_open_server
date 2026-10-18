import { api } from './client';

export type ListFilters = Record<string, Record<string, string | boolean>>;
export type Preferences = { tour_seen_version: number; tour_version: number; list_filters: ListFilters };

export const getPreferences = () => api<Preferences>('/preferences');
export const markTourSeen = (version: number) =>
  api<Preferences>('/preferences', 'PUT', { tour_seen_version: version });
/** Remember one list's filter chips in the data folder (kept by the backend: the port can change). */
export const saveListFilter = (list: string, values: Record<string, string | boolean>) =>
  api<Preferences>('/preferences', 'PUT', { list_filters: { [list]: values } });
