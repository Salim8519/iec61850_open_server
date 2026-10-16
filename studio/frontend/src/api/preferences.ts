import { api } from './client';

export type Preferences = { tour_seen_version: number; tour_version: number };

export const getPreferences = () => api<Preferences>('/preferences');
export const markTourSeen = (version: number) =>
  api<Preferences>('/preferences', 'PUT', { tour_seen_version: version });
