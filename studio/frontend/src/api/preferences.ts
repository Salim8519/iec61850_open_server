import { api } from './client';

export type Preferences = {
  tour_seen_version: number; tour_version: number; sld_goose_layer?: number; sld_readers_layer?: number;
};

export const getPreferences = () => api<Preferences>('/preferences');
export const markTourSeen = (version: number) =>
  api<Preferences>('/preferences', 'PUT', { tour_seen_version: version });
export const setPreferences = (changes: Partial<Omit<Preferences, 'tour_version'>>) =>
  api<Preferences>('/preferences', 'PUT', changes);
