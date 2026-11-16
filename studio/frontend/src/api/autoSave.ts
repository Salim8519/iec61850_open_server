import { api } from './client';

/** File › auto save (ADR-0139, Go engine): off keeps changes in the open project until Save. */
export type AutoSaveState = { on: boolean; unsaved: boolean };

export const getAutoSave = () => api<AutoSaveState>('/project/autosave');
/** Turning it on saves the unsaved changes first; the choice is remembered by the engine. */
export const setAutoSave = (on: boolean) => api<AutoSaveState>('/project/autosave', 'PUT', { on });
