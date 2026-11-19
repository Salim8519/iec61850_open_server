/** What the window shows while actions run: busy, the words of the latest action, and an error. */
export type BusyView = { busy: (on: boolean) => void; working: (label: string) => void; error: (message: string) => void };

/** Runs actions that may overlap (fix plan A16). The buttons stay disabled until the last running
 *  action ends, and the first failure among overlapping actions is the one shown. */
export function createBusyCount(view: BusyView) {
  let running = 0;
  let failed = false;
  return async function run(task: () => Promise<unknown>, label: string): Promise<void> {
    if (running === 0) { failed = false; view.error(''); }
    running += 1;
    view.busy(true); view.working(label);
    try { await task(); }
    catch (reason) {
      if (!failed) { failed = true; view.error(reason instanceof Error ? reason.message : String(reason)); }
    } finally {
      running -= 1;
      if (running === 0) { view.busy(false); view.working(''); }
    }
  };
}
