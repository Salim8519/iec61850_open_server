import { getPreferences } from '../api/preferences';
import { fitScale, layoutLimits, type DisplaySetting } from './displayFit';

/** What the Settings › Display line shows: the zoom in use and the real window size. */
export type FitState = { fit: number; width: number; height: number; setting: DisplaySetting; supported: boolean };

let state: FitState = { fit: 1, width: 0, height: 0, setting: 0, supported: false };
let limits: string | null = null;
let frame = 0;
let installed = false;
let chosen = false;
const listeners = new Set<() => void>();

/** CSS zoom with correct measuring (Edge/Chrome 128+); older browsers keep 100%. */
const zoomWorks = () => typeof Element !== 'undefined' && 'currentCSSZoom' in Element.prototype;

/** Measure the real window (CSS zoom on <html> does not change innerWidth or visualViewport) and apply. */
function apply() {
  frame = 0;
  const root = document.documentElement;
  const width = window.innerWidth, height = window.innerHeight;
  const view = window.visualViewport;
  const supported = zoomWorks();
  const fit = supported ? fitScale(view?.width ?? width, view?.height ?? height, state.setting) : 1;
  if (fit !== state.fit) {
    if (fit === 1) root.style.removeProperty('--fit'); else root.style.setProperty('--fit', String(fit));
  }
  const nextLimits = layoutLimits(width, height, fit);
  if (nextLimits !== limits) { limits = nextLimits; root.setAttribute('data-max', nextLimits); }
  if (fit === state.fit && width === state.width && height === state.height && supported === state.supported) return;
  state = { ...state, fit, width, height, supported };
  listeners.forEach(listener => listener());
}

/** One update per frame while the window is being resized. */
const soon = () => { if (!frame) frame = requestAnimationFrame(apply); };

/** Start fit to window; called once in main.tsx before the first render, so nothing jumps. */
export function installFit() {
  if (installed) return;
  installed = true;
  apply();
  window.addEventListener('resize', soon);
  window.visualViewport?.addEventListener('resize', soon);
  // The saved choice comes from the backend's preferences; until then (and on any error) it is fit.
  getPreferences().then(saved => { if (!chosen) choose(saved.display_size === 1 ? 1 : 0); }).catch(() => {});
}

/** Switch between fit to window (0) and always 100% (1); applies at once and wins over a late load. */
export function setDisplaySetting(setting: DisplaySetting) {
  chosen = true;
  choose(setting);
}

function choose(setting: DisplaySetting) {
  if (setting === state.setting) return;
  state = { ...state, setting };
  apply();
  listeners.forEach(listener => listener());
}

/** For useSyncExternalStore in Settings › Display. */
export function subscribeFit(listener: () => void) {
  listeners.add(listener);
  return () => { listeners.delete(listener); };
}
export const fitSnapshot = () => state;
