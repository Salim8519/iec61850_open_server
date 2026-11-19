/** Words shown when the window itself fails to draw (fix plan A15). The IEDs run in the engine,
 *  not in this window, so they keep running and reloading the window is safe. */
export const CRASH_COPY = {
  title: 'Something went wrong. Your IEDs are still running.',
  detail: 'This window stopped drawing because of an unexpected error. Reload the window to continue.',
  reload: 'Reload',
};
